package overlayhud

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"OOF_RL/internal/events"
	"OOF_RL/internal/momentum"
	"OOF_RL/internal/oofevents"
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

func TestOOFEventBusFeedsMomentumWithBallHitTeamCache(t *testing.T) {
	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatal(err)
	}
	defer bus.Stop()

	p := New()
	if err := p.Init(bus.ForPlugin(p.ID()), nil, nil); err != nil {
		t.Fatal(err)
	}
	defer p.Shutdown()

	guid := "guid-oof"
	bus.PublishAuthoritative(oofevents.NewStateUpdated(guid, []oofevents.PlayerSnapshot{{
		Name:      "Blue One",
		PrimaryID: "pid-blue",
		Shortcut:  7,
		TeamNum:   0,
	}}, oofevents.GameSnapshot{TimeSeconds: 245}))

	waitFor(t, func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.playerCacheGUID == guid
	})

	bus.PublishAuthoritative(oofevents.NewBallHit(guid, "Blue One", "pid-blue", 7, 900, 1100, 0, 0, 0))

	waitFor(t, func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()
		out := p.engine.Output()
		return out.Debug != nil && out.Debug.EventCounts[momentum.EventBallHit] == 1
	})
}

func TestOOFStatFeedMapsToMomentumPressureEvent(t *testing.T) {
	p := New()
	guid := "guid-statfeed"
	p.HandleOOFEvent(oofevents.NewStateUpdated(guid, []oofevents.PlayerSnapshot{{
		Name:      "Orange One",
		PrimaryID: "pid-orange",
		Shortcut:  8,
		TeamNum:   1,
	}}, oofevents.GameSnapshot{TimeSeconds: 244}))

	p.HandleOOFEvent(oofevents.NewStatFeed(guid, "Shot", "Orange One", 8, 1, "", 0))

	p.mu.Lock()
	out := p.engine.Output()
	p.mu.Unlock()
	if out.Debug == nil || out.Debug.EventCounts[momentum.EventShot] != 1 {
		t.Fatalf("oof stat.feed shot should become one momentum shot, got %+v", out.Debug)
	}
}

func TestOverlayPerfCountsOOFAndNormalizedEventsWhenEnabled(t *testing.T) {
	p := New()
	p.handlePerf(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/overlay/perf?enable=1", nil))

	guid := "guid-perf"
	p.HandleOOFEvent(oofevents.NewStateUpdated(guid, []oofevents.PlayerSnapshot{{
		Name:      "Blue One",
		PrimaryID: "pid-blue",
		Shortcut:  7,
		TeamNum:   0,
	}}, oofevents.GameSnapshot{TimeSeconds: 244}))
	p.HandleOOFEvent(oofevents.NewStatFeed(guid, "Shot", "Blue One", 7, 0, "", 0))

	rr := httptest.NewRecorder()
	p.handlePerf(rr, httptest.NewRequest(http.MethodGet, "/api/overlay/perf", nil))

	var snapshot overlayPerfSnapshot
	if err := json.Unmarshal(rr.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode perf snapshot: %v", err)
	}
	if !snapshot.Enabled {
		t.Fatal("expected perf counters to stay enabled")
	}
	if snapshot.Totals["oofevents.state.updated"] != 1 {
		t.Fatalf("state.updated count = %d, want 1", snapshot.Totals["oofevents.state.updated"])
	}
	if snapshot.Totals["oofevents.stat.feed"] != 1 {
		t.Fatalf("stat.feed count = %d, want 1", snapshot.Totals["oofevents.stat.feed"])
	}
	if snapshot.Totals["normalized.shot"] != 1 {
		t.Fatalf("normalized shot count = %d, want 1", snapshot.Totals["normalized.shot"])
	}
}

func TestOverlayPerfFrontendReportIncludesVisibilityFields(t *testing.T) {
	p := New()
	p.handlePerf(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/overlay/perf?enable=1", nil))

	body := []byte(`{
		"clientId": "hud-test",
		"perfSchemaVersion": 2,
		"isHud": true,
		"previewPaused": false,
		"documentHidden": false,
		"visibilityState": "visible",
		"windowFocused": true,
		"viewActive": true,
		"renderActive": true,
		"hudVisibleGuess": true,
		"f9WindowVisible": true,
		"f9WindowVisibilityKnown": true,
		"perfRole": "f9-hud-window",
		"perfStatus": "render-active",
		"visibilitySource": "f9-window-binding",
		"visual": "wheel",
		"variant": "full",
		"barHidden": true,
		"wheelHidden": false,
		"barDisplay": "none",
		"wheelDisplay": "grid",
		"barNodes": 12,
		"wheelNodes": 777,
		"currentSecond": {"wheel.update": 1},
		"totals": {"wheel.update": 1},
		"url": "http://localhost:8080/?overlay=1&view=overlay&hud=1"
	}`)
	rr := httptest.NewRecorder()
	p.handlePerfFrontend(rr, httptest.NewRequest(http.MethodPost, "/api/overlay/perf/frontend", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("frontend perf report status = %d, want 200: %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	p.handlePerf(rr, httptest.NewRequest(http.MethodGet, "/api/overlay/perf", nil))

	var snapshot overlayPerfSnapshot
	if err := json.Unmarshal(rr.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode perf snapshot: %v", err)
	}
	report, ok := snapshot.Frontend["hud-test"]
	if !ok {
		t.Fatalf("expected frontend report for hud-test, got %+v", snapshot.Frontend)
	}
	if !report.IsHUD || report.DocumentHidden || report.VisibilityState != "visible" {
		t.Fatalf("unexpected visibility fields: %+v", report)
	}
	if report.SchemaVersion != 2 {
		t.Fatalf("perf schema version = %d, want 2", report.SchemaVersion)
	}
	if !report.WindowFocused || !report.ViewActive || !report.RenderActive || !report.HUDVisibleGuess {
		t.Fatalf("expected active HUD visibility fields, got %+v", report)
	}
	if !report.BarHidden || report.WheelHidden || report.BarDisplay != "none" || report.WheelDisplay != "grid" {
		t.Fatalf("unexpected frontend visual DOM fields: %+v", report)
	}
	if !report.F9WindowVisible || !report.F9VisibilityKnown {
		t.Fatalf("expected F9 window visibility fields, got %+v", report)
	}
	if report.PerfRole != "f9-hud-window" || report.PerfStatus != "render-active" || report.VisibilitySource != "f9-window-binding" {
		t.Fatalf("unexpected frontend perf status fields: %+v", report)
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

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !condition() {
		t.Fatal("condition was not met before timeout")
	}
}
