package rlevents

import (
	"encoding/json"
	"log"

	"OOF_RL/internal/events"
	"OOF_RL/internal/oofevents"
)

// Translator converts raw RL events.Envelope values into typed OOF events
// and publishes them on the bus. It maintains minimal state for derived events.
type Translator struct {
	bus          oofevents.PluginBus
	currentGUID  string
	lastOvertime bool
}

// New returns a Translator that publishes on bus.
// bus should be the PluginBus scoped to the RL translator (pluginID = "").
func New(bus oofevents.PluginBus) *Translator {
	return &Translator{bus: bus}
}

// Translate maps one RL envelope to zero or more OOF events.
func (t *Translator) Translate(env events.Envelope) {
	switch env.Event {
	case "MatchCreated", "MatchInitialized":
		t.onMatchStart(env)
	case "MatchEnded":
		t.onMatchEnded(env)
	case "MatchDestroyed":
		t.bus.PublishAuthoritative(oofevents.NewMatchDestroyed())
		t.currentGUID = ""
		t.lastOvertime = false
	case "GoalScored":
		t.onGoalScored(env)
	case "StatfeedEvent":
		t.onStatFeed(env)
	case "ClockUpdatedSeconds":
		t.onClockUpdated(env)
	case "BallHit":
		t.onBallHit(env)
	case "CrossbarHit":
		t.onCrossbarHit(env)
	case "UpdateState":
		t.onUpdateState(env)
	}
}

// advanceGUID detects mid-session GUID changes (reconnect) and updates state.
func (t *Translator) advanceGUID(newGUID string) {
	if newGUID == "" {
		return
	}
	if t.currentGUID != "" && t.currentGUID != newGUID {
		t.bus.PublishInferred(oofevents.NewMatchRestarted(newGUID, t.currentGUID))
	}
	t.currentGUID = newGUID
}

func (t *Translator) onMatchStart(env events.Envelope) {
	d, ok := decode[events.MatchGuidData](env, env.Event)
	if !ok || d.MatchGuid == "" {
		return
	}
	prev := t.currentGUID
	t.advanceGUID(d.MatchGuid)
	if d.MatchGuid == prev {
		return // same GUID already seen; don't re-emit match.started
	}
	t.bus.PublishAuthoritative(oofevents.NewMatchStarted(d.MatchGuid))
}

func (t *Translator) onMatchEnded(env events.Envelope) {
	d, ok := decode[events.MatchEndedData](env, "MatchEnded")
	if !ok {
		return
	}
	t.bus.PublishAuthoritative(oofevents.NewMatchEnded(d.MatchGuid, d.WinnerTeamNum))
	t.lastOvertime = false
	t.currentGUID = ""
}

func (t *Translator) onGoalScored(env events.Envelope) {
	d, ok := decode[events.GoalScoredData](env, "GoalScored")
	if !ok {
		return
	}
	t.bus.PublishAuthoritative(oofevents.NewGoalScored(
		d.MatchGuid,
		d.Scorer.Name, d.Scorer.Shortcut,
		nameOrEmpty(d.Assister), shortcutOrZero(d.Assister),
		d.BallLastTouch.Player.Shortcut,
		d.GoalSpeed, d.GoalTime,
		d.ImpactLocation.X, d.ImpactLocation.Y, d.ImpactLocation.Z,
		d.Scorer.TeamNum,
	))
}

func (t *Translator) onStatFeed(env events.Envelope) {
	d, ok := decode[events.StatfeedEventData](env, "StatfeedEvent")
	if !ok {
		return
	}
	t.bus.PublishAuthoritative(oofevents.NewStatFeed(
		d.MatchGuid, d.EventName,
		d.MainTarget.Name, d.MainTarget.Shortcut, d.MainTarget.TeamNum,
		nameOrEmpty(d.SecondaryTarget), shortcutOrZero(d.SecondaryTarget),
	))
}

func (t *Translator) onClockUpdated(env events.Envelope) {
	d, ok := decode[events.ClockData](env, "ClockUpdatedSeconds")
	if !ok {
		return
	}
	if d.BOvertime && !t.lastOvertime {
		t.bus.PublishInferred(oofevents.NewOvertimeStarted(d.MatchGuid, d.TimeSeconds))
	}
	t.lastOvertime = d.BOvertime
	t.bus.PublishAuthoritative(oofevents.NewClockUpdated(d.MatchGuid, d.TimeSeconds, d.BOvertime))
}

func (t *Translator) onBallHit(env events.Envelope) {
	d, ok := decode[events.BallHitData](env, "BallHit")
	if !ok {
		return
	}
	playerName, playerPrimaryID := "", ""
	playerShortcut := 0
	if len(d.Players) > 0 {
		playerName = d.Players[0].Name
		playerPrimaryID = d.Players[0].PrimaryId
		playerShortcut = d.Players[0].Shortcut
	}
	t.bus.PublishAuthoritative(oofevents.NewBallHit(
		d.MatchGuid, playerName, playerPrimaryID, playerShortcut,
		d.Ball.PreHitSpeed, d.Ball.PostHitSpeed,
		d.Ball.Location.X, d.Ball.Location.Y, d.Ball.Location.Z,
	))
}

func (t *Translator) onCrossbarHit(env events.Envelope) {
	d, ok := decode[events.CrossbarHitData](env, "CrossbarHit")
	if !ok {
		return
	}
	t.bus.PublishAuthoritative(oofevents.NewCrossbarHit(
		d.MatchGuid, d.BallLastTouch.Player.Name,
		d.BallSpeed, d.ImpactForce,
	))
}

func (t *Translator) onUpdateState(env events.Envelope) {
	d, ok := decode[events.UpdateStateData](env, "UpdateState")
	if !ok {
		return
	}
	t.advanceGUID(d.MatchGuid)

	players := make([]oofevents.PlayerSnapshot, len(d.Players))
	for i, p := range d.Players {
		players[i] = oofevents.PlayerSnapshot{
			Name:           p.Name,
			PrimaryID:      p.PrimaryId,
			Shortcut:       p.Shortcut,
			TeamNum:        p.TeamNum,
			Score:          p.Score,
			Goals:          p.Goals,
			Shots:          p.Shots,
			Assists:        p.Assists,
			Saves:          p.Saves,
			Touches:        p.Touches,
			CarTouches:     p.CarTouches,
			Demos:          p.Demos,
			Speed:          p.Speed,
			Boost:          p.Boost,
			IsBoosting:     p.BBoosting,
			IsOnWall:       p.BOnWall,
			IsPowersliding: p.BPowersliding,
			IsDemolished:   p.BDemolished,
			IsSupersonic:   p.BSupersonic,
		}
	}

	teams := make([]oofevents.TeamSnapshot, len(d.Game.Teams))
	for i, tm := range d.Game.Teams {
		teams[i] = oofevents.TeamSnapshot{
			Name:           tm.Name,
			TeamNum:        tm.TeamNum,
			Score:          tm.Score,
			ColorPrimary:   tm.ColorPrimary,
			ColorSecondary: tm.ColorSecondary,
		}
	}

	t.bus.PublishAuthoritative(oofevents.NewStateUpdated(d.MatchGuid, players, oofevents.GameSnapshot{
		Teams:       teams,
		TimeSeconds: d.Game.TimeSeconds,
		IsOvertime:  d.Game.BOvertime,
		IsReplay:    d.Game.BReplay,
		Ball:        oofevents.BallSnapshot{Speed: d.Game.Ball.Speed},
		Arena:       d.Game.Arena,
		Playlist:    d.Game.Playlist,
		HasWinner:   d.Game.BHasWinner,
		Winner:      d.Game.Winner,
	}))
}

func decode[T any](env events.Envelope, label string) (T, bool) {
	var d T
	if err := json.Unmarshal(env.Data, &d); err != nil {
		log.Printf("[rlevents] %s decode: %v", label, err)
		return d, false
	}
	return d, true
}

func nameOrEmpty(p *events.PlayerRef) string {
	if p == nil {
		return ""
	}
	return p.Name
}

func shortcutOrZero(p *events.PlayerRef) int {
	if p == nil {
		return 0
	}
	return p.Shortcut
}