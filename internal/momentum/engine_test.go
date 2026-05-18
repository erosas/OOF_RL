package momentum

import (
	"testing"
	"time"

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

func TestSaveSplitsDefensiveControlAndAttackingPressure(t *testing.T) {
	state := NewEngine(Config{Decay: 1}).ApplyGameAction(oofevents.NewGameAction(
		"match-1", oofevents.ActionSave, oofevents.TeamBlue, "pid-a", "Alice",
	))

	blue := state.Teams[oofevents.TeamBlue]
	orange := state.Teams[oofevents.TeamOrange]
	if blue.EventDerivedControl <= 0 {
		t.Fatalf("defending team should receive control signal on save: blue=%+v", blue)
	}
	if orange.Pressure <= 0 {
		t.Fatalf("attacking team should retain forced-pressure signal on save: orange=%+v", orange)
	}
	if orange.Pressure <= blue.Pressure {
		t.Fatalf("save pressure should describe attacking pressure, got blue=%+v orange=%+v", blue, orange)
	}
}

func TestDemoBeforeShotAddsPressureBonus(t *testing.T) {
	base := NewEngine(Config{Decay: 1})
	baseShot := at(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice"), time.Unix(100, 0))
	withoutDemo := base.ApplyGameAction(baseShot).Teams[oofevents.TeamBlue]

	with := NewEngine(Config{Decay: 1})
	with.ApplyGameAction(at(oofevents.NewGameAction("match-1", oofevents.ActionDemo, oofevents.TeamBlue, "pid-a", "Alice", oofevents.WithVictim("pid-b")), time.Unix(99, 0)))
	withDemo := with.ApplyGameAction(baseShot).Teams[oofevents.TeamBlue]

	if withDemo.Pressure <= withoutDemo.Pressure {
		t.Fatalf("demo before shot should increase pressure: without=%+v with=%+v", withoutDemo, withDemo)
	}
}

func TestDemoBeforeGoalAddsPressureBonus(t *testing.T) {
	base := NewEngine(Config{Decay: 1})
	goal := at(oofevents.NewGameAction("match-1", oofevents.ActionGoal, oofevents.TeamOrange, "pid-b", "Bob"), time.Unix(100, 0))
	withoutDemo := base.ApplyGameAction(goal).Teams[oofevents.TeamOrange]

	with := NewEngine(Config{Decay: 1})
	with.ApplyGameAction(at(oofevents.NewGameAction("match-1", oofevents.ActionDemo, oofevents.TeamOrange, "pid-b", "Bob", oofevents.WithVictim("pid-a")), time.Unix(96, 0)))
	withDemo := with.ApplyGameAction(goal).Teams[oofevents.TeamOrange]

	if withDemo.Pressure <= withoutDemo.Pressure {
		t.Fatalf("demo before goal should increase pressure: without=%+v with=%+v", withoutDemo, withDemo)
	}
}

func TestSameTeamBallHitChainAddsControl(t *testing.T) {
	engine := NewEngine(Config{Decay: 1})
	first := at(oofevents.NewGameAction("match-1", oofevents.ActionBallHit, oofevents.TeamBlue, "pid-a", "Alice"), time.Unix(100, 0))
	second := at(oofevents.NewGameAction("match-1", oofevents.ActionBallHit, oofevents.TeamBlue, "pid-a", "Alice"), time.Unix(102, 0))

	afterFirst := engine.ApplyGameAction(first).Teams[oofevents.TeamBlue]
	afterSecond := engine.ApplyGameAction(second).Teams[oofevents.TeamBlue]

	if afterSecond.EventDerivedControl <= afterFirst.EventDerivedControl*2 {
		t.Fatalf("same-team touch chain should add bonus control: first=%+v second=%+v", afterFirst, afterSecond)
	}
}

func TestAlternatingBallHitsAddVolatilityAndReducePreviousControl(t *testing.T) {
	engine := NewEngine(Config{Decay: 1})
	engine.ApplyGameAction(at(oofevents.NewGameAction("match-1", oofevents.ActionBallHit, oofevents.TeamBlue, "pid-a", "Alice"), time.Unix(100, 0)))
	state := engine.ApplyGameAction(at(oofevents.NewGameAction("match-1", oofevents.ActionBallHit, oofevents.TeamOrange, "pid-b", "Bob"), time.Unix(101, 0)))

	blue := state.Teams[oofevents.TeamBlue]
	orange := state.Teams[oofevents.TeamOrange]
	if orange.Volatility <= DefaultConfig().BallHitPressure {
		t.Fatalf("alternating touches should add volatility: orange=%+v", orange)
	}
	if blue.EventDerivedControl >= DefaultConfig().BallHitControl {
		t.Fatalf("opponent touch should reduce previous team control: blue=%+v", blue)
	}
}

func TestConfigExposesEventWeights(t *testing.T) {
	cfg := Config{
		Decay:        1,
		ShotControl:  0.01,
		ShotPressure: 0.50,
	}
	state := NewEngine(cfg).ApplyGameAction(oofevents.NewGameAction(
		"match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice",
	))

	if got := state.Teams[oofevents.TeamBlue].Pressure; got != 0.50 {
		t.Fatalf("shot pressure = %f, want configured 0.50", got)
	}
	if got := state.Teams[oofevents.TeamBlue].EventDerivedControl; got != 0.01 {
		t.Fatalf("shot control = %f, want configured 0.01", got)
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

func at(event oofevents.GameActionEvent, t time.Time) oofevents.GameActionEvent {
	event.Base.At = t
	return event
}
