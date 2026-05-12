package overlayhud

import (
	"encoding/json"
	"testing"

	"OOF_RL/internal/events"
	"OOF_RL/internal/momentum"
)

func TestReplayStateClearsOnLiveBallHit(t *testing.T) {
	p := New()
	p.HandleEvent(envelope(t, "GoalScored", events.GoalScoredData{
		MatchGuid: "guid-replay",
		Scorer: events.PlayerRef{
			Name:      "Mr Mung Beans",
			PrimaryId: "pid",
			TeamNum:   0,
		},
	}))
	p.HandleEvent(envelope(t, "UpdateState", events.UpdateStateData{
		MatchGuid: "guid-replay",
		Game:      events.GameState{BReplay: true},
	}))

	p.mu.Lock()
	active := p.replayActive
	p.mu.Unlock()
	if !active {
		t.Fatal("expected replay state to become active")
	}

	p.HandleEvent(envelope(t, "BallHit", events.BallHitData{
		MatchGuid: "guid-replay",
		Players: []events.PlayerRef{{
			Name:      "Mr Mung Beans",
			PrimaryId: "pid",
			TeamNum:   0,
		}},
	}))

	p.mu.Lock()
	active = p.replayActive
	p.mu.Unlock()
	if active {
		t.Fatal("live ball hit should clear stale replay state")
	}
}

func TestReplayFileBallHitsDoNotClearMomentum(t *testing.T) {
	p := New()
	p.HandleEvent(envelope(t, "UpdateState", events.UpdateStateData{
		MatchGuid: "guid-replay-file",
		Game:      events.GameState{BReplay: true},
	}))
	p.HandleEvent(envelope(t, "BallHit", events.BallHitData{
		MatchGuid: "guid-replay-file",
		Players: []events.PlayerRef{{
			Name:      "Mr Mung Beans",
			PrimaryId: "pid",
			TeamNum:   0,
		}},
	}))
	p.HandleEvent(envelope(t, "BallHit", events.BallHitData{
		MatchGuid: "guid-replay-file",
		Players: []events.PlayerRef{{
			Name:      "Mr Mung Beans",
			PrimaryId: "pid",
			TeamNum:   0,
		}},
	}))

	p.mu.Lock()
	active := p.replayActive
	fileMode := p.replayFileMode
	resetAt := p.momentumResetAt
	out := p.engine.Output()
	p.mu.Unlock()

	if !active || !fileMode {
		t.Fatalf("expected replay-file mode to stay active, active=%v fileMode=%v", active, fileMode)
	}
	if resetAt != 0 {
		t.Fatalf("replay-file ball hits should not reset momentum, resetAt=%d", resetAt)
	}
	if out.Debug == nil || out.Debug.EventCounts[momentum.EventBallHit] != 2 {
		t.Fatalf("replay-file ball hits should accumulate, got %+v", out.Debug)
	}
}

func TestReplayStateClearsOnTerminalEvents(t *testing.T) {
	p := New()
	p.HandleEvent(envelope(t, "UpdateState", events.UpdateStateData{
		MatchGuid: "guid-replay",
		Game:      events.GameState{BReplay: true},
	}))
	p.HandleEvent(envelope(t, "MatchDestroyed", events.MatchGuidData{MatchGuid: "guid-replay"}))

	p.mu.Lock()
	active := p.replayActive
	p.mu.Unlock()
	if active {
		t.Fatal("terminal match event should clear replay state")
	}
}

func TestGoalFallbackResetsMomentumWhenReplayNeverAppears(t *testing.T) {
	p := New()
	p.HandleEvent(envelope(t, "GoalScored", events.GoalScoredData{
		MatchGuid: "guid-goal",
		Scorer: events.PlayerRef{
			Name:      "Mr Mung Beans",
			PrimaryId: "pid",
			TeamNum:   0,
		},
	}))

	p.mu.Lock()
	due := p.goalResetDue
	if due == 0 {
		p.mu.Unlock()
		t.Fatal("goal should arm skipped-replay fallback")
	}
	before := p.engine.Output()
	p.applyGoalFallback(due - 1)
	earlyReset := p.momentumResetAt
	p.applyGoalFallback(due)
	after := p.engine.Output()
	resetAt := p.momentumResetAt
	p.mu.Unlock()

	if earlyReset != 0 {
		t.Fatalf("fallback reset fired too early at %d", earlyReset)
	}
	if before.State == momentum.StateNeutral {
		t.Fatalf("goal should create pressure before fallback reset: %+v", before)
	}
	if after.State != momentum.StateNeutral {
		t.Fatalf("fallback should reset to neutral, got %+v", after)
	}
	if after.Overlay.MomentumBarBluePercent != 50 || after.Overlay.MomentumBarOrangePercent != 50 {
		t.Fatalf("fallback should reset bar to 50/50, got %+v", after.Overlay)
	}
	if resetAt != due {
		t.Fatalf("fallback reset timestamp = %d, want %d", resetAt, due)
	}
}

func TestGoalFallbackDoesNotFireWhenReplayAppears(t *testing.T) {
	p := New()
	p.HandleEvent(envelope(t, "GoalScored", events.GoalScoredData{
		MatchGuid: "guid-goal",
		Scorer: events.PlayerRef{
			Name:      "Mr Mung Beans",
			PrimaryId: "pid",
			TeamNum:   0,
		},
	}))
	p.HandleEvent(envelope(t, "UpdateState", events.UpdateStateData{
		MatchGuid: "guid-goal",
		Game:      events.GameState{BReplay: true},
	}))

	p.mu.Lock()
	due := p.goalResetDue
	p.applyGoalFallback(due + 1)
	resetAt := p.momentumResetAt
	p.mu.Unlock()
	if resetAt != 0 {
		t.Fatalf("fallback should not reset while replay phase is active, resetAt=%d", resetAt)
	}
}

func TestDuplicateExplicitGoalPairCountsOnce(t *testing.T) {
	p := New()
	p.HandleEvent(envelope(t, "StatfeedEvent", events.StatfeedEventData{
		MatchGuid: "guid-goal",
		EventName: "Goal",
		MainTarget: events.PlayerRef{
			Name:      "Mr Mung Beans",
			PrimaryId: "pid",
			TeamNum:   0,
		},
	}))
	p.HandleEvent(envelope(t, "GoalScored", events.GoalScoredData{
		MatchGuid: "guid-goal",
		GoalTime:  42,
		Scorer: events.PlayerRef{
			Name:      "Mr Mung Beans",
			PrimaryId: "pid",
			TeamNum:   0,
		},
	}))

	p.mu.Lock()
	out := p.engine.Output()
	p.mu.Unlock()
	if out.Debug == nil || out.Debug.EventCounts[momentum.EventGoal] != 1 {
		t.Fatalf("duplicate statfeed/goal scored pair should count once, got %+v", out.Debug)
	}
}

func TestDuplicateExplicitGoalPairMatchesNameFallback(t *testing.T) {
	p := New()
	p.HandleEvent(envelope(t, "StatfeedEvent", events.StatfeedEventData{
		MatchGuid: "guid-goal",
		EventName: "Goal",
		MainTarget: events.PlayerRef{
			Name:    "Mr Mung Beans",
			TeamNum: 0,
		},
	}))
	p.HandleEvent(envelope(t, "GoalScored", events.GoalScoredData{
		MatchGuid: "guid-goal",
		GoalTime:  42,
		Scorer: events.PlayerRef{
			Name:      "Mr Mung Beans",
			PrimaryId: "pid",
			TeamNum:   0,
		},
	}))

	p.mu.Lock()
	out := p.engine.Output()
	p.mu.Unlock()
	if out.Debug == nil || out.Debug.EventCounts[momentum.EventGoal] != 1 {
		t.Fatalf("duplicate explicit goal pair should match by player name fallback, got %+v", out.Debug)
	}
}

func envelope(t *testing.T, event string, data any) events.Envelope {
	t.Helper()
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal %s: %v", event, err)
	}
	return events.Envelope{Event: event, Data: b}
}
