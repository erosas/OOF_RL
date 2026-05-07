package session

import (
	"path/filepath"
	"testing"
	"time"

	"OOF_RL/internal/db"
)

func newSessionTestStore(t *testing.T) *store {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	if err := database.RunMigration(sessionSchema); err != nil {
		t.Fatalf("session migration: %v", err)
	}
	if err := database.RunMigration(`
		CREATE TABLE hist_matches (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			started_at      DATETIME NOT NULL,
			winner_team_num INTEGER,
			incomplete      BOOLEAN DEFAULT 0,
			forfeit         BOOLEAN DEFAULT 0,
			arena           TEXT,
			playlist_type   INTEGER
		);
		CREATE TABLE hist_player_match_stats (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			match_id   INTEGER NOT NULL,
			primary_id TEXT NOT NULL,
			team_num   INTEGER,
			score      INTEGER DEFAULT 0,
			goals      INTEGER DEFAULT 0,
			shots      INTEGER DEFAULT 0,
			assists    INTEGER DEFAULT 0,
			saves      INTEGER DEFAULT 0,
			demos      INTEGER DEFAULT 0
		);
	`); err != nil {
		t.Fatalf("history test schema: %v", err)
	}
	return &store{conn: database.Conn()}
}

func insertSessionTestMatch(t *testing.T, s *store, startedAt time.Time, primaryID string) int64 {
	t.Helper()
	res, err := s.conn.Exec(`
		INSERT INTO hist_matches(started_at, winner_team_num, incomplete, forfeit, arena, playlist_type)
		VALUES(?, 0, 0, 0, 'DFH Stadium', 11)`, startedAt)
	if err != nil {
		t.Fatalf("insert hist_match: %v", err)
	}
	matchID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId: %v", err)
	}
	_, err = s.conn.Exec(`
		INSERT INTO hist_player_match_stats(match_id, primary_id, team_num, score, goals, shots, assists, saves, demos)
		VALUES(?, ?, 0, 450, 2, 5, 1, 3, 1)`, matchID, primaryID)
	if err != nil {
		t.Fatalf("insert hist_player_match_stats: %v", err)
	}
	return matchID
}

func TestListSessionsWithStatsIncludesLocalHistoryInsideUTCSessionWindow(t *testing.T) {
	s := newSessionTestStore(t)
	playerID := "steam|player"
	local := time.FixedZone("EDT", -4*60*60)
	sessionStart := time.Date(2026, 5, 5, 20, 0, 0, 0, time.UTC)
	sessionEnd := sessionStart.Add(90 * time.Minute)
	matchStart := time.Date(2026, 5, 5, 16, 30, 0, 0, local) // same instant as 20:30 UTC

	if _, err := s.createSession(playerID, sessionStart, sessionEnd); err != nil {
		t.Fatalf("createSession: %v", err)
	}
	insertSessionTestMatch(t, s, matchStart, playerID)

	sessions, err := s.listSessionsWithStats(playerID)
	if err != nil {
		t.Fatalf("listSessionsWithStats: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected one saved session, got %d", len(sessions))
	}
	got := sessions[0]
	if got.Games != 1 || got.Wins != 1 || got.Losses != 0 {
		t.Fatalf("expected 1 game, 1 win, 0 losses, got games=%d wins=%d losses=%d", got.Games, got.Wins, got.Losses)
	}
	if got.Goals != 2 || got.Assists != 1 || got.Saves != 3 || got.Shots != 5 || got.Demos != 1 {
		t.Fatalf("unexpected stat totals: %+v", got)
	}
}

func TestSessionMatchesByPlayerIncludesLocalHistoryForUTCSince(t *testing.T) {
	s := newSessionTestStore(t)
	playerID := "steam|player"
	local := time.FixedZone("EDT", -4*60*60)
	since := time.Date(2026, 5, 5, 20, 0, 0, 0, time.UTC)
	matchStart := time.Date(2026, 5, 5, 16, 15, 0, 0, local) // same instant as 20:15 UTC

	matchID := insertSessionTestMatch(t, s, matchStart, playerID)

	matches, err := s.sessionMatchesByPlayer(since, playerID)
	if err != nil {
		t.Fatalf("sessionMatchesByPlayer: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one session match, got %d", len(matches))
	}
	if matches[0].MatchID != matchID || matches[0].Goals != 2 || matches[0].Score != 450 {
		t.Fatalf("unexpected match result: %+v", matches[0])
	}
}
