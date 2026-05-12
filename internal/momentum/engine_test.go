package momentum

import "testing"

func testEngine() *EventPressureEngine {
	return NewEventPressureEngine(DefaultConfig())
}

func ev(kind EventType, team Team, at int64) NormalizedGameEvent {
	return NormalizedGameEvent{
		Type:       kind,
		Team:       team,
		PlayerID:   string(team) + "-player",
		PlayerName: string(team) + " Player",
		Time:       at,
	}
}

func TestSameTeamTouchChainCreatesControl(t *testing.T) {
	engine := testEngine()
	engine.ProcessEvent(ev(EventBallHit, TeamBlue, 1000))
	engine.ProcessEvent(ev(EventBallHit, TeamBlue, 1800))
	out := engine.ProcessEvent(ev(EventBallHit, TeamBlue, 2500))

	if out.State != StateBlueControl {
		t.Fatalf("state: got %s, want %s; output=%+v", out.State, StateBlueControl, out)
	}
	if out.Blue.Control <= out.Orange.Control {
		t.Fatalf("blue control should dominate: %+v", out)
	}
	if out.Confidence <= 0.35 {
		t.Fatalf("confidence should rise above threshold, got %.3f", out.Confidence)
	}
}

func TestAlternatingTouchesCreateVolatileOrNeutral(t *testing.T) {
	engine := testEngine()
	engine.ProcessEvent(ev(EventBallHit, TeamBlue, 1000))
	engine.ProcessEvent(ev(EventBallHit, TeamOrange, 1500))
	engine.ProcessEvent(ev(EventBallHit, TeamBlue, 1900))
	out := engine.ProcessEvent(ev(EventBallHit, TeamOrange, 2300))

	if out.State == StateBlueControl || out.State == StateOrangeControl {
		t.Fatalf("alternating touches should not classify as strong control: %+v", out)
	}
	if out.State != StateVolatile && out.State != StateNeutral {
		t.Fatalf("state: got %s, want volatile or neutral", out.State)
	}
}

func TestShotAndSaveKeepAttackingPressure(t *testing.T) {
	engine := testEngine()
	engine.ProcessEvent(ev(EventShot, TeamBlue, 1000))
	out := engine.ProcessEvent(ev(EventSave, TeamOrange, 1800))

	if out.Blue.Pressure <= out.Orange.Pressure {
		t.Fatalf("blue pressure should remain meaningful after forced save: %+v", out)
	}
	if out.Orange.Control <= 0 {
		t.Fatalf("orange should receive defensive control after save: %+v", out)
	}
	if out.State == StateOrangePressure {
		t.Fatalf("save should not instantly flip to orange pressure: %+v", out)
	}
}

func TestDemoIntoShotAppliesPressureBonus(t *testing.T) {
	withDemo := testEngine()
	withDemo.ProcessEvent(ev(EventDemo, TeamBlue, 1000))
	demoOut := withDemo.ProcessEvent(ev(EventShot, TeamBlue, 4000))

	withoutDemo := testEngine()
	noDemoOut := withoutDemo.ProcessEvent(ev(EventShot, TeamBlue, 4000))

	if demoOut.Blue.Pressure <= noDemoOut.Blue.Pressure {
		t.Fatalf("demo into shot should add pressure: with=%+v without=%+v", demoOut, noDemoOut)
	}
	if demoOut.Overlay.Pulse != PulseDemoPressure {
		t.Fatalf("pulse: got %s, want %s", demoOut.Overlay.Pulse, PulseDemoPressure)
	}
}

func TestDemoIntoGoalAppliesGoalBurstAndBonus(t *testing.T) {
	engine := testEngine()
	engine.ProcessEvent(ev(EventDemo, TeamBlue, 1000))
	out := engine.ProcessEvent(ev(EventGoal, TeamBlue, 7000))

	if out.State != StateBluePressure {
		t.Fatalf("state: got %s, want %s; output=%+v", out.State, StateBluePressure, out)
	}
	if out.Overlay.Pulse != PulseGoalBurst {
		t.Fatalf("pulse: got %s, want %s", out.Overlay.Pulse, PulseGoalBurst)
	}
	if out.Confidence < 0.35 {
		t.Fatalf("goal should produce high confidence: %+v", out)
	}
}

func TestLongStaleGapTrendsNeutral(t *testing.T) {
	engine := testEngine()
	engine.ProcessEvent(ev(EventBallHit, TeamBlue, 1000))
	engine.ProcessEvent(ev(EventBallHit, TeamBlue, 1600))
	engine.ProcessEvent(ev(EventBallHit, TeamBlue, 2200))
	out := engine.Tick(32000)

	if out.State != StateNeutral {
		t.Fatalf("stale state should become neutral: %+v", out)
	}
	if out.Confidence >= 0.35 {
		t.Fatalf("stale confidence should decay below threshold: %+v", out)
	}
}

func TestGoalCreatesPressureBurst(t *testing.T) {
	engine := testEngine()
	out := engine.ProcessEvent(ev(EventGoal, TeamOrange, 1000))

	if out.State != StateOrangePressure {
		t.Fatalf("state: got %s, want %s; output=%+v", out.State, StateOrangePressure, out)
	}
	if out.Overlay.Pulse != PulseGoalBurst {
		t.Fatalf("pulse: got %s, want %s", out.Overlay.Pulse, PulseGoalBurst)
	}
	if out.Confidence < 0.35 {
		t.Fatalf("goal should produce high confidence: %+v", out)
	}
}

func TestPulseIsHeldBrieflyForPolling(t *testing.T) {
	engine := testEngine()
	out := engine.ProcessEvent(ev(EventShot, TeamBlue, 1000))
	if out.Overlay.Pulse != PulseShot {
		t.Fatalf("initial pulse: got %s, want %s", out.Overlay.Pulse, PulseShot)
	}

	held := engine.Tick(1800)
	if held.Overlay.Pulse != PulseShot {
		t.Fatalf("pulse should remain available for UI polling: %+v", held.Overlay)
	}

	expired := engine.Tick(2600)
	if expired.Overlay.Pulse != "" {
		t.Fatalf("pulse should expire after hold window: %+v", expired.Overlay)
	}
}

func TestDebugOutputIncludesSignalCounts(t *testing.T) {
	engine := testEngine()
	engine.ProcessEvent(NormalizedGameEvent{
		Type:        EventShot,
		Team:        TeamBlue,
		PlayerID:    "blue-1",
		PlayerName:  "Blue One",
		Time:        1000,
		SourceEvent: "UpdateStateDelta",
	})
	out := engine.Output()

	if out.Debug == nil {
		t.Fatal("debug output missing")
	}
	if out.Debug.EventCounts[EventShot] != 1 {
		t.Fatalf("shot count: got %+v", out.Debug.EventCounts)
	}
	if out.Debug.SourceCounts["UpdateStateDelta"] != 1 {
		t.Fatalf("source count: got %+v", out.Debug.SourceCounts)
	}
	if out.Debug.LastStrongEvent == nil || out.Debug.LastStrongEvent.Type != EventShot {
		t.Fatalf("last strong event: got %+v", out.Debug.LastStrongEvent)
	}
}
