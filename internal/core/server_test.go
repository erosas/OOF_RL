package core_test

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"OOF_RL/internal/config"
	"OOF_RL/internal/core"
	"OOF_RL/internal/db"
	"OOF_RL/internal/hub"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/plugin"
	"OOF_RL/internal/plugins/ballchasing"
	"OOF_RL/internal/plugins/history"
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
	h := hub.New()

	cfgPath := filepath.Join(tmpDir, "config.toml")
	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(bus.Stop)
	srv := core.NewServer(cfgPath, &cfg, database, h, http.NotFoundHandler(), func() {}, nil, bus)
	srv.Use(history.New(&cfg, database))
	srv.Use(ballchasing.New(&cfg, database, h))

	mux := http.NewServeMux()
	srv.Register(mux)
	return mux, &cfg
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
	if len(tabs) < 2 {
		t.Errorf("expected at least 2 tabs, got %d", len(tabs))
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
	if len(blobs) == 0 {
		t.Error("expected at least one settings blob")
	}
}

func TestDisabledPluginSkipsRuntimeButRemainsInSettings(t *testing.T) {
	tmpDir := t.TempDir()
	database, err := db.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	cfg := config.Defaults()
	cfg.DisabledPlugins = []string{"fake"}
	cfgPath := filepath.Join(tmpDir, "config.toml")
	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(bus.Stop)

	fp := &fakePlugin{}
	srv := core.NewServer(cfgPath, &cfg, database, hub.New(), http.NotFoundHandler(), func() {}, nil, bus)
	srv.Use(fp)

	mux := http.NewServeMux()
	srv.Register(mux)
	if err := srv.InitPlugins(); err != nil {
		t.Fatalf("InitPlugins: %v", err)
	}
	if fp.initCalls != 0 {
		t.Fatalf("disabled plugin init calls = %d, want 0", fp.initCalls)
	}

	if w := get(mux, "/fake-plugin-route"); w.Code != http.StatusNotFound {
		t.Fatalf("disabled plugin route status = %d, want 404", w.Code)
	}

	w := get(mux, "/api/nav")
	if w.Code != http.StatusOK {
		t.Fatalf("nav status = %d, want 200", w.Code)
	}
	if strings.Contains(w.Body.String(), "fake") {
		t.Fatalf("disabled plugin appeared in nav: %s", w.Body.String())
	}

	w = get(mux, "/api/settings/schema")
	if w.Code != http.StatusOK {
		t.Fatalf("settings schema status = %d, want 200", w.Code)
	}
	var blobs []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &blobs); err != nil {
		t.Fatalf("parse settings schema: %v", err)
	}
	found := false
	for _, blob := range blobs {
		if blob["plugin_id"] == "fake" {
			found = true
			if enabled, _ := blob["enabled"].(bool); enabled {
				t.Fatalf("disabled plugin settings blob enabled = true, want false")
			}
		}
	}
	if !found {
		t.Fatalf("disabled plugin missing from settings schema: %+v", blobs)
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

type fakePlugin struct {
	initCalls int
}

func (p *fakePlugin) ID() string         { return "fake" }
func (p *fakePlugin) DBPrefix() string   { return "" }
func (p *fakePlugin) Requires() []string { return nil }
func (p *fakePlugin) Init(_ oofevents.PluginBus, _ plugin.Registry, _ *db.DB) error {
	p.initCalls++
	return nil
}
func (p *fakePlugin) Shutdown() error                              { return nil }
func (p *fakePlugin) DeclaredEvents() []oofevents.EventDeclaration { return nil }
func (p *fakePlugin) NavTab() plugin.NavTab {
	return plugin.NavTab{ID: "fake", Label: "Fake", Order: 1}
}
func (p *fakePlugin) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/fake-plugin-route", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
}
func (p *fakePlugin) SettingsSchema() []plugin.Setting        { return nil }
func (p *fakePlugin) ApplySettings(_ map[string]string) error { return nil }
func (p *fakePlugin) Assets() fs.FS                           { return nil }
