package momentum

import (
	"testing"

	"OOF_RL/internal/oofevents"
)

func TestApplyGameActionShotUpdatesRuntimeSignal(t *testing.T) {
	engine := NewEngine(Config{Decay: 1})

	state := engine.ApplyGameAction(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice"))

	blue := state.Teams[oofevents.TeamBlue]
	orange := state.Teams[oofevents.TeamOrange]
	if state.MatchGUID != "match-1" {
		t.Fatalf("MatchGUID = %q, want match-1", state.MatchGUID)
	}
	if state.Sequence != 1 {
		t.Fatalf("Sequence = %d, want 1", state.Sequence)
	}
	if blue.Pressure <= orange.Pressure {
		t.Fatalf("blue pressure %f should exceed orange pressure %f", blue.Pressure, orange.Pressure)
	}
	if blue.MomentumInfluence <= 0 || blue.EventDerivedControl <= 0 || blue.Confidence <= 0 {
		t.Fatalf("blue signal not populated: %+v", blue)
	}
	if state.LastEvent.Action != oofevents.ActionShot || state.LastEvent.ImpactTeam != oofevents.TeamBlue {
		t.Fatalf("last event = %+v, want blue shot impact", state.LastEvent)
	}
}

func TestApplyGameActionSupportsAllV1Kinds(t *testing.T) {
	cases := []struct {
		name       string
		event      oofevents.GameActionEvent
		impactTeam oofevents.Team
	}{
		{
			name:       "ball hit",
			event:      oofevents.NewGameAction("match-1", oofevents.ActionBallHit, oofevents.TeamBlue, "pid-a", "Alice"),
			impactTeam: oofevents.TeamBlue,
		},
		{
			name:       "shot",
			event:      oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice"),
			impactTeam: oofevents.TeamBlue,
		},
		{
			name:       "save",
			event:      oofevents.NewGameAction("match-1", oofevents.ActionSave, oofevents.TeamOrange, "pid-b", "Bob"),
			impactTeam: oofevents.TeamOrange,
		},
		{
			name:       "epic save",
			event:      oofevents.NewGameAction("match-1", oofevents.ActionSave, oofevents.TeamOrange, "pid-b", "Bob", oofevents.WithEpicSave()),
			impactTeam: oofevents.TeamOrange,
		},
		{
			name:       "goal",
			event:      oofevents.NewGameAction("match-1", oofevents.ActionGoal, oofevents.TeamBlue, "pid-a", "Alice"),
			impactTeam: oofevents.TeamBlue,
		},
		{
			name:       "own goal",
			event:      oofevents.NewGameAction("match-1", oofevents.ActionGoal, oofevents.TeamBlue, "pid-a", "Alice", oofevents.WithOwnGoal()),
			impactTeam: oofevents.TeamOrange,
		},
		{
			name:       "assist",
			event:      oofevents.NewGameAction("match-1", oofevents.ActionAssist, oofevents.TeamBlue, "pid-a", "Alice"),
			impactTeam: oofevents.TeamBlue,
		},
		{
			name:       "demo",
			event:      oofevents.NewGameAction("match-1", oofevents.ActionDemo, oofevents.TeamOrange, "pid-b", "Bob", oofevents.WithVictim("pid-a")),
			impactTeam: oofevents.TeamOrange,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			engine := NewEngine(Config{Decay: 1})

			state := engine.ApplyGameAction(tc.event)

			if state.Sequence != 1 {
				t.Fatalf("Sequence = %d, want 1", state.Sequence)
			}
			if state.LastEvent.ImpactTeam != tc.impactTeam {
				t.Fatalf("ImpactTeam = %q, want %q", state.LastEvent.ImpactTeam, tc.impactTeam)
			}
			signal := state.Teams[tc.impactTeam]
			if signal.MomentumInfluence <= 0 {
				t.Fatalf("impact team signal not updated: %+v", signal)
			}
			if signal.Pressure < 0 || signal.Pressure > 1 || signal.Confidence < 0 || signal.Confidence > 1 {
				t.Fatalf("signal out of bounds: %+v", signal)
			}
		})
	}
}

func TestEpicSaveAddsMoreSignalThanRegularSave(t *testing.T) {
	regular := NewEngine(Config{Decay: 1}).ApplyGameAction(oofevents.NewGameAction(
		"match-1", oofevents.ActionSave, oofevents.TeamBlue, "pid-a", "Alice",
	))
	epic := NewEngine(Config{Decay: 1}).ApplyGameAction(oofevents.NewGameAction(
		"match-1", oofevents.ActionSave, oofevents.TeamBlue, "pid-a", "Alice", oofevents.WithEpicSave(),
	))

	if epic.Teams[oofevents.TeamBlue].MomentumInfluence <= regular.Teams[oofevents.TeamBlue].MomentumInfluence {
		t.Fatalf("epic save influence %f should exceed regular save %f",
			epic.Teams[oofevents.TeamBlue].MomentumInfluence,
			regular.Teams[oofevents.TeamBlue].MomentumInfluence)
	}
	if !epic.LastEvent.IsEpicSave {
		t.Fatal("last event should preserve IsEpicSave")
	}
}

func TestUnsupportedActionAndTeamAreIgnored(t *testing.T) {
	engine := NewEngine(Config{Decay: 1})

	state := engine.ApplyGameAction(oofevents.NewGameAction("match-1", oofevents.ActionKind("boost_grab"), oofevents.TeamBlue, "pid-a", "Alice"))
	if state.Sequence != 0 {
		t.Fatalf("unsupported action changed sequence to %d", state.Sequence)
	}

	state = engine.ApplyGameAction(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.Team("green"), "pid-a", "Alice"))
	if state.Sequence != 0 {
		t.Fatalf("unsupported team changed sequence to %d", state.Sequence)
	}
}

func TestSnapshotReturnsCopy(t *testing.T) {
	engine := NewEngine(Config{Decay: 1})
	state := engine.ApplyGameAction(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice"))
	state.Teams[oofevents.TeamBlue] = TeamSignal{}

	snapshot := engine.Snapshot()
	if snapshot.Teams[oofevents.TeamBlue].MomentumInfluence == 0 {
		t.Fatal("mutating returned state should not mutate engine state")
	}
}

func TestResetClearsRuntimeState(t *testing.T) {
	engine := NewEngine(Config{Decay: 1})
	engine.ApplyGameAction(oofevents.NewGameAction("match-1", oofevents.ActionGoal, oofevents.TeamBlue, "pid-a", "Alice"))

	engine.Reset()
	state := engine.Snapshot()

	if state.MatchGUID != "" || state.Sequence != 0 || state.LastEvent.Action != "" {
		t.Fatalf("state not reset: %+v", state)
	}
	if len(state.Teams) != 2 {
		t.Fatalf("team signals len = %d, want 2", len(state.Teams))
	}
}

func TestNewMatchGUIDResetsRuntimeState(t *testing.T) {
	engine := NewEngine(Config{Decay: 1})
	engine.ApplyGameAction(oofevents.NewGameAction("match-1", oofevents.ActionGoal, oofevents.TeamBlue, "pid-a", "Alice"))

	state := engine.ApplyGameAction(oofevents.NewGameAction("match-2", oofevents.ActionBallHit, oofevents.TeamOrange, "pid-b", "Bob"))

	if state.MatchGUID != "match-2" {
		t.Fatalf("MatchGUID = %q, want match-2", state.MatchGUID)
	}
	if state.Sequence != 1 {
		t.Fatalf("Sequence after new match = %d, want 1", state.Sequence)
	}
	if state.Teams[oofevents.TeamBlue].MomentumInfluence != 0 {
		t.Fatalf("blue influence carried across match: %+v", state.Teams[oofevents.TeamBlue])
	}
}

func TestSignalsRemainBounded(t *testing.T) {
	engine := NewEngine(Config{Decay: 1})

	for i := 0; i < 20; i++ {
		engine.ApplyGameAction(oofevents.NewGameAction("match-1", oofevents.ActionGoal, oofevents.TeamBlue, "pid-a", "Alice"))
	}

	signal := engine.Snapshot().Teams[oofevents.TeamBlue]
	assertBounded(t, signal.Pressure)
	assertBounded(t, signal.MomentumInfluence)
	assertBounded(t, signal.ContestInvolvement)
	assertBounded(t, signal.EventDerivedControl)
	assertBounded(t, signal.Confidence)
	assertBounded(t, signal.Volatility)
}

func assertBounded(t *testing.T, value float64) {
	t.Helper()
	if value < 0 || value > 1 {
		t.Fatalf("value %f outside [0, 1]", value)
	}
}
