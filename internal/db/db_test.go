package db

import (
	"path/filepath"
	"testing"
	"time"
)

// testSchema creates the plugin-owned tables so db query-method tests work without importing plugins.
const testSchema = `
CREATE TABLE IF NOT EXISTS hist_players (
	primary_id TEXT PRIMARY KEY,
	name       TEXT NOT NULL,
	last_seen  DATETIME NOT NULL
);
CREATE TABLE IF NOT EXISTS hist_matches (
	id              INTEGER PRIMARY KEY AUTOINCREMENT,
	match_guid      TEXT UNIQUE,
	arena           TEXT,
	started_at      DATETIME NOT NULL,
	ended_at        DATETIME,
	winner_team_num INTEGER,
	overtime        BOOLEAN DEFAULT 0,
	playlist_type   INTEGER
);
CREATE TABLE IF NOT EXISTS hist_player_match_stats (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	match_id    INTEGER NOT NULL REFERENCES hist_matches(id),
	primary_id  TEXT    NOT NULL REFERENCES hist_players(primary_id),
	team_num    INTEGER,
	score       INTEGER DEFAULT 0,
	goals       INTEGER DEFAULT 0,
	shots       INTEGER DEFAULT 0,
	assists     INTEGER DEFAULT 0,
	saves       INTEGER DEFAULT 0,
	touches     INTEGER DEFAULT 0,
	car_touches INTEGER DEFAULT 0,
	demos       INTEGER DEFAULT 0,
	UNIQUE(match_id, primary_id)
);
CREATE TABLE IF NOT EXISTS hist_goal_events (
	id                   INTEGER PRIMARY KEY AUTOINCREMENT,
	match_id             INTEGER NOT NULL REFERENCES hist_matches(id),
	scorer_id            TEXT REFERENCES hist_players(primary_id),
	scorer_name          TEXT NOT NULL DEFAULT '',
	assister_id          TEXT REFERENCES hist_players(primary_id),
	assister_name        TEXT NOT NULL DEFAULT '',
	ball_last_touch_id   TEXT REFERENCES hist_players(primary_id),
	goal_speed           REAL,
	goal_time            REAL,
	impact_x             REAL,
	impact_y             REAL,
	impact_z             REAL,
	scored_at            DATETIME NOT NULL
);
CREATE TABLE IF NOT EXISTS hist_ball_hit_events (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	match_id       INTEGER NOT NULL REFERENCES hist_matches(id),
	player_id      TEXT REFERENCES hist_players(primary_id),
	pre_hit_speed  REAL,
	post_hit_speed REAL,
	loc_x          REAL,
	loc_y          REAL,
	loc_z          REAL,
	hit_at         DATETIME NOT NULL
);
CREATE TABLE IF NOT EXISTS hist_tick_snapshots (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	match_id    INTEGER NOT NULL REFERENCES hist_matches(id),
	captured_at DATETIME NOT NULL,
	raw_json    TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS bc_uploads (
	replay_name    TEXT PRIMARY KEY,
	ballchasing_id TEXT NOT NULL,
	bc_url         TEXT NOT NULL,
	uploaded_at    DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_hist_pms_primary_id ON hist_player_match_stats(primary_id);
CREATE INDEX IF NOT EXISTS idx_hist_goal_scorer     ON hist_goal_events(scorer_id);
CREATE INDEX IF NOT EXISTS idx_hist_goal_match      ON hist_goal_events(match_id);
CREATE INDEX IF NOT EXISTS idx_hist_bh_player       ON hist_ball_hit_events(player_id);
CREATE INDEX IF NOT EXISTS idx_hist_tick_match      ON hist_tick_snapshots(match_id);
`

func newTestDB(t *testing.T) *DB {
	t.Helper()
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := d.RunMigration(testSchema); err != nil {
		t.Fatalf("RunMigration(testSchema): %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestMigrate(t *testing.T) {
	d := newTestDB(t)
	players, err := d.AllPlayers()
	if err != nil {
		t.Fatalf("AllPlayers after migrate: %v", err)
	}
	if len(players) != 0 {
		t.Errorf("expected empty players, got %d", len(players))
	}
}

func TestUpsertAndFetchPlayer(t *testing.T) {
	d := newTestDB(t)

	if err := d.UpsertPlayer("pid1", "Alice"); err != nil {
		t.Fatalf("UpsertPlayer: %v", err)
	}
	players, err := d.AllPlayers()
	if err != nil {
		t.Fatalf("AllPlayers: %v", err)
	}
	if len(players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(players))
	}
	if players[0].PrimaryID != "pid1" || players[0].Name != "Alice" {
		t.Errorf("unexpected player: %+v", players[0])
	}

	// Upsert should update the name.
	if err := d.UpsertPlayer("pid1", "Alice Updated"); err != nil {
		t.Fatalf("UpsertPlayer update: %v", err)
	}
	players, _ = d.AllPlayers()
	if players[0].Name != "Alice Updated" {
		t.Errorf("expected updated name, got %q", players[0].Name)
	}
}

func TestUpsertMatchNewAndDuplicate(t *testing.T) {
	d := newTestDB(t)

	id, err := d.UpsertMatch("guid-1", "DFH Stadium", time.Now())
	if err != nil {
		t.Fatalf("UpsertMatch: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}

	// Same GUID → same row, same id returned.
	id2, err := d.UpsertMatch("guid-1", "DFH Stadium", time.Now())
	if err != nil {
		t.Fatalf("UpsertMatch duplicate: %v", err)
	}
	if id2 != id {
		t.Errorf("expected same id %d on duplicate upsert, got %d", id, id2)
	}
}

func TestUpsertMatchEmptyGUID(t *testing.T) {
	d := newTestDB(t)
	id, err := d.UpsertMatch("", "DFH Stadium", time.Now())
	if err != nil {
		t.Fatalf("UpsertMatch empty guid: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}
}

func TestEndMatch(t *testing.T) {
	d := newTestDB(t)

	id, _ := d.UpsertMatch("guid-end", "Mannfield", time.Now())
	if err := d.EndMatch(id, 1, true); err != nil {
		t.Fatalf("EndMatch: %v", err)
	}

	matches, err := d.Matches("")
	if err != nil {
		t.Fatalf("Matches: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	m := matches[0]
	if m.WinnerTeamNum != 1 {
		t.Errorf("WinnerTeamNum: got %d, want 1", m.WinnerTeamNum)
	}
	if !m.Overtime {
		t.Error("expected Overtime true")
	}
}

func TestUpsertPlayerMatchStats(t *testing.T) {
	d := newTestDB(t)

	d.UpsertPlayer("pid1", "Alice")
	matchID, _ := d.UpsertMatch("guid-stats", "Wasteland", time.Now())

	if err := d.UpsertPlayerMatchStats(matchID, "pid1", 0, 500, 3, 5, 1, 2, 10, 8, 1); err != nil {
		t.Fatalf("UpsertPlayerMatchStats: %v", err)
	}

	players, err := d.MatchPlayers(matchID)
	if err != nil {
		t.Fatalf("MatchPlayers: %v", err)
	}
	if len(players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(players))
	}
	p := players[0]
	if p.Goals != 3 || p.Shots != 5 || p.Assists != 1 || p.Saves != 2 || p.Demos != 1 {
		t.Errorf("unexpected stats: %+v", p)
	}
	if p.Score != 500 {
		t.Errorf("Score: got %d, want 500", p.Score)
	}
}

func TestUpsertPlayerMatchStatsIdempotent(t *testing.T) {
	d := newTestDB(t)
	d.UpsertPlayer("pid1", "Alice")
	matchID, _ := d.UpsertMatch("guid-idem", "Neo Tokyo", time.Now())

	d.UpsertPlayerMatchStats(matchID, "pid1", 0, 500, 3, 5, 1, 2, 10, 8, 1)
	if err := d.UpsertPlayerMatchStats(matchID, "pid1", 0, 600, 4, 6, 2, 3, 11, 9, 2); err != nil {
		t.Fatalf("second UpsertPlayerMatchStats: %v", err)
	}
	players, _ := d.MatchPlayers(matchID)
	if players[0].Goals != 4 {
		t.Errorf("expected updated goals=4, got %d", players[0].Goals)
	}
}

func TestInsertAndFetchGoal(t *testing.T) {
	d := newTestDB(t)
	d.UpsertPlayer("scorer1", "Bob")
	d.UpsertPlayer("assist1", "Carol")
	d.UpsertPlayer("touch1", "Dave")
	matchID, _ := d.UpsertMatch("guid-goal", "Beckwith Park", time.Now())

	if err := d.InsertGoal(matchID, "scorer1", "Bob", "assist1", "Carol", "touch1", 120.5, 45.0, 1.0, 2.0, 3.0); err != nil {
		t.Fatalf("InsertGoal: %v", err)
	}

	goals, err := d.MatchGoals(matchID)
	if err != nil {
		t.Fatalf("MatchGoals: %v", err)
	}
	if len(goals) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(goals))
	}
	g := goals[0]
	if g.ScorerID != "scorer1" {
		t.Errorf("ScorerID: got %q, want scorer1", g.ScorerID)
	}
	if g.ScorerName != "Bob" {
		t.Errorf("ScorerName: got %q, want Bob", g.ScorerName)
	}
	if g.AssisterID != "assist1" {
		t.Errorf("AssisterID: got %q, want assist1", g.AssisterID)
	}
	if g.AssisterName != "Carol" {
		t.Errorf("AssisterName: got %q, want Carol", g.AssisterName)
	}
	if g.GoalSpeed != 120.5 {
		t.Errorf("GoalSpeed: got %f, want 120.5", g.GoalSpeed)
	}
	if g.ImpactX != 1.0 || g.ImpactY != 2.0 || g.ImpactZ != 3.0 {
		t.Errorf("Impact: got (%f,%f,%f)", g.ImpactX, g.ImpactY, g.ImpactZ)
	}
}

func TestInsertGoalNoAssister(t *testing.T) {
	d := newTestDB(t)
	d.UpsertPlayer("scorer1", "Bob")
	matchID, _ := d.UpsertMatch("guid-noassist", "Aquadome", time.Now())

	if err := d.InsertGoal(matchID, "scorer1", "Bob", "", "", "", 100.0, 60.0, 0, 0, 0); err != nil {
		t.Fatalf("InsertGoal without assister: %v", err)
	}
	goals, _ := d.MatchGoals(matchID)
	if goals[0].AssisterID != "" {
		t.Errorf("expected empty AssisterID, got %q", goals[0].AssisterID)
	}
}

func TestInsertBallHit(t *testing.T) {
	d := newTestDB(t)
	d.UpsertPlayer("pid1", "Eve")
	matchID, _ := d.UpsertMatch("guid-bh", "Urban Central", time.Now())

	if err := d.InsertBallHit(matchID, "pid1", 50.0, 80.0, 10.0, 20.0, 30.0); err != nil {
		t.Fatalf("InsertBallHit: %v", err)
	}
}

func TestInsertTick(t *testing.T) {
	d := newTestDB(t)
	matchID, _ := d.UpsertMatch("guid-tick", "Champions Field", time.Now())

	if err := d.InsertTick(matchID, `{"Game":{}}`); err != nil {
		t.Fatalf("InsertTick: %v", err)
	}
}

func TestMatchesFilter(t *testing.T) {
	d := newTestDB(t)
	d.UpsertPlayer("pid1", "Alice")
	d.UpsertPlayer("pid2", "Bob")

	m1, _ := d.UpsertMatch("guid-m1", "DFH Stadium", time.Now())
	m2, _ := d.UpsertMatch("guid-m2", "Mannfield", time.Now())
	d.UpsertPlayerMatchStats(m1, "pid1", 0, 100, 1, 1, 0, 0, 0, 0, 0)
	d.UpsertPlayerMatchStats(m2, "pid2", 1, 200, 2, 2, 0, 0, 0, 0, 0)

	all, err := d.Matches("")
	if err != nil {
		t.Fatalf("Matches all: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 matches, got %d", len(all))
	}

	filtered, err := d.Matches("pid1")
	if err != nil {
		t.Fatalf("Matches filtered: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 match for pid1, got %d", len(filtered))
	}
	if filtered[0].ID != m1 {
		t.Errorf("expected match id %d, got %d", m1, filtered[0].ID)
	}
}

func TestPlayerAggregate(t *testing.T) {
	d := newTestDB(t)
	d.UpsertPlayer("pid1", "Alice")

	m1, _ := d.UpsertMatch("guid-ag1", "DFH Stadium", time.Now())
	m2, _ := d.UpsertMatch("guid-ag2", "Mannfield", time.Now())
	d.UpsertPlayerMatchStats(m1, "pid1", 0, 500, 3, 5, 1, 2, 10, 8, 1)
	d.UpsertPlayerMatchStats(m2, "pid1", 0, 400, 2, 3, 0, 1, 8, 6, 0)

	agg, err := d.PlayerAggregate("pid1")
	if err != nil {
		t.Fatalf("PlayerAggregate: %v", err)
	}
	if agg.Matches != 2 {
		t.Errorf("Matches: got %d, want 2", agg.Matches)
	}
	if agg.Goals != 5 {
		t.Errorf("Goals: got %d, want 5", agg.Goals)
	}
	if agg.Assists != 1 {
		t.Errorf("Assists: got %d, want 1", agg.Assists)
	}
	if agg.Saves != 3 {
		t.Errorf("Saves: got %d, want 3", agg.Saves)
	}
	if agg.Demos != 1 {
		t.Errorf("Demos: got %d, want 1", agg.Demos)
	}
	if agg.Touches != 18 {
		t.Errorf("Touches: got %d, want 18", agg.Touches)
	}
	if agg.Name != "Alice" {
		t.Errorf("Name: got %q, want Alice", agg.Name)
	}
}

func TestMatchPlayersMultipleTeams(t *testing.T) {
	d := newTestDB(t)
	d.UpsertPlayer("blue1", "Alice")
	d.UpsertPlayer("orange1", "Bob")
	matchID, _ := d.UpsertMatch("guid-teams", "DFH Stadium", time.Now())
	d.UpsertPlayerMatchStats(matchID, "blue1", 0, 500, 3, 5, 0, 0, 0, 0, 0)
	d.UpsertPlayerMatchStats(matchID, "orange1", 1, 300, 1, 2, 0, 1, 0, 0, 0)

	players, err := d.MatchPlayers(matchID)
	if err != nil {
		t.Fatalf("MatchPlayers: %v", err)
	}
	if len(players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(players))
	}
	// Results are ordered by team_num, then score DESC.
	if players[0].TeamNum != 0 {
		t.Errorf("first player should be team 0, got %d", players[0].TeamNum)
	}
	if players[1].TeamNum != 1 {
		t.Errorf("second player should be team 1, got %d", players[1].TeamNum)
	}
}

func TestAllTeamGoals(t *testing.T) {
	d := newTestDB(t)
	d.UpsertPlayer("blue1", "Alice")
	d.UpsertPlayer("blue2", "Bob")
	d.UpsertPlayer("orange1", "Carol")
	m1, _ := d.UpsertMatch("guid-tg1", "DFH Stadium", time.Now())
	m2, _ := d.UpsertMatch("guid-tg2", "Mannfield", time.Now())
	d.UpsertPlayerMatchStats(m1, "blue1", 0, 0, 2, 0, 0, 0, 0, 0, 0)
	d.UpsertPlayerMatchStats(m1, "blue2", 0, 0, 1, 0, 0, 0, 0, 0, 0)
	d.UpsertPlayerMatchStats(m1, "orange1", 1, 0, 3, 0, 0, 0, 0, 0, 0)
	d.UpsertPlayerMatchStats(m2, "blue1", 0, 0, 4, 0, 0, 0, 0, 0, 0)

	goals, err := d.AllTeamGoals()
	if err != nil {
		t.Fatalf("AllTeamGoals: %v", err)
	}
	if goals[m1][0] != 3 {
		t.Errorf("m1 team0 goals: got %d, want 3", goals[m1][0])
	}
	if goals[m1][1] != 3 {
		t.Errorf("m1 team1 goals: got %d, want 3", goals[m1][1])
	}
	if goals[m2][0] != 4 {
		t.Errorf("m2 team0 goals: got %d, want 4", goals[m2][0])
	}
}

func TestMatchPlayerCounts(t *testing.T) {
	d := newTestDB(t)
	d.UpsertPlayer("p1", "Alice")
	d.UpsertPlayer("p2", "Bob")
	d.UpsertPlayer("p3", "Carol")
	m1, _ := d.UpsertMatch("guid-mpc1", "DFH Stadium", time.Now())
	m2, _ := d.UpsertMatch("guid-mpc2", "Mannfield", time.Now())
	d.UpsertPlayerMatchStats(m1, "p1", 0, 0, 0, 0, 0, 0, 0, 0, 0)
	d.UpsertPlayerMatchStats(m1, "p2", 1, 0, 0, 0, 0, 0, 0, 0, 0)
	d.UpsertPlayerMatchStats(m2, "p3", 0, 0, 0, 0, 0, 0, 0, 0, 0)

	counts, err := d.MatchPlayerCounts()
	if err != nil {
		t.Fatalf("MatchPlayerCounts: %v", err)
	}
	if counts[m1] != 2 {
		t.Errorf("m1 player count: got %d, want 2", counts[m1])
	}
	if counts[m2] != 1 {
		t.Errorf("m2 player count: got %d, want 1", counts[m2])
	}
}

func TestTrackerCache(t *testing.T) {
	d := newTestDB(t)

	// Not found before insert.
	_, _, found, err := d.GetTrackerCache("pid-x")
	if err != nil {
		t.Fatalf("GetTrackerCache: %v", err)
	}
	if found {
		t.Error("expected not found before insert")
	}

	payload := `{"rating":1234}`
	if err := d.UpsertTrackerCache("pid-x", payload); err != nil {
		t.Fatalf("UpsertTrackerCache: %v", err)
	}

	data, fetchedAt, found, err := d.GetTrackerCache("pid-x")
	if err != nil {
		t.Fatalf("GetTrackerCache after insert: %v", err)
	}
	if !found {
		t.Fatal("expected found after insert")
	}
	if data != payload {
		t.Errorf("data: got %q, want %q", data, payload)
	}
	if fetchedAt.IsZero() {
		t.Error("fetchedAt should not be zero")
	}

	// Upsert should update the payload.
	updated := `{"rating":9999}`
	if err := d.UpsertTrackerCache("pid-x", updated); err != nil {
		t.Fatalf("UpsertTrackerCache update: %v", err)
	}
	data, _, _, _ = d.GetTrackerCache("pid-x")
	if data != updated {
		t.Errorf("updated data: got %q, want %q", data, updated)
	}
}

func TestAllBCUploads(t *testing.T) {
	d := newTestDB(t)

	uploads, err := d.AllBCUploads()
	if err != nil {
		t.Fatalf("AllBCUploads empty: %v", err)
	}
	if len(uploads) != 0 {
		t.Errorf("expected 0 uploads, got %d", len(uploads))
	}

	if err := d.UpsertBCUpload("match.replay", "bc-id-1", "https://ballchasing.com/replay/bc-id-1"); err != nil {
		t.Fatalf("UpsertBCUpload: %v", err)
	}
	if err := d.UpsertBCUpload("match2.replay", "bc-id-2", "https://ballchasing.com/replay/bc-id-2"); err != nil {
		t.Fatalf("UpsertBCUpload 2: %v", err)
	}

	uploads, err = d.AllBCUploads()
	if err != nil {
		t.Fatalf("AllBCUploads: %v", err)
	}
	if len(uploads) != 2 {
		t.Errorf("expected 2 uploads, got %d", len(uploads))
	}
	if u, ok := uploads["match.replay"]; !ok || u.BallchasingID != "bc-id-1" {
		t.Errorf("upload for match.replay: %+v", uploads["match.replay"])
	}

	// Upsert should update.
	if err := d.UpsertBCUpload("match.replay", "bc-id-updated", "https://ballchasing.com/replay/bc-id-updated"); err != nil {
		t.Fatalf("UpsertBCUpload update: %v", err)
	}
	uploads, _ = d.AllBCUploads()
	if uploads["match.replay"].BallchasingID != "bc-id-updated" {
		t.Errorf("expected updated id, got %q", uploads["match.replay"].BallchasingID)
	}
}
