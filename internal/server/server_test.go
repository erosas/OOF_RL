package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"OOF_RL/internal/config"
	"OOF_RL/internal/db"
	"OOF_RL/internal/hub"
	"OOF_RL/internal/server"
)

func newTestMux(t *testing.T) (*http.ServeMux, *db.DB, *config.Config) {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	cfg := config.Defaults()
	cfg.RLInstallPath = ""
	h := hub.New()
	srv := server.New(&cfg, database, h, http.NotFoundHandler(), func() {})
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

	// Without filter: both matches.
	w := get(mux, "/api/matches")
	var all []any
	json.Unmarshal(w.Body.Bytes(), &all)
	if len(all) != 2 {
		t.Errorf("expected 2 matches, got %d", len(all))
	}

	// With filter: only pid1's match.
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

// --- /api/players/{id} ---

func TestGetPlayerDetail(t *testing.T) {
	mux, database, _ := newTestMux(t)
	database.UpsertPlayer("pid1", "Alice")
	matchID, _ := database.UpsertMatch("guid-1", "DFH Stadium", time.Now())
	database.UpsertPlayerMatchStats(matchID, "pid1", 0, 500, 3, 5, 1, 2, 10, 8, 1)

	w := get(mux, "/api/players/pid1")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — body: %s", w.Code, w.Body.String())
	}
	var detail map[string]any
	json.Unmarshal(w.Body.Bytes(), &detail)
	if _, ok := detail["aggregate"]; !ok {
		t.Error("expected aggregate key")
	}
	if _, ok := detail["matches"]; !ok {
		t.Error("expected matches key")
	}
}

func TestGetPlayerDetailNotFound(t *testing.T) {
	mux, _, _ := newTestMux(t)
	w := get(mux, "/api/players/nonexistent")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want 500", w.Code)
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

	// Write config.toml to a temp dir, not the real working directory.
	origDir, _ := os.Getwd()
	os.Chdir(t.TempDir())
	defer os.Chdir(origDir)

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
	// Clear USERPROFILE so detectUserConfigDir returns "" and we fall back to
	// the install-dir path, which points nowhere — triggering the error path.
	t.Setenv("USERPROFILE", "")

	mux, _, _ := newTestMux(t) // cfg.RLInstallPath = ""
	w := get(mux, "/api/config/ini")

	// Server returns 200 with error:true rather than 500.
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
	cfg.RLInstallPath = t.TempDir() // writable dir for the INI fallback path

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

// --- /api/replays ---

func TestGetReplaysEmpty(t *testing.T) {
	t.Setenv("USERPROFILE", t.TempDir())
	mux, _, _ := newTestMux(t)
	w := get(mux, "/api/replays")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var files []any
	if err := json.Unmarshal(w.Body.Bytes(), &files); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected empty replay list, got %d", len(files))
	}
}

func TestGetReplaysWithFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	demos := filepath.Join(home, "Documents", "My Games", "Rocket League", "TAGame", "Demos")
	if err := os.MkdirAll(demos, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(demos, "a.replay"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(demos, "ignore.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	mux, _, _ := newTestMux(t)
	w := get(mux, "/api/replays")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var files []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &files); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 replay file, got %d", len(files))
	}
	if files[0]["name"] != "a.replay" {
		t.Fatalf("expected a.replay, got %v", files[0]["name"])
	}
}

// --- /api/captures ---

// createTestCapture writes a minimal capture directory for testing.
func createTestCapture(t *testing.T, capturesDir, captureID, matchGuid string, eventLines []string) {
	t.Helper()
	dir := filepath.Join(capturesDir, captureID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir capture dir: %v", err)
	}

	ndjson := strings.Join(eventLines, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, "packets_normalized_001.ndjson"), []byte(ndjson), 0644); err != nil {
		t.Fatalf("write ndjson: %v", err)
	}

	meta := map[string]any{
		"meta_version":     1,
		"match_guid":       matchGuid,
		"started_at_utc":   "2026-01-01T12:00:00Z",
		"ended_at_utc":     "2026-01-01T12:05:00Z",
		"end_reason":       "MatchEnded",
		"duration_ms":      300000,
		"packet_count":     len(eventLines),
		"chunk_size":       10000,
		"chunk_count":      1,
		"normalized_files": []string{"packets_normalized_001.ndjson"},
		"wire_files":       []string{"packets_wire_001.ndjson"},
	}
	b, _ := json.Marshal(meta)
	if err := os.WriteFile(filepath.Join(dir, "capture_meta.json"), b, 0644); err != nil {
		t.Fatalf("write capture_meta.json: %v", err)
	}

	idx := map[string]any{
		"version":      1,
		"match_guid":   matchGuid,
		"packet_count": len(eventLines),
		"markers":      []any{},
		"event_counts": map[string]int{},
	}
	b, _ = json.Marshal(idx)
	if err := os.WriteFile(filepath.Join(dir, "capture_index.json"), b, 0644); err != nil {
		t.Fatalf("write capture_index.json: %v", err)
	}
}

func TestGetCapturesEmpty(t *testing.T) {
	mux, _, cfg := newTestMux(t)
	cfg.Storage.RawPacketsDir = t.TempDir()

	w := get(mux, "/api/captures")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var captures []any
	json.Unmarshal(w.Body.Bytes(), &captures)
	if len(captures) != 0 {
		t.Errorf("expected empty captures list, got %d", len(captures))
	}
}

func TestGetCapturesWithData(t *testing.T) {
	mux, _, cfg := newTestMux(t)
	capturesDir := t.TempDir()
	cfg.Storage.RawPacketsDir = capturesDir

	events := []string{
		`{"Event":"MatchInitialized","Data":{"MatchGuid":"test-guid-1"}}`,
		`{"Event":"UpdateState","Data":{}}`,
	}
	createTestCapture(t, capturesDir, "TESTGUID1_20260101_120000", "test-guid-1", events)

	w := get(mux, "/api/captures")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var captures []map[string]any
	json.Unmarshal(w.Body.Bytes(), &captures)
	if len(captures) != 1 {
		t.Fatalf("expected 1 capture, got %d", len(captures))
	}
	if captures[0]["match_guid"] != "test-guid-1" {
		t.Errorf("match_guid: got %v", captures[0]["match_guid"])
	}
	if captures[0]["id"] != "TESTGUID1_20260101_120000" {
		t.Errorf("id: got %v", captures[0]["id"])
	}
}

func TestGetCaptureDetailMeta(t *testing.T) {
	mux, _, cfg := newTestMux(t)
	capturesDir := t.TempDir()
	cfg.Storage.RawPacketsDir = capturesDir

	createTestCapture(t, capturesDir, "CAP_20260101_120000", "guid-meta", []string{`{"Event":"A","Data":{}}`})

	w := get(mux, "/api/captures/CAP_20260101_120000/meta")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var meta map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &meta); err != nil {
		t.Fatalf("parse meta: %v", err)
	}
	if meta["match_guid"] != "guid-meta" {
		t.Errorf("match_guid: got %v", meta["match_guid"])
	}
}

func TestGetCaptureDetailIndex(t *testing.T) {
	mux, _, cfg := newTestMux(t)
	capturesDir := t.TempDir()
	cfg.Storage.RawPacketsDir = capturesDir

	createTestCapture(t, capturesDir, "CAP_20260101_120001", "guid-idx", []string{`{"Event":"A","Data":{}}`})

	w := get(mux, "/api/captures/CAP_20260101_120001/index")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var idx map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &idx); err != nil {
		t.Fatalf("parse index: %v", err)
	}
	if idx["match_guid"] != "guid-idx" {
		t.Errorf("match_guid: got %v", idx["match_guid"])
	}
}

func TestGetCaptureDetailEvents(t *testing.T) {
	mux, _, cfg := newTestMux(t)
	capturesDir := t.TempDir()
	cfg.Storage.RawPacketsDir = capturesDir

	events := []string{
		`{"Event":"MatchInitialized","Data":{"MatchGuid":"guid-ev"}}`,
		`{"Event":"UpdateState","Data":{}}`,
	}
	createTestCapture(t, capturesDir, "CAP_20260101_120002", "guid-ev", events)

	w := get(mux, "/api/captures/CAP_20260101_120002/events")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	body := strings.TrimSpace(w.Body.String())
	lines := strings.Split(body, "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 event lines, got %d: %q", len(lines), body)
	}
}

func TestGetCaptureDetailBadPath(t *testing.T) {
	mux, _, cfg := newTestMux(t)
	capturesDir := t.TempDir()
	cfg.Storage.RawPacketsDir = capturesDir
	createTestCapture(t, capturesDir, "CAP_20260101_120003", "g", []string{`{}`})

	// Missing sub-resource (no /kind segment).
	w := get(mux, "/api/captures/")
	if w.Code != http.StatusBadRequest {
		t.Errorf("empty capture id: got %d, want 400", w.Code)
	}

	// Unknown sub-resource kind.
	w = get(mux, "/api/captures/CAP_20260101_120003/unknown")
	if w.Code != http.StatusBadRequest {
		t.Errorf("unknown sub-resource: got %d, want 400", w.Code)
	}
}

func TestGetCaptureDetailNotFound(t *testing.T) {
	mux, _, cfg := newTestMux(t)
	cfg.Storage.RawPacketsDir = t.TempDir()

	w := get(mux, "/api/captures/NONEXISTENT_20260101_120000/meta")
	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", w.Code)
	}
}
