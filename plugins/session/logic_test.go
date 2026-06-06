//go:build !wasip1

package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	sdk "github.com/erosas/oof-plugin-sdk"
)

func resetSince() {
	mu.Lock()
	since = time.Time{}
	mu.Unlock()
}

func stubDBQuery(t *testing.T, fn func(string, []string) []map[string]any) {
	t.Helper()
	old := dbQuery
	dbQuery = fn
	t.Cleanup(func() {
		dbQuery = old
	})
}

// --- onEvent ---

func TestOnEventMatchStartedSetsSince(t *testing.T) {
	resetSince()
	onEvent("match.started")

	mu.RLock()
	s := since
	mu.RUnlock()

	if s.IsZero() {
		t.Fatal("since should be set after match.started")
	}
}

func TestOnEventMatchStartedDoesNotOverwrite(t *testing.T) {
	fixed := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	mu.Lock()
	since = fixed
	mu.Unlock()

	onEvent("match.started")

	mu.RLock()
	s := since
	mu.RUnlock()

	if !s.Equal(fixed) {
		t.Errorf("since should not be overwritten when already set; got %v", s)
	}
}

func TestOnEventUnknownTypeIgnored(t *testing.T) {
	resetSince()
	onEvent("unknown.event")

	mu.RLock()
	s := since
	mu.RUnlock()

	if !s.IsZero() {
		t.Fatal("unknown event should not affect since")
	}
}

// --- handleStart ---

func TestHandleStartGETNoSince(t *testing.T) {
	resetSince()
	resp := handleHTTP(sdk.HTTPRequest{Method: "GET", Path: "/api/session/start"})

	if resp.Status != 200 {
		t.Fatalf("status: got %d, want 200", resp.Status)
	}
	var body map[string]any
	json.Unmarshal([]byte(resp.Body), &body)
	if body["active"] != false {
		t.Errorf("active: got %v, want false", body["active"])
	}
	if body["since"] != "" {
		t.Errorf("since: got %v, want empty string", body["since"])
	}
}

func TestHandleStartPOSTSetsSince(t *testing.T) {
	resetSince()
	resp := handleHTTP(sdk.HTTPRequest{
		Method: "POST",
		Path:   "/api/session/start",
		Body:   `{}`,
	})

	if resp.Status != 200 {
		t.Fatalf("status: got %d", resp.Status)
	}
	var body map[string]any
	json.Unmarshal([]byte(resp.Body), &body)
	if body["active"] != true {
		t.Errorf("active: got %v, want true", body["active"])
	}
}

func TestHandleStartPOSTInvalidSince(t *testing.T) {
	resetSince()
	resp := handleHTTP(sdk.HTTPRequest{
		Method: "POST",
		Path:   "/api/session/start",
		Body:   `{"since":"not-a-date"}`,
	})
	if resp.Status != 400 {
		t.Fatalf("status: got %d, want 400", resp.Status)
	}
}

func TestHandleStartBadMethod(t *testing.T) {
	resp := handleHTTP(sdk.HTTPRequest{Method: "DELETE", Path: "/api/session/start"})
	if resp.Status != 405 {
		t.Fatalf("status: got %d, want 405", resp.Status)
	}
}

// --- handleStats ---

func TestHandleStatsMissingPlayer(t *testing.T) {
	resp := handleHTTP(sdk.HTTPRequest{Method: "GET", Path: "/api/session/stats"})
	if resp.Status != 400 {
		t.Fatalf("status: got %d, want 400", resp.Status)
	}
}

func TestHandleStatsNoActiveSince(t *testing.T) {
	resetSince()
	resp := handleHTTP(sdk.HTTPRequest{
		Method: "GET",
		Path:   "/api/session/stats",
		Query:  "player=steam%7Calice%7C0",
	})
	if resp.Status != 200 {
		t.Fatalf("status: got %d", resp.Status)
	}
	var body map[string]any
	json.Unmarshal([]byte(resp.Body), &body)
	if body["matches"] == nil {
		t.Error("matches field missing")
	}
}

// --- handleHistory ---

func TestHandleHistoryMissingPlayer(t *testing.T) {
	resp := handleHTTP(sdk.HTTPRequest{Method: "GET", Path: "/api/session/history"})
	if resp.Status != 400 {
		t.Fatalf("status: got %d, want 400", resp.Status)
	}
}

func TestHandleHistoryBadMethod(t *testing.T) {
	resp := handleHTTP(sdk.HTTPRequest{Method: "POST", Path: "/api/session/history"})
	if resp.Status != 405 {
		t.Fatalf("status: got %d, want 405", resp.Status)
	}
}

// --- handleHistoryItem ---

func TestHandleHistoryItemBadID(t *testing.T) {
	resp := handleHTTP(sdk.HTTPRequest{Method: "GET", Path: "/api/session/history/notanumber"})
	if resp.Status != 400 {
		t.Fatalf("status: got %d, want 400", resp.Status)
	}
}

func TestHandleHistoryItemGETReturnsSessionMatches(t *testing.T) {
	playerID := "steam|alice|0"
	startedAt := "2024-01-01T12:00:00Z"
	endedAt := "2024-01-01T13:00:00Z"
	stubDBQuery(t, func(sql string, args []string) []map[string]any {
		switch {
		case strings.Contains(sql, "FROM sessions WHERE id=?"):
			if len(args) != 1 || args[0] != "1" {
				t.Fatalf("session query args: got %v", args)
			}
			return []map[string]any{{
				"id":         float64(1),
				"player_id":  playerID,
				"started_at": startedAt,
				"ended_at":   endedAt,
			}}
		case strings.Contains(sql, "FROM hist_matches"):
			if len(args) != 1 || args[0] != playerID {
				t.Fatalf("match query args: got %v", args)
			}
			return []map[string]any{{
				"match_id":        float64(10),
				"arena":           "DFH Stadium",
				"started_at":      "2024-01-01T12:15:00Z",
				"winner_team_num": float64(0),
				"incomplete":      float64(0),
				"forfeit":         float64(0),
				"team_num":        float64(0),
				"goals":           float64(2),
				"assists":         float64(1),
				"saves":           float64(3),
				"shots":           float64(4),
				"demos":           float64(1),
				"score":           float64(500),
				"playlist_type":   float64(13),
				"player_count":    float64(6),
			}}
		default:
			return nil
		}
	})

	resp := handleHTTP(sdk.HTTPRequest{Method: "GET", Path: "/api/session/history/1"})
	if resp.Status != 200 {
		t.Fatalf("status: got %d, want 200; body=%s", resp.Status, resp.Body)
	}
	var body struct {
		Session SavedSession   `json:"session"`
		Matches []SessionMatch `json:"matches"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &body); err != nil {
		t.Fatal(err)
	}
	if body.Session.ID != 1 || body.Session.PlayerID != playerID {
		t.Fatalf("session: got %+v", body.Session)
	}
	if body.Session.Games != 1 || body.Session.Wins != 1 || body.Session.Goals != 2 {
		t.Fatalf("session stats: got %+v", body.Session)
	}
	if len(body.Matches) != 1 || body.Matches[0].MatchID != 10 {
		t.Fatalf("matches: got %+v", body.Matches)
	}
}

func TestHandleHistoryItemGETUnknownID(t *testing.T) {
	stubDBQuery(t, func(string, []string) []map[string]any { return nil })
	resp := handleHTTP(sdk.HTTPRequest{Method: "GET", Path: "/api/session/history/404"})
	if resp.Status != 404 {
		t.Fatalf("status: got %d, want 404", resp.Status)
	}
}

func TestHandleHistoryItemPUTInvalidTime(t *testing.T) {
	resp := handleHTTP(sdk.HTTPRequest{
		Method: "PUT",
		Path:   "/api/session/history/1",
		Body:   `{"started_at":"bad","ended_at":"bad"}`,
	})
	if resp.Status != 400 {
		t.Fatalf("status: got %d, want 400", resp.Status)
	}
}

func TestHandleHistoryItemPUTEndBeforeStart(t *testing.T) {
	resp := handleHTTP(sdk.HTTPRequest{
		Method: "PUT",
		Path:   "/api/session/history/1",
		Body:   `{"started_at":"2024-01-01T12:00:00Z","ended_at":"2024-01-01T11:00:00Z"}`,
	})
	if resp.Status != 400 {
		t.Fatalf("status: got %d, want 400", resp.Status)
	}
}

func TestHandleHistoryItemBadMethod(t *testing.T) {
	resp := handleHTTP(sdk.HTTPRequest{Method: "PATCH", Path: "/api/session/history/1"})
	if resp.Status != 405 {
		t.Fatalf("status: got %d, want 405", resp.Status)
	}
}

// --- handleNew ---

func TestHandleNewBadMethod(t *testing.T) {
	resp := handleHTTP(sdk.HTTPRequest{Method: "GET", Path: "/api/session/new"})
	if resp.Status != 405 {
		t.Fatalf("status: got %d, want 405", resp.Status)
	}
}

func TestHandleNewResetsSince(t *testing.T) {
	fixed := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	mu.Lock()
	since = fixed
	mu.Unlock()

	resp := handleHTTP(sdk.HTTPRequest{Method: "POST", Path: "/api/session/new", Body: `{}`})
	if resp.Status != 200 {
		t.Fatalf("status: got %d", resp.Status)
	}

	mu.RLock()
	s := since
	mu.RUnlock()
	if s.Equal(fixed) {
		t.Error("since should have been reset to a new time")
	}
}

func TestHandleNewWithoutActiveSessionStillResets(t *testing.T) {
	resetSince()
	resp := handleHTTP(sdk.HTTPRequest{
		Method: "POST",
		Path:   "/api/session/new",
		Body:   `{"player_id":"steam|alice|0"}`,
	})
	if resp.Status != 200 {
		t.Fatalf("status: got %d", resp.Status)
	}
	mu.RLock()
	s := since
	mu.RUnlock()
	if s.IsZero() {
		t.Error("since should have been set to a new time")
	}
}

// --- utilities ---

func TestQueryParam(t *testing.T) {
	if got := sdk.QueryParam("player=abc&foo=bar", "player"); got != "abc" {
		t.Errorf("got %q, want abc", got)
	}
	if got := sdk.QueryParam("player=steam%7Calice%7C0", "player"); got != "steam|alice|0" {
		t.Errorf("URL decode: got %q", got)
	}
	if got := sdk.QueryParam("", "player"); got != "" {
		t.Errorf("empty query: got %q", got)
	}
}

func TestParseTime(t *testing.T) {
	cases := []string{
		"2024-01-15T10:30:00Z",
		"2024-01-15T10:30:00.123456789Z",
		"2024-01-15 10:30:00",
	}
	for _, s := range cases {
		if sdk.ParseTime(s).IsZero() {
			t.Errorf("sdk.ParseTime(%q) returned zero", s)
		}
	}
	if !sdk.ParseTime("not-a-time").IsZero() {
		t.Error("sdk.ParseTime(invalid) should return zero")
	}
}
