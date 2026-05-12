package momentum

import (
	"encoding/json"
	"fmt"
	"time"

	"OOF_RL/internal/events"
)

const updateStateSuppressWindowMs int64 = 2500

type playerStatSnapshot struct {
	Goals   int
	Shots   int
	Assists int
	Saves   int
	Demos   int
}

type recentExplicitEvent struct {
	eventType EventType
	team      Team
	playerKey string
	matchGUID string
	time      int64
}

// UpdateStateDeltaNormalizer derives strong game events from UpdateState stat
// counter changes when Rocket League does not emit dedicated statfeed events.
// The first snapshot per player is only a baseline so mid-match startup does
// not create false shots, goals, saves, assists, or demos.
type UpdateStateDeltaNormalizer struct {
	matchGUID string
	players   map[string]playerStatSnapshot
	explicit  []recentExplicitEvent
}

func NewUpdateStateDeltaNormalizer() *UpdateStateDeltaNormalizer {
	return &UpdateStateDeltaNormalizer{
		players: make(map[string]playerStatSnapshot),
	}
}

func (n *UpdateStateDeltaNormalizer) Reset() {
	n.matchGUID = ""
	n.players = make(map[string]playerStatSnapshot)
	n.explicit = nil
}

func (n *UpdateStateDeltaNormalizer) MarkExplicit(ev NormalizedGameEvent) {
	playerKey := playerKeyForExplicit(ev)
	if ev.Type == EventBallHit || playerKey == "" {
		return
	}
	n.trimExplicit(ev.Time)
	n.explicit = append(n.explicit, recentExplicitEvent{
		eventType: ev.Type,
		team:      ev.Team,
		playerKey: playerKey,
		matchGUID: ev.MatchGUID,
		time:      ev.Time,
	})
}

func (n *UpdateStateDeltaNormalizer) NormalizeEnvelope(env events.Envelope, receivedAt time.Time) []NormalizedGameEvent {
	switch env.Event {
	case "MatchCreated", "MatchInitialized", "MatchEnded", "MatchDestroyed":
		n.Reset()
		return nil
	case "UpdateState":
	default:
		return nil
	}

	var d events.UpdateStateData
	if json.Unmarshal(env.Data, &d) != nil {
		return nil
	}
	if d.MatchGuid == "" || len(d.Players) == 0 {
		return nil
	}
	if n.matchGUID != "" && n.matchGUID != d.MatchGuid {
		n.Reset()
	}
	n.matchGUID = d.MatchGuid

	now := receivedAt.UnixMilli()
	n.trimExplicit(now)
	var matchClock *int
	if d.Game.TimeSeconds > 0 {
		clock := d.Game.TimeSeconds
		matchClock = &clock
	}

	var out []NormalizedGameEvent
	for _, p := range d.Players {
		team, ok := TeamFromNum(p.TeamNum)
		if !ok {
			continue
		}
		playerID := playerIDForUpdateState(p)
		if playerID == "" {
			continue
		}
		key := d.MatchGuid + ":" + playerID
		current := playerStatSnapshot{
			Goals:   p.Goals,
			Shots:   p.Shots,
			Assists: p.Assists,
			Saves:   p.Saves,
			Demos:   p.Demos,
		}
		previous, seen := n.players[key]
		n.players[key] = current
		if !seen {
			continue
		}

		out = appendStatDeltas(out, n, EventShot, current.Shots-previous.Shots, team, playerID, p.Name, d.MatchGuid, matchClock, now)
		out = appendStatDeltas(out, n, EventSave, current.Saves-previous.Saves, team, playerID, p.Name, d.MatchGuid, matchClock, now)
		out = appendStatDeltas(out, n, EventDemo, current.Demos-previous.Demos, team, playerID, p.Name, d.MatchGuid, matchClock, now)
		out = appendStatDeltas(out, n, EventAssist, current.Assists-previous.Assists, team, playerID, p.Name, d.MatchGuid, matchClock, now)
		out = appendStatDeltas(out, n, EventGoal, current.Goals-previous.Goals, team, playerID, p.Name, d.MatchGuid, matchClock, now)
	}
	return out
}

func appendStatDeltas(out []NormalizedGameEvent, normalizer *UpdateStateDeltaNormalizer, eventType EventType, delta int, team Team, playerID, playerName, matchGUID string, matchClock *int, now int64) []NormalizedGameEvent {
	if delta <= 0 {
		return out
	}
	if delta > 5 {
		delta = 5
	}
	for i := 0; i < delta; i++ {
		if normalizer.consumeExplicit(eventType, team, playerID, playerName, matchGUID, now) {
			continue
		}
		out = append(out, NormalizedGameEvent{
			Type:        eventType,
			Team:        team,
			PlayerID:    playerID,
			PlayerName:  playerName,
			Time:        now,
			MatchClock:  matchClock,
			MatchGUID:   matchGUID,
			SourceEvent: "UpdateStateDelta",
		})
	}
	return out
}

func (n *UpdateStateDeltaNormalizer) consumeExplicit(eventType EventType, team Team, playerID, playerName, matchGUID string, now int64) bool {
	playerKeys := map[string]bool{}
	if playerID != "" {
		playerKeys[playerID] = true
	}
	if playerName != "" {
		playerKeys["name:"+playerName] = true
	}
	for i, explicit := range n.explicit {
		if now-explicit.time > updateStateSuppressWindowMs {
			continue
		}
		if explicit.eventType == eventType && explicit.team == team && playerKeys[explicit.playerKey] && explicit.matchGUID == matchGUID {
			n.explicit = append(n.explicit[:i], n.explicit[i+1:]...)
			return true
		}
	}
	return false
}

func (n *UpdateStateDeltaNormalizer) trimExplicit(now int64) {
	keep := n.explicit[:0]
	for _, explicit := range n.explicit {
		if now-explicit.time <= updateStateSuppressWindowMs {
			keep = append(keep, explicit)
		}
	}
	n.explicit = keep
}

func playerIDForUpdateState(p events.Player) string {
	if p.PrimaryId != "" {
		return p.PrimaryId
	}
	if p.Shortcut != 0 {
		return fmt.Sprintf("shortcut:%d", p.Shortcut)
	}
	if p.Name != "" {
		return "name:" + p.Name
	}
	return ""
}

func playerKeyForExplicit(ev NormalizedGameEvent) string {
	if ev.PlayerID != "" {
		return ev.PlayerID
	}
	if ev.PlayerName != "" {
		return "name:" + ev.PlayerName
	}
	return ""
}
