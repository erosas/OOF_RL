package core_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"OOF_RL/internal/config"
	"OOF_RL/internal/core"
	"OOF_RL/internal/db"
	"OOF_RL/internal/hub"
	"OOF_RL/internal/plugins/ballchasing"
	"OOF_RL/internal/plugins/history"
)

func newTestMux(t *testing.T) (*http.ServeMux, *db.DB, *config.Config) {
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
	srv := core.NewServer(cfgPath, &cfg, database, h, http.NotFoundHandler(), func() {}, nil)
	srv.Use(history.New(&cfg, database))
	srv.Use(ballchasing.New(&cfg, database, h))

	mux := http.NewServeMux()
	srv.Register(mux)
	return mux, database, &cfg
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

// --- /api/players ---

func TestGetPlayersEmpty(t *testing.T) {
	mux, _, _ := newTestMux(t)
	w := get(mux, "/api/players")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
	var players []any
	if err := json.Unmarshal(w.Body.Bytes(), &players); err != nil {
		t.Fatalf("parse: %v — body: %s", err, w.Body.String())
	}
	if len(players) != 0 {
		t.Errorf("expected empty array, got %d", len(players))
	}
}

func TestGetPlayersWithData(t *testing.T) {
	mux, database, _ := newTestMux(t)
	database.UpsertPlayer("pid1", "Alice")
	database.UpsertPlayer("pid2", "Bob")

	w := get(mux, "/api/players")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var players []any
	json.Unmarshal(w.Body.Bytes(), &players)
	if len(players) != 2 {
		t.Errorf("expected 2 players, got %d", len(players))
	}
}

// --- /api/matches ---

func TestGetMatchesEmpty(t *testing.T) {
	mux, _, _ := newTestMux(t)
	w := get(mux, "/api/matches")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var matches []any
	json.Unmarshal(w.Body.Bytes(), &matches)
	if len(matches) != 0 {
		t.Errorf("expected empty, got %d", len(matches))
	}
}

func TestGetMatchesWithPlayerFilter(t *testing.T) {
	mux, database, _ := newTestMux(t)
	database.UpsertPlayer("pid1", "Alice")
	database.UpsertPlayer("pid2", "Bob")
	m1, _ := database.UpsertMatch("guid-1", "DFH Stadium", time.Now())
	m2, _ := database.UpsertMatch("guid-2", "Mannfield", time.Now())
	database.UpsertPlayerMatchStats(m1, "pid1", 0, 100, 1, 1, 0, 0, 0, 0, 0)
	database.UpsertPlayerMatchStats(m2, "pid2", 1, 200, 2, 2, 0, 0, 0, 0, 0)

	w := get(mux, "/api/matches")
	var all []any
	json.Unmarshal(w.Body.Bytes(), &all)
	if len(all) != 2 {
		t.Errorf("expected 2 matches, got %d", len(all))
	}

	w = get(mux, "/api/matches?player=pid1")
	var filtered []any
	json.Unmarshal(w.Body.Bytes(), &filtered)
	if len(filtered) != 1 {
		t.Errorf("expected 1 match for pid1, got %d", len(filtered))
	}
}

// --- /api/matches/{id} ---

func TestGetMatchDetailBadID(t *testing.T) {
	mux, _, _ := newTestMux(t)
	w := get(mux, "/api/matches/not-a-number")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestGetMatchDetail(t *testing.T) {
	mux, database, _ := newTestMux(t)
	database.UpsertPlayer("pid1", "Alice")
	matchID, _ := database.UpsertMatch("guid-1", "DFH Stadium", time.Now())
	database.UpsertPlayerMatchStats(matchID, "pid1", 0, 500, 3, 5, 1, 2, 10, 8, 1)
	database.InsertGoal(matchID, "pid1", "Alice", "", "", "", 110.0, 45.0, 0, 0, 0)

	w := get(mux, "/api/matches/"+strconv.FormatInt(matchID, 10))
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var detail map[string]any
	json.Unmarshal(w.Body.Bytes(), &detail)
	if _, ok := detail["players"]; !ok {
		t.Error("expected players key in response")
	}
	if _, ok := detail["goals"]; !ok {
		t.Error("expected goals key in response")
	}
}

// --- /api/config ---

func TestGetConfig(t *testing.T) {
	mux, _, cfg := newTestMux(t)
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
	mux, _, _ := newTestMux(t)

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
	mux, _, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestConfigMethodNotAllowed(t *testing.T) {
	mux, _, _ := newTestMux(t)
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
	mux, _, _ := newTestMux(t)
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
	mux, _, cfg := newTestMux(t)
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
	mux, _, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodPost, "/api/config/ini", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestINIMethodNotAllowed(t *testing.T) {
	mux, _, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/config/ini", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want 405", w.Code)
	}
}

// --- /api/nav ---

func TestGetNav(t *testing.T) {
	mux, _, _ := newTestMux(t)
	w := get(mux, "/api/nav")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var tabs []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &tabs); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(tabs) < 2 {
		t.Errorf("expected at least 2 tabs (history + ballchasing), got %d", len(tabs))
	}
}

// --- /api/settings/schema ---

func TestGetSettingsSchema(t *testing.T) {
	mux, _, _ := newTestMux(t)
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

// --- /api/settings ---

func TestPostSettings(t *testing.T) {
	mux, _, cfg := newTestMux(t)

	w := postJSON(mux, "/api/settings", map[string]string{
		"ballchasing_api_key": "test-key",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — body: %s", w.Code, w.Body.String())
	}
	if cfg.BallchasingAPIKey != "test-key" {
		t.Errorf("bc key: got %q, want test-key", cfg.BallchasingAPIKey)
	}

	// Verify TrackerCacheTTLMinutes is saved via /api/config.
	// Use a full Defaults() + override to avoid zeroing unrelated fields.
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
	mux, _, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want 405", w.Code)
	}
}

func TestPostSettingsBadJSON(t *testing.T) {
	mux, _, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodPost, "/api/settings", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}