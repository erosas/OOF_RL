package rlevents_test

import (
	"encoding/json"
	"testing"
	"time"

	"OOF_RL/internal/events"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/rlevents"
)

// -- helpers --

func env(event string, data any) events.Envelope {
	b, _ := json.Marshal(data)
	return events.Envelope{Event: event, Data: b}
}

func newBusWithTranslator(t *testing.T) (oofevents.Bus, *rlevents.Translator) {
	t.Helper()
	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(bus.Stop)
	return bus, rlevents.New(bus.ForPlugin(""))
}

func subscribe(t *testing.T, bus oofevents.Bus, typ string) chan oofevents.OOFEvent {
	t.Helper()
	ch := make(chan oofevents.OOFEvent, 1)
	sub := bus.Subscribe(typ, func(e oofevents.OOFEvent) {
		select {
		case ch <- e:
		default:
		}
	})
	t.Cleanup(sub.Cancel)
	return ch
}

func mustRecv(t *testing.T, ch chan oofevents.OOFEvent) oofevents.OOFEvent {
	t.Helper()
	select {
	case e := <-ch:
		return e
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for event")
		return nil
	}
}

func mustNotRecv(t *testing.T, ch chan oofevents.OOFEvent) {
	t.Helper()
	select {
	case e := <-ch:
		t.Fatalf("unexpected event: %s", e.Type())
	case <-time.After(50 * time.Millisecond):
	}
}

// as unwraps a bus-stamped event and asserts to the concrete type T.
func as[T oofevents.OOFEvent](t *testing.T, e oofevents.OOFEvent) T {
	t.Helper()
	inner := oofevents.Unwrap(e)
	v, ok := inner.(T)
	if !ok {
		t.Fatalf("type assertion failed: got %T (inner %T)", e, inner)
	}
	return v
}

// -- MatchCreated / MatchInitialized --

func TestTranslateMatchCreated(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	ch := subscribe(t, bus, oofevents.TypeMatchStarted)
	tr.Translate(env("MatchCreated", events.MatchGuidData{MatchGuid: "g1"}))
	e := mustRecv(t, ch)
	if e.MatchGUID() != "g1" {
		t.Errorf("guid %q, want g1", e.MatchGUID())
	}
	if e.Certainty() != oofevents.Authoritative {
		t.Errorf("certainty %s, want authoritative", e.Certainty())
	}
}

func TestTranslateMatchInitialized(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	ch := subscribe(t, bus, oofevents.TypeMatchStarted)
	tr.Translate(env("MatchInitialized", events.MatchGuidData{MatchGuid: "g2"}))
	e := mustRecv(t, ch)
	if e.MatchGUID() != "g2" {
		t.Errorf("guid %q, want g2", e.MatchGUID())
	}
}

func TestTranslateMatchCreatedEmptyGUIDIgnored(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	ch := subscribe(t, bus, oofevents.TypeMatchStarted)
	tr.Translate(env("MatchCreated", events.MatchGuidData{MatchGuid: ""}))
	mustNotRecv(t, ch)
}

// -- MatchEnded --

func TestTranslateMatchEnded(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	ch := subscribe(t, bus, oofevents.TypeMatchEnded)
	tr.Translate(env("MatchEnded", events.MatchEndedData{MatchGuid: "g1", WinnerTeamNum: 1}))
	e := as[oofevents.MatchEndedEvent](t, mustRecv(t, ch))
	if e.MatchGUID() != "g1" {
		t.Errorf("guid %q, want g1", e.MatchGUID())
	}
	if e.WinnerTeamNum != 1 {
		t.Errorf("winner team %d, want 1", e.WinnerTeamNum)
	}
}

// -- MatchDestroyed --

func TestTranslateMatchDestroyed(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	ch := subscribe(t, bus, oofevents.TypeMatchDestroyed)
	tr.Translate(env("MatchDestroyed", nil))
	mustRecv(t, ch)
}

// -- GoalScored --

func TestTranslateGoalScoredNoAssister(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	ch := subscribe(t, bus, oofevents.TypeGoalScored)
	tr.Translate(env("GoalScored", events.GoalScoredData{
		MatchGuid:      "g1",
		Scorer:         events.PlayerRef{Name: "Alice", Shortcut: 5, TeamNum: 0},
		GoalSpeed:      120.5,
		GoalTime:       90.0,
		ImpactLocation: events.Vec3{X: 1, Y: 2, Z: 3},
		BallLastTouch:  events.LastTouch{Player: events.PlayerRef{Shortcut: 5}},
	}))
	e := as[oofevents.GoalScoredEvent](t, mustRecv(t, ch))
	if e.Scorer != "Alice" {
		t.Errorf("scorer %q, want Alice", e.Scorer)
	}
	if e.ScorerShortcut != 5 {
		t.Errorf("scorer shortcut %d, want 5", e.ScorerShortcut)
	}
	if e.Assister != "" {
		t.Errorf("assister %q, want empty", e.Assister)
	}
	if e.AssisterShortcut != 0 {
		t.Errorf("assister shortcut %d, want 0", e.AssisterShortcut)
	}
	if e.LastTouchShortcut != 5 {
		t.Errorf("last touch shortcut %d, want 5", e.LastTouchShortcut)
	}
	if e.GoalSpeed != 120.5 {
		t.Errorf("goal speed %f, want 120.5", e.GoalSpeed)
	}
	if e.ImpactX != 1 || e.ImpactY != 2 || e.ImpactZ != 3 {
		t.Errorf("impact (%f,%f,%f), want (1,2,3)", e.ImpactX, e.ImpactY, e.ImpactZ)
	}
}

func TestTranslateGoalScoredWithAssister(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	ch := subscribe(t, bus, oofevents.TypeGoalScored)
	assister := events.PlayerRef{Name: "Bob", Shortcut: 7}
	tr.Translate(env("GoalScored", events.GoalScoredData{
		MatchGuid: "g1",
		Scorer:    events.PlayerRef{Name: "Alice", Shortcut: 5},
		Assister:  &assister,
	}))
	e := as[oofevents.GoalScoredEvent](t, mustRecv(t, ch))
	if e.Assister != "Bob" {
		t.Errorf("assister %q, want Bob", e.Assister)
	}
	if e.AssisterShortcut != 7 {
		t.Errorf("assister shortcut %d, want 7", e.AssisterShortcut)
	}
}

// -- StatfeedEvent --

func TestTranslateStatFeed(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	ch := subscribe(t, bus, oofevents.TypeStatFeed)
	tr.Translate(env("StatfeedEvent", events.StatfeedEventData{
		MatchGuid:  "g1",
		EventName:  "Save",
		MainTarget: events.PlayerRef{Name: "Alice", PrimaryId: "pid-a", Shortcut: 3, TeamNum: 1},
	}))
	e := as[oofevents.StatFeedEvent](t, mustRecv(t, ch))
	if e.EventName != "Save" {
		t.Errorf("event name %q, want Save", e.EventName)
	}
	if e.MainTarget != "Alice" || e.MainTargetPrimaryID != "pid-a" {
		t.Errorf("main target name=%q id=%q", e.MainTarget, e.MainTargetPrimaryID)
	}
	if e.MainTargetShortcut != 3 {
		t.Errorf("main target shortcut %d, want 3", e.MainTargetShortcut)
	}
	if e.MainTargetTeamNum != 1 {
		t.Errorf("main target team %d, want 1", e.MainTargetTeamNum)
	}
	if e.SecondaryTarget != "" || e.SecondaryTargetPrimaryID != "" {
		t.Errorf("secondary target name=%q id=%q, want empty", e.SecondaryTarget, e.SecondaryTargetPrimaryID)
	}
}

func TestTranslateStatFeedWithSecondary(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	ch := subscribe(t, bus, oofevents.TypeStatFeed)
	victim := events.PlayerRef{Name: "Bob", PrimaryId: "pid-b", Shortcut: 8}
	tr.Translate(env("StatfeedEvent", events.StatfeedEventData{
		MatchGuid:       "g1",
		EventName:       "Demolish",
		MainTarget:      events.PlayerRef{Name: "Alice", PrimaryId: "pid-a", Shortcut: 3},
		SecondaryTarget: &victim,
	}))
	e := as[oofevents.StatFeedEvent](t, mustRecv(t, ch))
	if e.SecondaryTarget != "Bob" || e.SecondaryTargetPrimaryID != "pid-b" {
		t.Errorf("secondary target name=%q id=%q", e.SecondaryTarget, e.SecondaryTargetPrimaryID)
	}
	if e.SecondaryTargetShortcut != 8 {
		t.Errorf("secondary shortcut %d, want 8", e.SecondaryTargetShortcut)
	}
}

// -- ClockUpdatedSeconds --

func TestTranslateClockUpdated(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	ch := subscribe(t, bus, oofevents.TypeClockUpdated)
	tr.Translate(env("ClockUpdatedSeconds", events.ClockData{MatchGuid: "g1", TimeSeconds: 180}))
	e := as[oofevents.ClockUpdatedEvent](t, mustRecv(t, ch))
	if e.TimeSeconds != 180 {
		t.Errorf("time %d, want 180", e.TimeSeconds)
	}
	if e.IsOvertime {
		t.Error("expected not overtime")
	}
}

func TestTranslateClockOvertimeTransition(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	otCh := subscribe(t, bus, oofevents.TypeOvertimeStarted)
	clockCh := subscribe(t, bus, oofevents.TypeClockUpdated)

	tr.Translate(env("ClockUpdatedSeconds", events.ClockData{MatchGuid: "g1", TimeSeconds: 0, BOvertime: true}))

	ot := as[oofevents.OvertimeStartedEvent](t, mustRecv(t, otCh))
	if ot.ClockSeconds != 0 {
		t.Errorf("clock seconds %d, want 0", ot.ClockSeconds)
	}
	if ot.Certainty() != oofevents.Inferred {
		t.Errorf("certainty %s, want inferred", ot.Certainty())
	}
	clock := as[oofevents.ClockUpdatedEvent](t, mustRecv(t, clockCh))
	if !clock.IsOvertime {
		t.Error("expected overtime flag on clock event")
	}
}

func TestTranslateOvertimeNotRepublished(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	otCh := subscribe(t, bus, oofevents.TypeOvertimeStarted)

	// First overtime tick — should emit OvertimeStarted.
	tr.Translate(env("ClockUpdatedSeconds", events.ClockData{MatchGuid: "g1", BOvertime: true}))
	mustRecv(t, otCh)

	// Second overtime tick — should NOT emit OvertimeStarted again.
	tr.Translate(env("ClockUpdatedSeconds", events.ClockData{MatchGuid: "g1", BOvertime: true}))
	mustNotRecv(t, otCh)
}

// -- BallHit --

func TestTranslateBallHitWithPlayer(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	ch := subscribe(t, bus, oofevents.TypeBallHit)
	tr.Translate(env("BallHit", events.BallHitData{
		MatchGuid: "g1",
		Players:   []events.PlayerRef{{Name: "Alice", PrimaryId: "steam|123", Shortcut: 2, TeamNum: 1}},
		Ball:      events.BallHitBall{PreHitSpeed: 10, PostHitSpeed: 50, Location: events.Vec3{X: 1, Y: 2, Z: 3}},
	}))
	e := as[oofevents.BallHitEvent](t, mustRecv(t, ch))
	if e.PlayerName != "Alice" {
		t.Errorf("player name %q, want Alice", e.PlayerName)
	}
	if e.PlayerPrimaryID != "steam|123" {
		t.Errorf("primary id %q, want steam|123", e.PlayerPrimaryID)
	}
	if e.PlayerShortcut != 2 {
		t.Errorf("shortcut %d, want 2", e.PlayerShortcut)
	}
	if e.PlayerTeamNum != 1 {
		t.Errorf("team num %d, want 1", e.PlayerTeamNum)
	}
	if e.PreHitSpeed != 10 || e.PostHitSpeed != 50 {
		t.Errorf("speeds (%f,%f), want (10,50)", e.PreHitSpeed, e.PostHitSpeed)
	}
	if e.LocX != 1 || e.LocY != 2 || e.LocZ != 3 {
		t.Errorf("location (%f,%f,%f), want (1,2,3)", e.LocX, e.LocY, e.LocZ)
	}
}

func TestTranslateBallHitNoPlayers(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	ch := subscribe(t, bus, oofevents.TypeBallHit)
	tr.Translate(env("BallHit", events.BallHitData{MatchGuid: "g1", Players: nil}))
	e := as[oofevents.BallHitEvent](t, mustRecv(t, ch))
	if e.PlayerName != "" || e.PlayerPrimaryID != "" || e.PlayerShortcut != 0 || e.PlayerTeamNum != 0 {
		t.Errorf("expected empty player fields, got name=%q id=%q sc=%d team=%d", e.PlayerName, e.PlayerPrimaryID, e.PlayerShortcut, e.PlayerTeamNum)
	}
}

// -- CrossbarHit --

func TestTranslateCrossbarHit(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	ch := subscribe(t, bus, oofevents.TypeCrossbarHit)
	tr.Translate(env("CrossbarHit", events.CrossbarHitData{
		MatchGuid:     "g1",
		BallSpeed:     200.0,
		ImpactForce:   50.0,
		BallLastTouch: events.LastTouch{Player: events.PlayerRef{Name: "Alice"}},
	}))
	e := as[oofevents.CrossbarHitEvent](t, mustRecv(t, ch))
	if e.BallSpeed != 200.0 {
		t.Errorf("ball speed %f, want 200.0", e.BallSpeed)
	}
	if e.ImpactForce != 50.0 {
		t.Errorf("impact force %f, want 50.0", e.ImpactForce)
	}
	if e.LastToucher != "Alice" {
		t.Errorf("last toucher %q, want Alice", e.LastToucher)
	}
}

// -- UpdateState --

func TestTranslateUpdateState(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	ch := subscribe(t, bus, oofevents.TypeStateUpdated)
	tr.Translate(env("UpdateState", events.UpdateStateData{
		MatchGuid: "g1",
		Players: []events.Player{
			{Name: "Alice", PrimaryId: "steam|1", Shortcut: 1, TeamNum: 0, Score: 100, Goals: 2, CarTouches: 10},
		},
		Game: events.GameState{
			Teams:       []events.Team{{Name: "Blue", TeamNum: 0, Score: 2}},
			TimeSeconds: 240,
			Arena:       "DFH Stadium",
		},
	}))
	e := as[oofevents.StateUpdatedEvent](t, mustRecv(t, ch))
	if len(e.Players) != 1 {
		t.Fatalf("got %d players, want 1", len(e.Players))
	}
	p := e.Players[0]
	if p.Name != "Alice" || p.PrimaryID != "steam|1" || p.Shortcut != 1 || p.CarTouches != 10 {
		t.Errorf("player fields: %+v", p)
	}
	if e.Game.Arena != "DFH Stadium" || e.Game.TimeSeconds != 240 {
		t.Errorf("game fields: %+v", e.Game)
	}
	if len(e.Game.Teams) != 1 || e.Game.Teams[0].Score != 2 {
		t.Errorf("team fields: %+v", e.Game.Teams)
	}
}

func TestTranslateUpdateStateGUIDChangeEmitsRestart(t *testing.T) {
	bus, tr := newBusWithTranslator(t)

	// Establish initial GUID via MatchCreated so advanceGUID sets currentGUID.
	startCh := subscribe(t, bus, oofevents.TypeMatchStarted)
	tr.Translate(env("MatchCreated", events.MatchGuidData{MatchGuid: "g1"}))
	mustRecv(t, startCh)

	// UpdateState with a different GUID should emit MatchRestarted then StateUpdated.
	restartCh := subscribe(t, bus, oofevents.TypeMatchRestarted)
	stateCh := subscribe(t, bus, oofevents.TypeStateUpdated)
	tr.Translate(env("UpdateState", events.UpdateStateData{MatchGuid: "g2"}))

	restart := as[oofevents.MatchRestartedEvent](t, mustRecv(t, restartCh))
	if restart.MatchGUID() != "g2" {
		t.Errorf("restart new guid %q, want g2", restart.MatchGUID())
	}
	if restart.PreviousGUID != "g1" {
		t.Errorf("restart prev guid %q, want g1", restart.PreviousGUID)
	}
	mustRecv(t, stateCh)
}

// -- Error / unknown paths --

func TestTranslateUnknownEventIgnored(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	allCh := make(chan oofevents.OOFEvent, 1)
	sub := bus.SubscribeAll(func(e oofevents.OOFEvent) {
		select {
		case allCh <- e:
		default:
		}
	})
	t.Cleanup(sub.Cancel)
	tr.Translate(events.Envelope{Event: "SomeUnknownEvent"})
	mustNotRecv(t, allCh)
}

func TestTranslateInvalidJSONIgnored(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	ch := subscribe(t, bus, oofevents.TypeMatchStarted)
	tr.Translate(events.Envelope{Event: "MatchCreated", Data: []byte(`not json`)})
	mustNotRecv(t, ch)
}

// -- GameActionEvent --

func TestTranslateBallHitEmitsGameAction(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	gaCh := subscribe(t, bus, oofevents.TypeGameAction)
	tr.Translate(env("BallHit", events.BallHitData{
		MatchGuid: "g1",
		Players:   []events.PlayerRef{{Name: "Alice", PrimaryId: "pid-a", Shortcut: 2, TeamNum: 0}},
		Ball:      events.BallHitBall{PreHitSpeed: 10, PostHitSpeed: 50},
	}))
	e := as[oofevents.GameActionEvent](t, mustRecv(t, gaCh))
	if e.Action != oofevents.ActionBallHit {
		t.Errorf("action %q, want ball_hit", e.Action)
	}
	if e.Team != oofevents.TeamBlue {
		t.Errorf("team %q, want blue", e.Team)
	}
	if e.PlayerID != "pid-a" || e.PlayerName != "Alice" {
		t.Errorf("player id=%q name=%q", e.PlayerID, e.PlayerName)
	}
	if e.MatchGUID() != "g1" {
		t.Errorf("guid %q, want g1", e.MatchGUID())
	}
}

func TestTranslateBallHitNoPlayersNoGameAction(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	gaCh := subscribe(t, bus, oofevents.TypeGameAction)
	tr.Translate(env("BallHit", events.BallHitData{MatchGuid: "g1", Players: nil}))
	mustNotRecv(t, gaCh)
}

func TestTranslateGoalScoredNoGameAction(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	gaCh := subscribe(t, bus, oofevents.TypeGameAction)
	tr.Translate(env("GoalScored", events.GoalScoredData{
		MatchGuid: "g1",
		Scorer:    events.PlayerRef{Name: "Alice", PrimaryId: "pid-a", TeamNum: 0},
	}))
	mustNotRecv(t, gaCh)
}

func TestTranslateStatFeedShotEmitsGameAction(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	gaCh := subscribe(t, bus, oofevents.TypeGameAction)
	tr.Translate(env("StatfeedEvent", events.StatfeedEventData{
		MatchGuid:  "g1",
		EventName:  "Shot",
		MainTarget: events.PlayerRef{Name: "Alice", PrimaryId: "pid-a", TeamNum: 0},
	}))
	e := as[oofevents.GameActionEvent](t, mustRecv(t, gaCh))
	if e.Action != oofevents.ActionShot || e.Team != oofevents.TeamBlue {
		t.Errorf("action=%q team=%q", e.Action, e.Team)
	}
	if e.PlayerID != "pid-a" {
		t.Errorf("player id %q, want pid-a", e.PlayerID)
	}
}

func TestTranslateStatFeedSaveEmitsGameAction(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	gaCh := subscribe(t, bus, oofevents.TypeGameAction)
	tr.Translate(env("StatfeedEvent", events.StatfeedEventData{
		MatchGuid:  "g1",
		EventName:  "Save",
		MainTarget: events.PlayerRef{Name: "Alice", PrimaryId: "pid-a", TeamNum: 1},
	}))
	e := as[oofevents.GameActionEvent](t, mustRecv(t, gaCh))
	if e.Action != oofevents.ActionSave || e.Team != oofevents.TeamOrange {
		t.Errorf("action=%q team=%q", e.Action, e.Team)
	}
	if e.IsEpicSave {
		t.Error("IsEpicSave should be false for Save")
	}
}

func TestTranslateStatFeedEpicSaveEmitsGameAction(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	gaCh := subscribe(t, bus, oofevents.TypeGameAction)
	tr.Translate(env("StatfeedEvent", events.StatfeedEventData{
		MatchGuid:  "g1",
		EventName:  "EpicSave",
		MainTarget: events.PlayerRef{Name: "Alice", PrimaryId: "pid-a", TeamNum: 0},
	}))
	e := as[oofevents.GameActionEvent](t, mustRecv(t, gaCh))
	if e.Action != oofevents.ActionSave {
		t.Errorf("action %q, want save", e.Action)
	}
	if !e.IsEpicSave {
		t.Error("IsEpicSave should be true for EpicSave")
	}
}

func TestTranslateStatFeedGoalEmitsGameAction(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	gaCh := subscribe(t, bus, oofevents.TypeGameAction)
	tr.Translate(env("StatfeedEvent", events.StatfeedEventData{
		MatchGuid:  "g1",
		EventName:  "Goal",
		MainTarget: events.PlayerRef{Name: "Alice", PrimaryId: "pid-a", TeamNum: 0},
	}))
	e := as[oofevents.GameActionEvent](t, mustRecv(t, gaCh))
	if e.Action != oofevents.ActionGoal || e.IsOwnGoal {
		t.Errorf("action=%q ownGoal=%v", e.Action, e.IsOwnGoal)
	}
}

func TestTranslateStatFeedOwnGoalEmitsGameAction(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	gaCh := subscribe(t, bus, oofevents.TypeGameAction)
	tr.Translate(env("StatfeedEvent", events.StatfeedEventData{
		MatchGuid:  "g1",
		EventName:  "OwnGoal",
		MainTarget: events.PlayerRef{Name: "Alice", PrimaryId: "pid-a", TeamNum: 0},
	}))
	e := as[oofevents.GameActionEvent](t, mustRecv(t, gaCh))
	if e.Action != oofevents.ActionGoal || !e.IsOwnGoal {
		t.Errorf("action=%q ownGoal=%v", e.Action, e.IsOwnGoal)
	}
}

func TestTranslateStatFeedAssistEmitsGameAction(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	gaCh := subscribe(t, bus, oofevents.TypeGameAction)
	tr.Translate(env("StatfeedEvent", events.StatfeedEventData{
		MatchGuid:  "g1",
		EventName:  "Assist",
		MainTarget: events.PlayerRef{Name: "Alice", PrimaryId: "pid-a", TeamNum: 0},
	}))
	e := as[oofevents.GameActionEvent](t, mustRecv(t, gaCh))
	if e.Action != oofevents.ActionAssist {
		t.Errorf("action %q, want assist", e.Action)
	}
}

func TestTranslateStatFeedDemoEmitsGameAction(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	gaCh := subscribe(t, bus, oofevents.TypeGameAction)
	victim := events.PlayerRef{Name: "Bob", PrimaryId: "pid-b", Shortcut: 8}
	tr.Translate(env("StatfeedEvent", events.StatfeedEventData{
		MatchGuid:       "g1",
		EventName:       "Demolish",
		MainTarget:      events.PlayerRef{Name: "Alice", PrimaryId: "pid-a", TeamNum: 0},
		SecondaryTarget: &victim,
	}))
	e := as[oofevents.GameActionEvent](t, mustRecv(t, gaCh))
	if e.Action != oofevents.ActionDemo {
		t.Errorf("action %q, want demo", e.Action)
	}
	if e.VictimID != "pid-b" {
		t.Errorf("victim id %q, want pid-b", e.VictimID)
	}
}

func TestTranslateStatFeedUnknownNoGameAction(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	gaCh := subscribe(t, bus, oofevents.TypeGameAction)
	tr.Translate(env("StatfeedEvent", events.StatfeedEventData{
		MatchGuid:  "g1",
		EventName:  "SomeUnknownFeed",
		MainTarget: events.PlayerRef{Name: "Alice", PrimaryId: "pid-a", TeamNum: 0},
	}))
	mustNotRecv(t, gaCh)
}

func TestTranslateStatFeedAlsoEmitsStatFeedEvent(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	sfCh := subscribe(t, bus, oofevents.TypeStatFeed)
	gaCh := subscribe(t, bus, oofevents.TypeGameAction)
	tr.Translate(env("StatfeedEvent", events.StatfeedEventData{
		MatchGuid:  "g1",
		EventName:  "Shot",
		MainTarget: events.PlayerRef{Name: "Alice", PrimaryId: "pid-a", TeamNum: 0},
	}))
	as[oofevents.StatFeedEvent](t, mustRecv(t, sfCh))
	as[oofevents.GameActionEvent](t, mustRecv(t, gaCh))
}

func TestTranslateBallHitAlsoemitsBallHitEvent(t *testing.T) {
	bus, tr := newBusWithTranslator(t)
	bhCh := subscribe(t, bus, oofevents.TypeBallHit)
	gaCh := subscribe(t, bus, oofevents.TypeGameAction)
	tr.Translate(env("BallHit", events.BallHitData{
		MatchGuid: "g1",
		Players:   []events.PlayerRef{{Name: "Alice", PrimaryId: "pid-a", TeamNum: 1}},
		Ball:      events.BallHitBall{PreHitSpeed: 10},
	}))
	as[oofevents.BallHitEvent](t, mustRecv(t, bhCh))
	as[oofevents.GameActionEvent](t, mustRecv(t, gaCh))
}