package overlayhud

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
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

func TestTypedGoalScoredMatchesLegacyGoalBehavior(t *testing.T) {
	now := time.UnixMilli(12345)
	legacy := New()
	typed := New()

	handleLegacyEnvelopeAt(t, legacy, envelope(t, "GoalScored", events.GoalScoredData{
		MatchGuid: "guid-goal",
		GoalTime:  42,
		Scorer: events.PlayerRef{
			Name:     "Blue One",
			Shortcut: 7,
			TeamNum:  0,
		},
	}), now)
	handleTypedOOFAt(t, typed, oofevents.NewGoalScored("guid-goal", "Blue One", 7, "", 0, 0, 99, 42, 0, 0, 0, 0), now)

	assertMomentumDebugParity(t, legacy, typed)
}

func TestTypedGoalScoredMatchesLegacyAssistBehavior(t *testing.T) {
	now := time.UnixMilli(12345)
	legacy := New()
	typed := New()

	handleLegacyEnvelopeAt(t, legacy, envelope(t, "GoalScored", events.GoalScoredData{
		MatchGuid: "guid-goal",
		GoalTime:  42,
		Scorer: events.PlayerRef{
			Name:     "Blue One",
			Shortcut: 7,
			TeamNum:  0,
		},
		Assister: &events.PlayerRef{
			Name:     "Blue Two",
			Shortcut: 8,
			TeamNum:  0,
		},
	}), now)
	handleTypedOOFAt(t, typed, oofevents.NewGoalScored("guid-goal", "Blue One", 7, "Blue Two", 8, 0, 99, 42, 0, 0, 0, 0), now)

	assertMomentumDebugParity(t, legacy, typed)
	if got := eventCount(t, typed, momentum.EventAssist); got != 1 {
		t.Fatalf("typed assisted goal should emit one assist, got %d", got)
	}
}

func TestTypedStatFeedMappingsMatchLegacy(t *testing.T) {
	cases := []struct {
		name      string
		eventName string
		want      momentum.EventType
	}{
		{"shot", "Shot", momentum.EventShot},
		{"save", "Save", momentum.EventSave},
		{"epic-save", "EpicSave", momentum.EventSave},
		{"assist", "Assist", momentum.EventAssist},
		{"demo", "Demolish", momentum.EventDemo},
		{"goal", "Goal", momentum.EventGoal},
		{"own-goal", "OwnGoal", momentum.EventGoal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			now := time.UnixMilli(12345)
			legacy := New()
			typed := New()

			handleLegacyEnvelopeAt(t, legacy, envelope(t, "StatfeedEvent", events.StatfeedEventData{
				MatchGuid: "guid-statfeed",
				EventName: tc.eventName,
				MainTarget: events.PlayerRef{
					Name:     "Orange One",
					Shortcut: 8,
					TeamNum:  1,
				},
				SecondaryTarget: &events.PlayerRef{
					Name:     "Blue Victim",
					Shortcut: 7,
					TeamNum:  0,
				},
			}), now)
			handleTypedOOFAt(t, typed, oofevents.NewStatFeed("guid-statfeed", tc.eventName, "Orange One", 8, 1, "Blue Victim", 7), now)

			assertMomentumDebugParity(t, legacy, typed)
			if got := eventCount(t, typed, tc.want); got != 1 {
				t.Fatalf("typed statfeed %s should emit %s once, got %d", tc.eventName, tc.want, got)
			}
		})
	}
}

func TestTypedBallHitUsesTeamCacheAndDropsCacheMiss(t *testing.T) {
	guid := "guid-ballhit"
	now := time.UnixMilli(12345)
	p := New()
	p.mu.Lock()
	p.perfResetLocked(now.UnixMilli(), true)
	p.mu.Unlock()

	handleTypedOOFAt(t, p, oofevents.NewBallHit(guid, "Blue One", "pid-blue", 7, 900, 1100, 0, 0, 0), now)
	if got := eventCount(t, p, momentum.EventBallHit); got != 0 {
		t.Fatalf("typed ball hit before cache should drop, got %d ball hits", got)
	}

	handleTypedOOFAt(t, p, oofevents.NewStateUpdated(guid, []oofevents.PlayerSnapshot{{
		Name:      "Blue One",
		PrimaryID: "pid-blue",
		Shortcut:  7,
		TeamNum:   0,
	}}, oofevents.GameSnapshot{TimeSeconds: 245}), now)
	handleTypedOOFAt(t, p, oofevents.NewBallHit(guid, "Blue One", "pid-blue", 7, 900, 1100, 0, 0, 0), now.Add(100*time.Millisecond))

	if got := eventCount(t, p, momentum.EventBallHit); got != 1 {
		t.Fatalf("typed ball hit after cache should emit once, got %d", got)
	}
	p.mu.Lock()
	cacheMisses := p.perf.Totals["cache.miss.ballHit"]
	p.mu.Unlock()
	if cacheMisses != 1 {
		t.Fatalf("cache miss count = %d, want 1", cacheMisses)
	}
}

func TestTypedGoalReplayEndSuppressionMatchesLegacy(t *testing.T) {
	now := time.UnixMilli(12345)
	legacy := New()
	typed := New()

	handleLegacyEnvelopeAt(t, legacy, envelope(t, "GoalScored", events.GoalScoredData{
		MatchGuid: "guid-replay-end",
		Scorer: events.PlayerRef{
			TeamNum: 0,
		},
		BallLastTouch: events.LastTouch{
			Player: events.PlayerRef{
				Name:    "Real scorer",
				TeamNum: 1,
			},
		},
	}), now)
	handleTypedOOFAt(t, typed, oofevents.NewGoalScored("guid-replay-end", "", 0, "", 0, 7, 99, 42, 0, 0, 0, 0), now)

	assertMomentumDebugParity(t, legacy, typed)
	if got := eventCount(t, typed, momentum.EventGoal); got != 0 {
		t.Fatalf("typed replay-end goal packet should be suppressed, got %d goals", got)
	}
}

func TestTypedDuplicateGoalStatFeedPairCountsOnce(t *testing.T) {
	p := New()
	now := time.UnixMilli(12345)

	handleTypedOOFAt(t, p, oofevents.NewStatFeed("guid-goal", "Goal", "Blue One", 7, 0, "", 0), now)
	handleTypedOOFAt(t, p, oofevents.NewGoalScored("guid-goal", "Blue One", 7, "", 0, 0, 99, 42, 0, 0, 0, 0), now.Add(100*time.Millisecond))

	if got := eventCount(t, p, momentum.EventGoal); got != 1 {
		t.Fatalf("typed duplicate statfeed/goal scored pair should count once, got %d", got)
	}
}

func TestTypedLifecycleResetMatchesLegacy(t *testing.T) {
	now := time.UnixMilli(12345)
	legacy := New()
	typed := New()

	handleLegacyEnvelopeAt(t, legacy, envelope(t, "GoalScored", events.GoalScoredData{
		MatchGuid: "guid-goal",
		Scorer: events.PlayerRef{
			Name:    "Blue One",
			TeamNum: 0,
		},
	}), now)
	handleTypedOOFAt(t, typed, oofevents.NewGoalScored("guid-goal", "Blue One", 7, "", 0, 0, 99, 42, 0, 0, 0, 0), now)
	handleLegacyEnvelopeAt(t, legacy, envelope(t, "MatchEnded", events.MatchEndedData{MatchGuid: "guid-goal", WinnerTeamNum: 0}), now.Add(time.Second))
	handleTypedOOFAt(t, typed, oofevents.NewMatchEnded("guid-goal", 0), now.Add(time.Second))

	assertMomentumDebugParity(t, legacy, typed)
	if state := outputState(t, typed); state != momentum.StateNeutral {
		t.Fatalf("typed match ended should reset momentum to neutral, got %s", state)
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
		"visibilitySource": "native-hud-query",
		"nativeHud": true,
		"assetVersion": "20260514041035.009302500",
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
		"url": "http://localhost:8080/?overlay=1&view=overlay&hud=1&nativeHud=1&assetVersion=20260514041035.009302500"
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
	if report.PerfRole != "f9-hud-window" || report.PerfStatus != "render-active" || report.VisibilitySource != "native-hud-query" {
		t.Fatalf("unexpected frontend perf status fields: %+v", report)
	}
	if !report.NativeHUD || report.AssetVersion != "20260514041035.009302500" || report.ClientClass != "native-f9-hud" {
		t.Fatalf("expected native F9 HUD classification fields, got %+v", report)
	}
}

func TestNativeHUDVisibilityCanStartHidden(t *testing.T) {
	p := New()

	rr := httptest.NewRecorder()
	p.handleNativeHUDVisibility(rr, httptest.NewRequest(http.MethodPost, "/api/overlay/hud/native-visibility?visible=0", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("POST hidden status = %d, want 200: %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	p.handleNativeHUDVisibility(rr, httptest.NewRequest(http.MethodGet, "/api/overlay/hud/native-visibility", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET hidden status = %d, want 200: %s", rr.Code, rr.Body.String())
	}

	var state nativeHUDState
	if err := json.Unmarshal(rr.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode native HUD state: %v", err)
	}
	if !state.Known || state.Visible {
		t.Fatalf("native HUD state = %+v, want known hidden", state)
	}
}

func TestOverlayPerfFrontendUnregisterRemovesClient(t *testing.T) {
	p := New()
	p.handlePerf(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/overlay/perf?enable=1", nil))

	body := []byte(`{
		"clientId": "hud-test",
		"perfSchemaVersion": 2,
		"isHud": true,
		"nativeHud": true,
		"assetVersion": "20260514041035.009302500",
		"currentSecond": {"wheel.update": 1},
		"totals": {"wheel.update": 1}
	}`)
	rr := httptest.NewRecorder()
	p.handlePerfFrontend(rr, httptest.NewRequest(http.MethodPost, "/api/overlay/perf/frontend", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("frontend perf report status = %d, want 200: %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	p.handlePerfFrontend(rr, httptest.NewRequest(http.MethodPost, "/api/overlay/perf/frontend", bytes.NewReader([]byte(`{
		"clientId": "hud-test",
		"unregister": true
	}`))))
	if rr.Code != http.StatusOK {
		t.Fatalf("frontend perf unregister status = %d, want 200: %s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	p.handlePerf(rr, httptest.NewRequest(http.MethodGet, "/api/overlay/perf", nil))

	var snapshot overlayPerfSnapshot
	if err := json.Unmarshal(rr.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode perf snapshot: %v", err)
	}
	if _, ok := snapshot.Frontend["hud-test"]; ok {
		t.Fatalf("expected hud-test to be unregistered, got %+v", snapshot.Frontend)
	}
}

func TestOverlayPerfPrunesExpiredFrontendReports(t *testing.T) {
	p := New()
	now := time.Now().UnixMilli()

	p.mu.Lock()
	p.perfResetLocked(now, true)
	p.perf.Frontend["old"] = overlayPerfFrontendReport{
		ClientID:      "old",
		SchemaVersion: 0,
		IsHUD:         true,
		At:            now - overlayPerfFrontendReportTTLMS - 1,
	}
	p.perf.Frontend["fresh"] = overlayPerfFrontendReport{
		ClientID:      "fresh",
		SchemaVersion: 2,
		IsHUD:         true,
		NativeHUD:     true,
		AssetVersion:  "20260514041035.009302500",
		At:            now,
	}
	snapshot := p.perfSnapshotLocked(now)
	_, oldStored := p.perf.Frontend["old"]
	p.mu.Unlock()

	if oldStored {
		t.Fatal("expected expired frontend report to be pruned from backend storage")
	}
	if _, ok := snapshot.Frontend["old"]; ok {
		t.Fatalf("expected expired frontend report to be absent from snapshot, got %+v", snapshot.Frontend)
	}
	if fresh, ok := snapshot.Frontend["fresh"]; !ok || fresh.ClientClass != "native-f9-hud" {
		t.Fatalf("expected fresh native F9 report to remain classified, got %+v", snapshot.Frontend)
	}
}

func TestOverlayPerfClassifiesLegacyAndManualHudClients(t *testing.T) {
	legacy := sanitizeFrontendPerfReport(overlayPerfFrontendReport{
		ClientID: "legacy",
		IsHUD:    true,
	})
	if legacy.ClientClass != "legacy-hud-client" {
		t.Fatalf("legacy client class = %q, want legacy-hud-client", legacy.ClientClass)
	}

	manual := sanitizeFrontendPerfReport(overlayPerfFrontendReport{
		ClientID:      "manual",
		SchemaVersion: 2,
		IsHUD:         true,
	})
	if manual.ClientClass != "manual-hud-url" {
		t.Fatalf("manual client class = %q, want manual-hud-url", manual.ClientClass)
	}

	lab := sanitizeFrontendPerfReport(overlayPerfFrontendReport{
		ClientID: "lab",
		IsHUD:    false,
	})
	if lab.ClientClass != "overlay-lab-preview" {
		t.Fatalf("lab client class = %q, want overlay-lab-preview", lab.ClientClass)
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

func handleLegacyEnvelopeAt(t *testing.T, p *Plugin, env events.Envelope, now time.Time) {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handleEnvelopeLocked(env, now)
}

func handleTypedOOFAt(t *testing.T, p *Plugin, ev oofevents.OOFEvent, now time.Time) {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handleOOFEventLocked(ev, now)
}

func eventCount(t *testing.T, p *Plugin, typ momentum.EventType) int {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	out := p.engine.Output()
	if out.Debug == nil || out.Debug.EventCounts == nil {
		return 0
	}
	return out.Debug.EventCounts[typ]
}

func outputState(t *testing.T, p *Plugin) momentum.FlowState {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.engine.Output().State
}

func assertMomentumDebugParity(t *testing.T, legacy, typed *Plugin) {
	t.Helper()
	legacy.mu.Lock()
	legacyOut := legacy.engine.Output()
	legacy.mu.Unlock()
	typed.mu.Lock()
	typedOut := typed.engine.Output()
	typed.mu.Unlock()
	if !reflect.DeepEqual(legacyOut, typedOut) {
		t.Fatalf("typed momentum output diverged from legacy path\nlegacy: %+v\ntyped:  %+v", legacyOut, typedOut)
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
