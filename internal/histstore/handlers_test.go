package histstore_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"OOF_RL/internal/db"
	"OOF_RL/internal/histstore"
)

func newHandlerMux(t *testing.T) (*http.ServeMux, *histstore.Store) {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	if err := histstore.Migrate(database); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	s := histstore.NewStore(database)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/players", s.HandlePlayers)
	mux.HandleFunc("/api/matches", s.HandleMatches)
	mux.HandleFunc("/api/matches/", s.HandleMatchDetail)
	return mux, s
}

func hget(mux *http.ServeMux, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestHandlePlayersEmpty(t *testing.T) {
	mux, _ := newHandlerMux(t)
	w := hget(mux, "/api/players")
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

func TestHandlePlayersWithData(t *testing.T) {
	mux, s := newHandlerMux(t)
	s.UpsertPlayer("pid1", "Alice")
	s.UpsertPlayer("pid2", "Bob")

	w := hget(mux, "/api/players")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var players []any
	json.Unmarshal(w.Body.Bytes(), &players)
	if len(players) != 2 {
		t.Errorf("expected 2 players, got %d", len(players))
	}
}

func TestHandleMatchesEmpty(t *testing.T) {
	mux, _ := newHandlerMux(t)
	w := hget(mux, "/api/matches")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var matches []any
	json.Unmarshal(w.Body.Bytes(), &matches)
	if len(matches) != 0 {
		t.Errorf("expected empty, got %d", len(matches))
	}
}

func TestHandleMatchesWithPlayerFilter(t *testing.T) {
	mux, s := newHandlerMux(t)
	s.UpsertPlayer("pid1", "Alice")
	s.UpsertPlayer("pid2", "Bob")
	m1, _ := s.UpsertMatch("guid-1", "DFH Stadium", time.Now())
	m2, _ := s.UpsertMatch("guid-2", "Mannfield", time.Now())
	s.UpsertPlayerMatchStats(m1, "pid1", 0, 100, 1, 1, 0, 0, 0, 0, 0)
	s.UpsertPlayerMatchStats(m2, "pid2", 1, 200, 2, 2, 0, 0, 0, 0, 0)

	w := hget(mux, "/api/matches")
	var all []any
	json.Unmarshal(w.Body.Bytes(), &all)
	if len(all) != 2 {
		t.Errorf("expected 2 matches, got %d", len(all))
	}

	w = hget(mux, "/api/matches?player=pid1")
	var filtered []any
	json.Unmarshal(w.Body.Bytes(), &filtered)
	if len(filtered) != 1 {
		t.Errorf("expected 1 match for pid1, got %d", len(filtered))
	}
}

func TestHandleMatchDetailBadID(t *testing.T) {
	mux, _ := newHandlerMux(t)
	w := hget(mux, "/api/matches/not-a-number")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestHandleMatchDetail(t *testing.T) {
	mux, s := newHandlerMux(t)
	s.UpsertPlayer("pid1", "Alice")
	matchID, _ := s.UpsertMatch("guid-1", "DFH Stadium", time.Now())
	s.UpsertPlayerMatchStats(matchID, "pid1", 0, 500, 3, 5, 1, 2, 10, 8, 1)
	s.InsertGoal(matchID, "pid1", "Alice", "", "", "", 110.0, 45.0, 0, 0, 0)

	w := hget(mux, "/api/matches/"+strconv.FormatInt(matchID, 10))
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