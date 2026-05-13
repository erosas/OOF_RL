package oofevents_test

import (
	"testing"

	"OOF_RL/internal/oofevents"
)

func TestCertaintyString(t *testing.T) {
	cases := []struct {
		c    oofevents.Certainty
		want string
	}{
		{oofevents.Authoritative, "authoritative"},
		{oofevents.Inferred, "inferred"},
		{oofevents.Signal, "signal"},
		{oofevents.Certainty(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.c.String(); got != tc.want {
			t.Errorf("Certainty(%d).String() = %q, want %q", tc.c, got, tc.want)
		}
	}
}

func TestNewBaseFields(t *testing.T) {
	b := oofevents.NewBase("test.type", oofevents.Authoritative, "guid1")
	if b.Type() != "test.type" {
		t.Errorf("type %q, want test.type", b.Type())
	}
	if b.MatchGUID() != "guid1" {
		t.Errorf("guid %q, want guid1", b.MatchGUID())
	}
	if b.Certainty() != oofevents.Authoritative {
		t.Errorf("certainty %s, want authoritative", b.Certainty())
	}
	if b.OccurredAt().IsZero() {
		t.Error("OccurredAt is zero")
	}
	if b.Source() != (oofevents.Source{}) {
		t.Errorf("source %+v, want empty", b.Source())
	}
}

func TestNewMatchStarted(t *testing.T) {
	e := oofevents.NewMatchStarted("g1")
	if e.Type() != oofevents.TypeMatchStarted {
		t.Errorf("type %q", e.Type())
	}
	if e.MatchGUID() != "g1" {
		t.Errorf("guid %q", e.MatchGUID())
	}
	if e.Certainty() != oofevents.Authoritative {
		t.Errorf("certainty %s", e.Certainty())
	}
}

func TestNewMatchEnded(t *testing.T) {
	e := oofevents.NewMatchEnded("g1", 1)
	if e.Type() != oofevents.TypeMatchEnded {
		t.Errorf("type %q", e.Type())
	}
	if e.WinnerTeamNum != 1 {
		t.Errorf("winner team %d, want 1", e.WinnerTeamNum)
	}
}

func TestNewMatchDestroyed(t *testing.T) {
	e := oofevents.NewMatchDestroyed()
	if e.Type() != oofevents.TypeMatchDestroyed {
		t.Errorf("type %q", e.Type())
	}
	if e.MatchGUID() != "" {
		t.Errorf("guid %q, want empty", e.MatchGUID())
	}
}

func TestNewMatchRestarted(t *testing.T) {
	e := oofevents.NewMatchRestarted("g2", "g1")
	if e.Type() != oofevents.TypeMatchRestarted {
		t.Errorf("type %q", e.Type())
	}
	if e.PreviousGUID != "g1" {
		t.Errorf("prev guid %q, want g1", e.PreviousGUID)
	}
	if e.MatchGUID() != "g2" {
		t.Errorf("new guid %q, want g2", e.MatchGUID())
	}
	if e.Certainty() != oofevents.Inferred {
		t.Errorf("certainty %s, want inferred", e.Certainty())
	}
}

func TestNewOvertimeStarted(t *testing.T) {
	e := oofevents.NewOvertimeStarted("g1", 5)
	if e.Type() != oofevents.TypeOvertimeStarted {
		t.Errorf("type %q", e.Type())
	}
	if e.ClockSeconds != 5 {
		t.Errorf("clock seconds %d, want 5", e.ClockSeconds)
	}
	if e.Certainty() != oofevents.Inferred {
		t.Errorf("certainty %s, want inferred", e.Certainty())
	}
}

func TestNewGoalScored(t *testing.T) {
	e := oofevents.NewGoalScored("g1", "Alice", 5, "Bob", 7, 3, 120.0, 90.0, 1.0, 2.0, 3.0, 0)
	if e.Type() != oofevents.TypeGoalScored {
		t.Errorf("type %q", e.Type())
	}
	if e.Scorer != "Alice" || e.ScorerShortcut != 5 {
		t.Errorf("scorer: name=%q sc=%d", e.Scorer, e.ScorerShortcut)
	}
	if e.Assister != "Bob" || e.AssisterShortcut != 7 {
		t.Errorf("assister: name=%q sc=%d", e.Assister, e.AssisterShortcut)
	}
	if e.LastTouchShortcut != 3 {
		t.Errorf("last touch shortcut %d, want 3", e.LastTouchShortcut)
	}
	if e.GoalSpeed != 120.0 || e.GoalTime != 90.0 {
		t.Errorf("goal speed/time: %f/%f", e.GoalSpeed, e.GoalTime)
	}
	if e.ImpactX != 1.0 || e.ImpactY != 2.0 || e.ImpactZ != 3.0 {
		t.Errorf("impact (%f,%f,%f), want (1,2,3)", e.ImpactX, e.ImpactY, e.ImpactZ)
	}
}

func TestNewStatFeed(t *testing.T) {
	e := oofevents.NewStatFeed("g1", "Save", "Alice", 3, 1, "Bob", 8)
	if e.Type() != oofevents.TypeStatFeed {
		t.Errorf("type %q", e.Type())
	}
	if e.EventName != "Save" {
		t.Errorf("event name %q, want Save", e.EventName)
	}
	if e.MainTarget != "Alice" || e.MainTargetShortcut != 3 || e.MainTargetTeamNum != 1 {
		t.Errorf("main target: name=%q sc=%d team=%d", e.MainTarget, e.MainTargetShortcut, e.MainTargetTeamNum)
	}
	if e.SecondaryTarget != "Bob" || e.SecondaryTargetShortcut != 8 {
		t.Errorf("secondary: name=%q sc=%d", e.SecondaryTarget, e.SecondaryTargetShortcut)
	}
}

func TestNewClockUpdated(t *testing.T) {
	e := oofevents.NewClockUpdated("g1", 180, true)
	if e.Type() != oofevents.TypeClockUpdated {
		t.Errorf("type %q", e.Type())
	}
	if e.TimeSeconds != 180 {
		t.Errorf("time %d, want 180", e.TimeSeconds)
	}
	if !e.IsOvertime {
		t.Error("expected overtime=true")
	}
}

func TestNewBallHit(t *testing.T) {
	e := oofevents.NewBallHit("g1", "Alice", "steam|1", 2, 10.0, 50.0, 1.0, 2.0, 3.0)
	if e.Type() != oofevents.TypeBallHit {
		t.Errorf("type %q", e.Type())
	}
	if e.PlayerName != "Alice" || e.PlayerPrimaryID != "steam|1" || e.PlayerShortcut != 2 {
		t.Errorf("player: name=%q id=%q sc=%d", e.PlayerName, e.PlayerPrimaryID, e.PlayerShortcut)
	}
	if e.PreHitSpeed != 10.0 || e.PostHitSpeed != 50.0 {
		t.Errorf("speeds: pre=%f post=%f", e.PreHitSpeed, e.PostHitSpeed)
	}
	if e.LocX != 1.0 || e.LocY != 2.0 || e.LocZ != 3.0 {
		t.Errorf("location (%f,%f,%f), want (1,2,3)", e.LocX, e.LocY, e.LocZ)
	}
}

func TestNewCrossbarHit(t *testing.T) {
	e := oofevents.NewCrossbarHit("g1", "Alice", 200.0, 50.0)
	if e.Type() != oofevents.TypeCrossbarHit {
		t.Errorf("type %q", e.Type())
	}
	if e.LastToucher != "Alice" {
		t.Errorf("last toucher %q, want Alice", e.LastToucher)
	}
	if e.BallSpeed != 200.0 || e.ImpactForce != 50.0 {
		t.Errorf("speed=%f force=%f", e.BallSpeed, e.ImpactForce)
	}
}

func TestNewStateUpdated(t *testing.T) {
	players := []oofevents.PlayerSnapshot{{Name: "Alice"}}
	game := oofevents.GameSnapshot{Arena: "DFH Stadium"}
	e := oofevents.NewStateUpdated("g1", players, game)
	if e.Type() != oofevents.TypeStateUpdated {
		t.Errorf("type %q", e.Type())
	}
	if len(e.Players) != 1 || e.Players[0].Name != "Alice" {
		t.Errorf("players: %+v", e.Players)
	}
	if e.Game.Arena != "DFH Stadium" {
		t.Errorf("arena %q, want DFH Stadium", e.Game.Arena)
	}
}