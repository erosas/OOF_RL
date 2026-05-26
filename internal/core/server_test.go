package core_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"OOF_RL/internal/config"
	"OOF_RL/internal/core"
	"OOF_RL/internal/db"
	"OOF_RL/internal/events"
	"OOF_RL/internal/hub"
	"OOF_RL/internal/momentum"
	"OOF_RL/internal/oofevents"
	_ "OOF_RL/internal/plugins/overlayhud"
)

func newTestMux(t *testing.T) (*http.ServeMux, *config.Config) {
	t.Helper()
	tmpDir := t.TempDir()
	database, err := db.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	cfg := config.Defaults()
	cfg.RLInstallPath = ""
	cfg.DevMode = true
	h := hub.New()

	cfgPath := filepath.Join(tmpDir, "config.toml")
	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(bus.Stop)
	srv := core.NewServer(cfgPath, &cfg, database, h, http.NotFoundHandler(), func() {}, nil, bus)
	mux := http.NewServeMux()
	srv.Register(mux)
	return mux, &cfg
}

func newTestServer(t *testing.T) (*core.Server, oofevents.Bus) {
	t.Helper()
	tmpDir := t.TempDir()
	database, err := db.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	cfg := config.Defaults()
	h := hub.New()
	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(bus.Stop)

	srv := core.NewServer(filepath.Join(tmpDir, "config.toml"), &cfg, database, h, http.NotFoundHandler(), func() {}, nil, bus)
	return srv, bus
}

func get(mux *http.ServeMux, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func postJSON(mux *http.ServeMux, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestServerConfigReturnsConfig(t *testing.T) {
	srv, _ := newTestServer(t)
	if srv.Config() == nil {
		t.Fatal("Config() should return a non-nil config")
	}
}

func TestServerLoadPlugins(t *testing.T) {
	srv, _ := newTestServer(t)
	if err := srv.LoadPlugins(); err != nil {
		t.Fatalf("LoadPlugins: %v", err)
	}
	if len(srv.List()) == 0 {
		t.Fatal("LoadPlugins should register at least the imported plugins")
	}
}

func TestServerRegistersMomentumRuntime(t *testing.T) {
	srv, _ := newTestServer(t)

	if srv.Momentum() == nil {
		t.Fatal("Momentum() returned nil")
	}
	var _ momentum.SnapshotProvider = srv.Momentum()
}

func TestServerMomentumRuntimeReceivesTranslatedGameActions(t *testing.T) {
	srv, _ := newTestServer(t)

	srv.DispatchEvent(events.Envelope{
		Event: "StatfeedEvent",
		Data: []byte(`{
			"MatchGuid":"match-1",
			"EventName":"Shot",
			"MainTarget":{"Name":"Alice","PrimaryId":"pid-a","Shortcut":1,"TeamNum":0}
		}`),
	})

	waitForCoreTest(t, func() bool {
		return srv.Momentum().Snapshot().Sequence == 1
	})
	if srv.Momentum().Snapshot().Teams[oofevents.TeamBlue].MomentumInfluence <= 0 {
		t.Fatalf("momentum snapshot not updated: %+v", srv.Momentum().Snapshot())
	}
}

func TestServerShutdownRuntimeStopsMomentumWiring(t *testing.T) {
	srv, _ := newTestServer(t)

	srv.ShutdownRuntime()
	srv.DispatchEvent(events.Envelope{
		Event: "StatfeedEvent",
		Data: []byte(`{
			"MatchGuid":"match-1",
			"EventName":"Shot",
			"MainTarget":{"Name":"Alice","PrimaryId":"pid-a","Shortcut":1,"TeamNum":0}
		}`),
	})
	time.Sleep(20 * time.Millisecond)
	if srv.Momentum().Snapshot().Sequence != 0 {
		t.Fatalf("momentum updated after runtime shutdown: %+v", srv.Momentum().Snapshot())
	}
}

func waitForCoreTest(t *testing.T, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

// --- /api/config ---

func TestGetConfig(t *testing.T) {
	mux, cfg := newTestMux(t)
	cfg.AppPort = 8080

	w := get(mux, "/api/config")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var got map[string]any
	json.Unmarshal(w.Body.Bytes(), &got)
	if _, ok := got["app_port"]; !ok {
		t.Error("expected app_port in response")
	}
	if _, ok := got["storage"]; !ok {
		t.Error("expected storage in response")
	}
}

func TestPostConfig(t *testing.T) {
	mux, _ := newTestMux(t)

	newCfg := config.Defaults()
	newCfg.AppPort = 9999

	w := postJSON(mux, "/api/config", newCfg)
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — body: %s", w.Code, w.Body.String())
	}
	var returned map[string]any
	json.Unmarshal(w.Body.Bytes(), &returned)
	if port, _ := returned["app_port"].(float64); int(port) != 9999 {
		t.Errorf("app_port: got %v, want 9999", returned["app_port"])
	}
}

func TestPostConfigBadJSON(t *testing.T) {
	mux, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestConfigMethodNotAllowed(t *testing.T) {
	mux, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/config", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want 405", w.Code)
	}
}

// --- /api/config/ini ---

func TestGetINIWhenNotFound(t *testing.T) {
	t.Setenv("USERPROFILE", "")
	mux, _ := newTestMux(t)
	w := get(mux, "/api/config/ini")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if errFlag, _ := resp["error"].(bool); !errFlag {
		t.Errorf("expected error:true in response, got: %v", resp)
	}
}

func TestPostINI(t *testing.T) {
	t.Setenv("USERPROFILE", "")
	mux, cfg := newTestMux(t)
	cfg.RLInstallPath = t.TempDir()

	w := postJSON(mux, "/api/config/ini", config.INISettings{PacketSendRate: 60, Port: 49123})
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — body: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status:ok, got %v", resp)
	}
}

func TestPostINIBadJSON(t *testing.T) {
	mux, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodPost, "/api/config/ini", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestINIMethodNotAllowed(t *testing.T) {
	mux, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/config/ini", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want 405", w.Code)
	}
}

// --- /api/nav ---

func TestGetNav(t *testing.T) {
	mux, _ := newTestMux(t)
	w := get(mux, "/api/nav")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var tabs []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &tabs); err != nil {
		t.Fatalf("parse: %v", err)
	}
}

// --- /api/settings/schema ---

func TestGetSettingsSchema(t *testing.T) {
	mux, _ := newTestMux(t)
	w := get(mux, "/api/settings/schema")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var blobs []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &blobs); err != nil {
		t.Fatalf("parse: %v", err)
	}
}

// --- /api/settings ---

func TestPostSettings(t *testing.T) {
	mux, cfg := newTestMux(t)

	w := postJSON(mux, "/api/settings", map[string]string{
		"ballchasing_api_key": "test-key",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — body: %s", w.Code, w.Body.String())
	}
	if cfg.BallchasingAPIKey != "test-key" {
		t.Errorf("bc key: got %q, want test-key", cfg.BallchasingAPIKey)
	}

	newCfg := config.Defaults()
	newCfg.TrackerCacheTTLMinutes = 30
	w = postJSON(mux, "/api/config", newCfg)
	if w.Code != http.StatusOK {
		t.Fatalf("config status: got %d — body: %s", w.Code, w.Body.String())
	}
	if cfg.TrackerCacheTTLMinutes != 30 {
		t.Errorf("tracker TTL: got %d, want 30", cfg.TrackerCacheTTLMinutes)
	}
}

func TestPostSettingsMethodNotAllowed(t *testing.T) {
	mux, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want 405", w.Code)
	}
}

func TestPostSettingsBadJSON(t *testing.T) {
	mux, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodPost, "/api/settings", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

// --- InitPlugins / ShutdownPlugins / Get ---

func TestInitPluginsAndShutdownPlugins(t *testing.T) {
	srv, _ := newTestServer(t)
	if err := srv.LoadPlugins(); err != nil {
		t.Fatalf("LoadPlugins: %v", err)
	}
	if err := srv.InitPlugins(); err != nil {
		t.Fatalf("InitPlugins: %v", err)
	}
	srv.ShutdownPlugins()
}

func TestServerGet(t *testing.T) {
	srv, _ := newTestServer(t)
	if err := srv.LoadPlugins(); err != nil {
		t.Fatalf("LoadPlugins: %v", err)
	}
	if _, ok := srv.Get("overlayhud"); !ok {
		t.Error("Get(overlayhud): want found")
	}
	if _, ok := srv.Get("nonexistent-plugin"); ok {
		t.Error("Get(nonexistent-plugin): want not found")
	}
}

// --- LoadWASMPlugins ---

func TestLoadWASMPlugins_MissingDir(t *testing.T) {
	srv, _ := newTestServer(t)
	if err := srv.LoadWASMPlugins(filepath.Join(t.TempDir(), "nonexistent")); err != nil {
		t.Fatalf("want nil for missing dir, got: %v", err)
	}
}

func TestLoadWASMPlugins_EmptyDir(t *testing.T) {
	srv, _ := newTestServer(t)
	if err := srv.LoadWASMPlugins(t.TempDir()); err != nil {
		t.Fatalf("LoadWASMPlugins empty dir: %v", err)
	}
	if len(srv.List()) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(srv.List()))
	}
}

func TestLoadWASMPlugins_InvalidWasm(t *testing.T) {
	srv, _ := newTestServer(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.wasm"), []byte("not wasm"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Bad WASM is logged and skipped; no error is returned to the caller.
	if err := srv.LoadWASMPlugins(dir); err != nil {
		t.Fatalf("want nil (bad wasm skipped), got: %v", err)
	}
	if len(srv.List()) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(srv.List()))
	}
}

// --- /api/plugins/<id>/view ---

func TestHandlePluginView_MissingID(t *testing.T) {
	mux, _ := newTestMux(t)
	w := get(mux, "/api/plugins/")
	if w.Code != http.StatusBadRequest {
		t.Errorf("missing id: got %d, want 400", w.Code)
	}
}

func TestHandlePluginView_NotFound(t *testing.T) {
	mux, _ := newTestMux(t)
	w := get(mux, "/api/plugins/nonexistent-xyz/view")
	if w.Code != http.StatusNotFound {
		t.Errorf("not found: got %d, want 404", w.Code)
	}
}



// --- /api/data-dir ---

func TestHandleDataDir(t *testing.T) {
	mux, cfg := newTestMux(t)
	w := get(mux, "/api/data-dir")
	if w.Code != http.StatusOK {
		t.Fatalf("handleDataDir: got %d, want 200", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if resp["path"] != cfg.DataDir {
		t.Errorf("path: got %q, want %q", resp["path"], cfg.DataDir)
	}
}

// --- /ws ---

func TestHandleWS_NonUpgradable(t *testing.T) {
	mux, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	// Gorilla's Upgrade writes its own error when the request is not upgradable;
	// we just verify no panic and that the WebSocket upgrade did not succeed (no 101).
	if w.Code == http.StatusSwitchingProtocols {
		t.Error("did not expect 101 Switching Protocols for a plain HTTP request")
	}
}

// --- /api/db/open-folder ---

func TestHandleDBOpenFolder_MethodNotAllowed(t *testing.T) {
	mux, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodPost, "/api/db/open-folder", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("handleDBOpenFolder POST: got %d, want 405", w.Code)
	}
}

// --- applyCoreSettings (via /api/settings) ---

func TestApplyCoreSettings(t *testing.T) {
	mux, cfg := newTestMux(t)
	w := postJSON(mux, "/api/settings", map[string]string{
		"storage.ball_hit_events": "true",
		"storage.raw_packets":     "true",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — body: %s", w.Code, w.Body.String())
	}
	if !cfg.Storage.BallHitEvents {
		t.Error("BallHitEvents: want true")
	}
	if !cfg.Storage.RawPackets {
		t.Error("RawPackets: want true")
	}
}
