package histstore_test

import (
	"path/filepath"
	"testing"
	"time"

	"OOF_RL/internal/db"
	"OOF_RL/internal/histstore"
)

func newTestStore(t *testing.T) *histstore.Store {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := histstore.Migrate(database); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return histstore.NewStore(database)
}

func TestUpsertAndFetchPlayer(t *testing.T) {
	s := newTestStore(t)

	if err := s.UpsertPlayer("pid1", "Alice"); err != nil {
		t.Fatalf("UpsertPlayer: %v", err)
	}
	players, err := s.AllPlayers()
	if err != nil {
		t.Fatalf("AllPlayers: %v", err)
	}
	if len(players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(players))
	}
	if players[0].PrimaryID != "pid1" || players[0].Name != "Alice" {
		t.Errorf("unexpected player: %+v", players[0])
	}

	if err := s.UpsertPlayer("pid1", "Alice Updated"); err != nil {
		t.Fatalf("UpsertPlayer update: %v", err)
	}
	players, _ = s.AllPlayers()
	if players[0].Name != "Alice Updated" {
		t.Errorf("expected updated name, got %q", players[0].Name)
	}
}

func TestUpsertMatchNewAndDuplicate(t *testing.T) {
	s := newTestStore(t)

	id, err := s.UpsertMatch("guid-1", "DFH Stadium", time.Now())
	if err != nil {
		t.Fatalf("UpsertMatch: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}

	id2, err := s.UpsertMatch("guid-1", "DFH Stadium", time.Now())
	if err != nil {
		t.Fatalf("UpsertMatch duplicate: %v", err)
	}
	if id2 != id {
		t.Errorf("expected same id %d on duplicate upsert, got %d", id, id2)
	}
}

func TestUpsertMatchEmptyGUID(t *testing.T) {
	s := newTestStore(t)
	id, err := s.UpsertMatch("", "DFH Stadium", time.Now())
	if err != nil {
		t.Fatalf("UpsertMatch empty guid: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}
}

func TestEndMatch(t *testing.T) {
	s := newTestStore(t)

	id, _ := s.UpsertMatch("guid-end", "Mannfield", time.Now())
	if err := s.EndMatch(id, 1, true, false, false); err != nil {
		t.Fatalf("EndMatch: %v", err)
	}

	matches, err := s.Matches("")
	if err != nil {
		t.Fatalf("matches: %v", err)
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
	s := newTestStore(t)
	s.UpsertPlayer("pid1", "Alice")
	matchID, _ := s.UpsertMatch("guid-stats", "Wasteland", time.Now())

	if err := s.UpsertPlayerMatchStats(matchID, "pid1", 0, 500, 3, 5, 1, 2, 10, 8, 1); err != nil {
		t.Fatalf("UpsertPlayerMatchStats: %v", err)
	}

	players, err := s.MatchPlayers(matchID)
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
	s := newTestStore(t)
	s.UpsertPlayer("pid1", "Alice")
	matchID, _ := s.UpsertMatch("guid-idem", "Neo Tokyo", time.Now())

	s.UpsertPlayerMatchStats(matchID, "pid1", 0, 500, 3, 5, 1, 2, 10, 8, 1)
	if err := s.UpsertPlayerMatchStats(matchID, "pid1", 0, 600, 4, 6, 2, 3, 11, 9, 2); err != nil {
		t.Fatalf("second UpsertPlayerMatchStats: %v", err)
	}
	players, _ := s.MatchPlayers(matchID)
	if players[0].Goals != 4 {
		t.Errorf("expected updated goals=4, got %d", players[0].Goals)
	}
}

func TestMatchPlayersPrefersPerMatchNameSnapshot(t *testing.T) {
	s := newTestStore(t)
	s.UpsertPlayer("pid1", "Old Name")
	matchID, _ := s.UpsertMatch("guid-name-snapshot", "Mannfield", time.Now())

	if err := s.UpsertPlayerMatchStatsWithName(matchID, "pid1", "Old Name", 0, 500, 3, 5, 1, 2, 10, 8, 1); err != nil {
		t.Fatalf("UpsertPlayerMatchStatsWithName: %v", err)
	}
	if err := s.UpsertPlayer("pid1", "New Name"); err != nil {
		t.Fatalf("UpsertPlayer rename: %v", err)
	}

	players, err := s.MatchPlayers(matchID)
	if err != nil {
		t.Fatalf("MatchPlayers: %v", err)
	}
	if len(players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(players))
	}
	if players[0].Name != "Old Name" {
		t.Errorf("Name: got %q, want Old Name", players[0].Name)
	}
}

func TestMatchPlayersFallsBackToLatestPlayerName(t *testing.T) {
	s := newTestStore(t)
	s.UpsertPlayer("pid1", "Current Name")
	matchID, _ := s.UpsertMatch("guid-name-fallback", "Mannfield", time.Now())

	if err := s.UpsertPlayerMatchStats(matchID, "pid1", 0, 500, 3, 5, 1, 2, 10, 8, 1); err != nil {
		t.Fatalf("UpsertPlayerMatchStats: %v", err)
	}

	players, err := s.MatchPlayers(matchID)
	if err != nil {
		t.Fatalf("MatchPlayers: %v", err)
	}
	if len(players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(players))
	}
	if players[0].Name != "Current Name" {
		t.Errorf("Name: got %q, want Current Name", players[0].Name)
	}
}

func TestInsertAndFetchGoal(t *testing.T) {
	s := newTestStore(t)
	s.UpsertPlayer("scorer1", "Bob")
	s.UpsertPlayer("assist1", "Carol")
	s.UpsertPlayer("touch1", "Dave")
	matchID, _ := s.UpsertMatch("guid-goal", "Beckwith Park", time.Now())

	if err := s.InsertGoal(matchID, "scorer1", "Bob", "assist1", "Carol", "touch1", 120.5, 45.0, 1.0, 2.0, 3.0); err != nil {
		t.Fatalf("InsertGoal: %v", err)
	}

	goals, err := s.MatchGoals(matchID)
	if err != nil {
		t.Fatalf("MatchGoals: %v", err)
	}
	if len(goals) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(goals))
	}
	g := goals[0]
	if g.ScorerID != "scorer1" || g.ScorerName != "Bob" {
		t.Errorf("scorer: got %q/%q", g.ScorerID, g.ScorerName)
	}
	if g.AssisterID != "assist1" || g.AssisterName != "Carol" {
		t.Errorf("assister: got %q/%q", g.AssisterID, g.AssisterName)
	}
	if g.GoalSpeed != 120.5 {
		t.Errorf("GoalSpeed: got %f, want 120.5", g.GoalSpeed)
	}
	if g.ImpactX != 1.0 || g.ImpactY != 2.0 || g.ImpactZ != 3.0 {
		t.Errorf("impact: got (%f,%f,%f)", g.ImpactX, g.ImpactY, g.ImpactZ)
	}
}

func TestInsertGoalNoAssister(t *testing.T) {
	s := newTestStore(t)
	s.UpsertPlayer("scorer1", "Bob")
	matchID, _ := s.UpsertMatch("guid-noassist", "Aquadome", time.Now())

	if err := s.InsertGoal(matchID, "scorer1", "Bob", "", "", "", 100.0, 60.0, 0, 0, 0); err != nil {
		t.Fatalf("InsertGoal without assister: %v", err)
	}
	goals, _ := s.MatchGoals(matchID)
	if goals[0].AssisterID != "" {
		t.Errorf("expected empty AssisterID, got %q", goals[0].AssisterID)
	}
}

func TestInsertBallHit(t *testing.T) {
	s := newTestStore(t)
	s.UpsertPlayer("pid1", "Eve")
	matchID, _ := s.UpsertMatch("guid-bh", "Urban Central", time.Now())

	if err := s.InsertBallHit(matchID, "pid1", 50.0, 80.0, 10.0, 20.0, 30.0); err != nil {
		t.Fatalf("InsertBallHit: %v", err)
	}
}

func TestMatchesFilter(t *testing.T) {
	s := newTestStore(t)
	s.UpsertPlayer("pid1", "Alice")
	s.UpsertPlayer("pid2", "Bob")

	m1, _ := s.UpsertMatch("guid-m1", "DFH Stadium", time.Now())
	m2, _ := s.UpsertMatch("guid-m2", "Mannfield", time.Now())
	s.UpsertPlayerMatchStats(m1, "pid1", 0, 100, 1, 1, 0, 0, 0, 0, 0)
	s.UpsertPlayerMatchStats(m2, "pid2", 1, 200, 2, 2, 0, 0, 0, 0, 0)

	all, err := s.Matches("")
	if err != nil {
		t.Fatalf("Matches all: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 matches, got %d", len(all))
	}

	filtered, err := s.Matches("pid1")
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
	s := newTestStore(t)
	s.UpsertPlayer("pid1", "Alice")

	m1, _ := s.UpsertMatch("guid-ag1", "DFH Stadium", time.Now())
	m2, _ := s.UpsertMatch("guid-ag2", "Mannfield", time.Now())
	s.UpsertPlayerMatchStats(m1, "pid1", 0, 500, 3, 5, 1, 2, 10, 8, 1)
	s.UpsertPlayerMatchStats(m2, "pid1", 0, 400, 2, 3, 0, 1, 8, 6, 0)

	agg, err := s.PlayerAggregate("pid1")
	if err != nil {
		t.Fatalf("PlayerAggregate: %v", err)
	}
	if agg.Matches != 2 || agg.Goals != 5 || agg.Assists != 1 || agg.Saves != 3 || agg.Demos != 1 || agg.Touches != 18 {
		t.Errorf("unexpected aggregate: %+v", agg)
	}
	if agg.Name != "Alice" {
		t.Errorf("Name: got %q, want Alice", agg.Name)
	}
}

func TestMatchPlayersMultipleTeams(t *testing.T) {
	s := newTestStore(t)
	s.UpsertPlayer("blue1", "Alice")
	s.UpsertPlayer("orange1", "Bob")
	matchID, _ := s.UpsertMatch("guid-teams", "DFH Stadium", time.Now())
	s.UpsertPlayerMatchStats(matchID, "blue1", 0, 500, 3, 5, 0, 0, 0, 0, 0)
	s.UpsertPlayerMatchStats(matchID, "orange1", 1, 300, 1, 2, 0, 1, 0, 0, 0)

	players, err := s.MatchPlayers(matchID)
	if err != nil {
		t.Fatalf("MatchPlayers: %v", err)
	}
	if len(players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(players))
	}
	if players[0].TeamNum != 0 || players[1].TeamNum != 1 {
		t.Errorf("wrong team order: %d, %d", players[0].TeamNum, players[1].TeamNum)
	}
}

func TestAllTeamGoals(t *testing.T) {
	s := newTestStore(t)
	s.UpsertPlayer("blue1", "Alice")
	s.UpsertPlayer("blue2", "Bob")
	s.UpsertPlayer("orange1", "Carol")
	m1, _ := s.UpsertMatch("guid-tg1", "DFH Stadium", time.Now())
	m2, _ := s.UpsertMatch("guid-tg2", "Mannfield", time.Now())
	s.UpsertPlayerMatchStats(m1, "blue1", 0, 0, 2, 0, 0, 0, 0, 0, 0)
	s.UpsertPlayerMatchStats(m1, "blue2", 0, 0, 1, 0, 0, 0, 0, 0, 0)
	s.UpsertPlayerMatchStats(m1, "orange1", 1, 0, 3, 0, 0, 0, 0, 0, 0)
	s.UpsertPlayerMatchStats(m2, "blue1", 0, 0, 4, 0, 0, 0, 0, 0, 0)

	goals, err := s.AllTeamGoals()
	if err != nil {
		t.Fatalf("AllTeamGoals: %v", err)
	}
	if goals[m1][0] != 3 || goals[m1][1] != 3 || goals[m2][0] != 4 {
		t.Errorf("unexpected goals: m1=%v m2=%v", goals[m1], goals[m2])
	}
}

func TestMatchPlayerCounts(t *testing.T) {
	s := newTestStore(t)
	s.UpsertPlayer("p1", "Alice")
	s.UpsertPlayer("p2", "Bob")
	s.UpsertPlayer("p3", "Carol")
	m1, _ := s.UpsertMatch("guid-mpc1", "DFH Stadium", time.Now())
	m2, _ := s.UpsertMatch("guid-mpc2", "Mannfield", time.Now())
	s.UpsertPlayerMatchStats(m1, "p1", 0, 0, 0, 0, 0, 0, 0, 0, 0)
	s.UpsertPlayerMatchStats(m1, "p2", 1, 0, 0, 0, 0, 0, 0, 0, 0)
	s.UpsertPlayerMatchStats(m2, "p3", 0, 0, 0, 0, 0, 0, 0, 0, 0)

	counts, err := s.MatchPlayerCounts()
	if err != nil {
		t.Fatalf("MatchPlayerCounts: %v", err)
	}
	if counts[m1] != 2 || counts[m2] != 1 {
		t.Errorf("unexpected counts: %v", counts)
	}
}

func TestMatchBotCounts(t *testing.T) {
	s := newTestStore(t)
	s.UpsertPlayer("p1", "Alice")
	s.UpsertPlayer("bot:guid-bots:5", "Gerwin")
	s.UpsertPlayer("bot:guid:7", "Foamer")
	s.UpsertPlayer("steam|botnamedplayer|0", "Human With Bot Name")

	m1, _ := s.UpsertMatch("guid-bots", "Utopia Coliseum", time.Now())
	m2, _ := s.UpsertMatch("guid-human", "Mannfield", time.Now())
	s.UpsertPlayerMatchStats(m1, "p1", 0, 0, 0, 0, 0, 0, 0, 0, 0)
	s.UpsertPlayerMatchStats(m1, "bot:guid-bots:5", 1, 0, 0, 0, 0, 0, 0, 0, 0)
	s.UpsertPlayerMatchStats(m1, "bot:guid:7", 1, 0, 0, 0, 0, 0, 0, 0, 0)
	s.UpsertPlayerMatchStats(m2, "steam|botnamedplayer|0", 0, 0, 0, 0, 0, 0, 0, 0, 0)

	counts, err := s.MatchBotCounts()
	if err != nil {
		t.Fatalf("MatchBotCounts: %v", err)
	}
	if counts[m1] != 2 {
		t.Errorf("bot match count: got %d, want 2", counts[m1])
	}
	if counts[m2] != 0 {
		t.Errorf("human match count: got %d, want 0", counts[m2])
	}
}

func TestUpdateMatchPlaylist(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.UpsertMatch("guid-pl", "DFH Stadium", time.Now())

	if err := s.UpdateMatchPlaylist(id, 3); err != nil {
		t.Fatalf("UpdateMatchPlaylist: %v", err)
	}
	m, err := s.MatchByID(id)
	if err != nil {
		t.Fatalf("MatchByID: %v", err)
	}
	if m == nil || m.PlaylistType == nil || *m.PlaylistType != 3 {
		t.Errorf("expected PlaylistType=3, got %v", m)
	}
}

func TestMatchByIDNotFound(t *testing.T) {
	s := newTestStore(t)
	m, err := s.MatchByID(9999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m != nil {
		t.Errorf("expected nil for missing ID, got %+v", m)
	}
}

func TestMatchByIDWithAllNullableFields(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.UpsertMatch("guid-nullable", "Champions Field", time.Now())
	s.EndMatch(id, 0, false, false, false)
	s.UpdateTeamScores(id, 2, 1)
	s.UpdateMatchPlaylist(id, 5)

	m, err := s.MatchByID(id)
	if err != nil {
		t.Fatalf("MatchByID: %v", err)
	}
	if m == nil {
		t.Fatal("expected match, got nil")
	}
	if m.TeamScore0 == nil || *m.TeamScore0 != 2 {
		t.Errorf("TeamScore0: got %v, want 2", m.TeamScore0)
	}
	if m.TeamScore1 == nil || *m.TeamScore1 != 1 {
		t.Errorf("TeamScore1: got %v, want 1", m.TeamScore1)
	}
	if m.PlaylistType == nil || *m.PlaylistType != 5 {
		t.Errorf("PlaylistType: got %v, want 5", m.PlaylistType)
	}
}

func TestInsertAndFetchStatfeedEvent(t *testing.T) {
	s := newTestStore(t)
	s.UpsertPlayer("pid1", "Alice")
	matchID, _ := s.UpsertMatch("guid-sf", "Utopia Coliseum", time.Now())

	if err := s.InsertStatfeedEvent(matchID, "pid1", "Alice", 0, "Save", "", ""); err != nil {
		t.Fatalf("InsertStatfeedEvent: %v", err)
	}
	if err := s.InsertStatfeedEvent(matchID, "pid1", "Alice", 0, "Demo", "pid2", "Bob"); err != nil {
		t.Fatalf("InsertStatfeedEvent with target: %v", err)
	}

	evts, err := s.MatchStatfeedEvents(matchID)
	if err != nil {
		t.Fatalf("MatchStatfeedEvents: %v", err)
	}
	if len(evts) != 2 {
		t.Fatalf("expected 2 events, got %d", len(evts))
	}
	if evts[0].EventType != "Save" || evts[0].PlayerName != "Alice" {
		t.Errorf("first event: type=%q name=%q", evts[0].EventType, evts[0].PlayerName)
	}
	if evts[1].EventType != "Demo" || evts[1].TargetName != "Bob" {
		t.Errorf("second event: type=%q target=%q", evts[1].EventType, evts[1].TargetName)
	}
}
