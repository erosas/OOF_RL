package rlevents_test

import (
	"testing"

	"OOF_RL/internal/events"
	"OOF_RL/internal/oofevents"
)

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

func TestTranslateBallHitAlsoEmitsBallHitEvent(t *testing.T) {
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