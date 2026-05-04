package history

import (
	"path/filepath"
	"testing"
	"time"

	"OOF_RL/internal/db"
)

func newTestStore(t *testing.T) *store {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := database.RunMigration(historySchema); err != nil {
		t.Fatalf("RunMigration: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return &store{conn: database.Conn()}
}

func TestUpsertAndFetchPlayer(t *testing.T) {
	s := newTestStore(t)

	if err := s.upsertPlayer("pid1", "Alice"); err != nil {
		t.Fatalf("upsertPlayer: %v", err)
	}
	players, err := s.allPlayers()
	if err != nil {
		t.Fatalf("allPlayers: %v", err)
	}
	if len(players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(players))
	}
	if players[0].PrimaryID != "pid1" || players[0].Name != "Alice" {
		t.Errorf("unexpected player: %+v", players[0])
	}

	if err := s.upsertPlayer("pid1", "Alice Updated"); err != nil {
		t.Fatalf("upsertPlayer update: %v", err)
	}
	players, _ = s.allPlayers()
	if players[0].Name != "Alice Updated" {
		t.Errorf("expected updated name, got %q", players[0].Name)
	}
}

func TestUpsertMatchNewAndDuplicate(t *testing.T) {
	s := newTestStore(t)

	id, err := s.upsertMatch("guid-1", "DFH Stadium", time.Now())
	if err != nil {
		t.Fatalf("upsertMatch: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}

	id2, err := s.upsertMatch("guid-1", "DFH Stadium", time.Now())
	if err != nil {
		t.Fatalf("upsertMatch duplicate: %v", err)
	}
	if id2 != id {
		t.Errorf("expected same id %d on duplicate upsert, got %d", id, id2)
	}
}

func TestUpsertMatchEmptyGUID(t *testing.T) {
	s := newTestStore(t)
	id, err := s.upsertMatch("", "DFH Stadium", time.Now())
	if err != nil {
		t.Fatalf("upsertMatch empty guid: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}
}

func TestEndMatch(t *testing.T) {
	s := newTestStore(t)

	id, _ := s.upsertMatch("guid-end", "Mannfield", time.Now())
	if err := s.endMatch(id, 1, true, false, false); err != nil {
		t.Fatalf("endMatch: %v", err)
	}

	matches, err := s.matches("")
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
	s.upsertPlayer("pid1", "Alice")
	matchID, _ := s.upsertMatch("guid-stats", "Wasteland", time.Now())

	if err := s.upsertPlayerMatchStats(matchID, "pid1", 0, 500, 3, 5, 1, 2, 10, 8, 1); err != nil {
		t.Fatalf("upsertPlayerMatchStats: %v", err)
	}

	players, err := s.matchPlayers(matchID)
	if err != nil {
		t.Fatalf("matchPlayers: %v", err)
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
	s.upsertPlayer("pid1", "Alice")
	matchID, _ := s.upsertMatch("guid-idem", "Neo Tokyo", time.Now())

	s.upsertPlayerMatchStats(matchID, "pid1", 0, 500, 3, 5, 1, 2, 10, 8, 1)
	if err := s.upsertPlayerMatchStats(matchID, "pid1", 0, 600, 4, 6, 2, 3, 11, 9, 2); err != nil {
		t.Fatalf("second upsertPlayerMatchStats: %v", err)
	}
	players, _ := s.matchPlayers(matchID)
	if players[0].Goals != 4 {
		t.Errorf("expected updated goals=4, got %d", players[0].Goals)
	}
}

func TestInsertAndFetchGoal(t *testing.T) {
	s := newTestStore(t)
	s.upsertPlayer("scorer1", "Bob")
	s.upsertPlayer("assist1", "Carol")
	s.upsertPlayer("touch1", "Dave")
	matchID, _ := s.upsertMatch("guid-goal", "Beckwith Park", time.Now())

	if err := s.insertGoal(matchID, "scorer1", "Bob", "assist1", "Carol", "touch1", 120.5, 45.0, 1.0, 2.0, 3.0); err != nil {
		t.Fatalf("insertGoal: %v", err)
	}

	goals, err := s.matchGoals(matchID)
	if err != nil {
		t.Fatalf("matchGoals: %v", err)
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
	s.upsertPlayer("scorer1", "Bob")
	matchID, _ := s.upsertMatch("guid-noassist", "Aquadome", time.Now())

	if err := s.insertGoal(matchID, "scorer1", "Bob", "", "", "", 100.0, 60.0, 0, 0, 0); err != nil {
		t.Fatalf("insertGoal without assister: %v", err)
	}
	goals, _ := s.matchGoals(matchID)
	if goals[0].AssisterID != "" {
		t.Errorf("expected empty AssisterID, got %q", goals[0].AssisterID)
	}
}

func TestInsertBallHit(t *testing.T) {
	s := newTestStore(t)
	s.upsertPlayer("pid1", "Eve")
	matchID, _ := s.upsertMatch("guid-bh", "Urban Central", time.Now())

	if err := s.insertBallHit(matchID, "pid1", 50.0, 80.0, 10.0, 20.0, 30.0); err != nil {
		t.Fatalf("insertBallHit: %v", err)
	}
}

func TestInsertTick(t *testing.T) {
	s := newTestStore(t)
	matchID, _ := s.upsertMatch("guid-tick", "Champions Field", time.Now())

	if err := s.insertTick(matchID, `{"Game":{}}`); err != nil {
		t.Fatalf("insertTick: %v", err)
	}
}

func TestMatchesFilter(t *testing.T) {
	s := newTestStore(t)
	s.upsertPlayer("pid1", "Alice")
	s.upsertPlayer("pid2", "Bob")

	m1, _ := s.upsertMatch("guid-m1", "DFH Stadium", time.Now())
	m2, _ := s.upsertMatch("guid-m2", "Mannfield", time.Now())
	s.upsertPlayerMatchStats(m1, "pid1", 0, 100, 1, 1, 0, 0, 0, 0, 0)
	s.upsertPlayerMatchStats(m2, "pid2", 1, 200, 2, 2, 0, 0, 0, 0, 0)

	all, err := s.matches("")
	if err != nil {
		t.Fatalf("matches all: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 matches, got %d", len(all))
	}

	filtered, err := s.matches("pid1")
	if err != nil {
		t.Fatalf("matches filtered: %v", err)
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
	s.upsertPlayer("pid1", "Alice")

	m1, _ := s.upsertMatch("guid-ag1", "DFH Stadium", time.Now())
	m2, _ := s.upsertMatch("guid-ag2", "Mannfield", time.Now())
	s.upsertPlayerMatchStats(m1, "pid1", 0, 500, 3, 5, 1, 2, 10, 8, 1)
	s.upsertPlayerMatchStats(m2, "pid1", 0, 400, 2, 3, 0, 1, 8, 6, 0)

	agg, err := s.playerAggregate("pid1")
	if err != nil {
		t.Fatalf("playerAggregate: %v", err)
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
	s.upsertPlayer("blue1", "Alice")
	s.upsertPlayer("orange1", "Bob")
	matchID, _ := s.upsertMatch("guid-teams", "DFH Stadium", time.Now())
	s.upsertPlayerMatchStats(matchID, "blue1", 0, 500, 3, 5, 0, 0, 0, 0, 0)
	s.upsertPlayerMatchStats(matchID, "orange1", 1, 300, 1, 2, 0, 1, 0, 0, 0)

	players, err := s.matchPlayers(matchID)
	if err != nil {
		t.Fatalf("matchPlayers: %v", err)
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
	s.upsertPlayer("blue1", "Alice")
	s.upsertPlayer("blue2", "Bob")
	s.upsertPlayer("orange1", "Carol")
	m1, _ := s.upsertMatch("guid-tg1", "DFH Stadium", time.Now())
	m2, _ := s.upsertMatch("guid-tg2", "Mannfield", time.Now())
	s.upsertPlayerMatchStats(m1, "blue1", 0, 0, 2, 0, 0, 0, 0, 0, 0)
	s.upsertPlayerMatchStats(m1, "blue2", 0, 0, 1, 0, 0, 0, 0, 0, 0)
	s.upsertPlayerMatchStats(m1, "orange1", 1, 0, 3, 0, 0, 0, 0, 0, 0)
	s.upsertPlayerMatchStats(m2, "blue1", 0, 0, 4, 0, 0, 0, 0, 0, 0)

	goals, err := s.allTeamGoals()
	if err != nil {
		t.Fatalf("allTeamGoals: %v", err)
	}
	if goals[m1][0] != 3 || goals[m1][1] != 3 || goals[m2][0] != 4 {
		t.Errorf("unexpected goals: m1=%v m2=%v", goals[m1], goals[m2])
	}
}

func TestMatchPlayerCounts(t *testing.T) {
	s := newTestStore(t)
	s.upsertPlayer("p1", "Alice")
	s.upsertPlayer("p2", "Bob")
	s.upsertPlayer("p3", "Carol")
	m1, _ := s.upsertMatch("guid-mpc1", "DFH Stadium", time.Now())
	m2, _ := s.upsertMatch("guid-mpc2", "Mannfield", time.Now())
	s.upsertPlayerMatchStats(m1, "p1", 0, 0, 0, 0, 0, 0, 0, 0, 0)
	s.upsertPlayerMatchStats(m1, "p2", 1, 0, 0, 0, 0, 0, 0, 0, 0)
	s.upsertPlayerMatchStats(m2, "p3", 0, 0, 0, 0, 0, 0, 0, 0, 0)

	counts, err := s.matchPlayerCounts()
	if err != nil {
		t.Fatalf("matchPlayerCounts: %v", err)
	}
	if counts[m1] != 2 || counts[m2] != 1 {
		t.Errorf("unexpected counts: %v", counts)
	}
}
