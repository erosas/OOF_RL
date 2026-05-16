package momentum

import (
	"encoding/json"
	"strings"
	"time"

	"OOF_RL/internal/events"
)

func NormalizeEnvelope(env events.Envelope, receivedAt time.Time) []NormalizedGameEvent {
	now := receivedAt.UnixMilli()
	switch env.Event {
	case "BallHit":
		var d events.BallHitData
		if json.Unmarshal(env.Data, &d) != nil || len(d.Players) == 0 {
			return nil
		}
		p := d.Players[0]
		team, ok := TeamFromNum(p.TeamNum)
		if !ok {
			return nil
		}
		return []NormalizedGameEvent{{
			Type:        EventBallHit,
			Team:        team,
			PlayerID:    p.PrimaryId,
			PlayerName:  p.Name,
			Time:        now,
			MatchGUID:   d.MatchGuid,
			SourceEvent: env.Event,
		}}
	case "GoalScored":
		var d events.GoalScoredData
		if json.Unmarshal(env.Data, &d) != nil {
			return nil
		}
		if d.Scorer.Name == "" && d.Scorer.PrimaryId == "" && d.Scorer.Shortcut == 0 {
			return nil
		}
		team, ok := TeamFromNum(d.Scorer.TeamNum)
		if !ok {
			return nil
		}
		matchClock := int(d.GoalTime)
		evs := []NormalizedGameEvent{{
			Type:        EventGoal,
			Team:        team,
			PlayerID:    d.Scorer.PrimaryId,
			PlayerName:  d.Scorer.Name,
			Time:        now,
			MatchClock:  &matchClock,
			MatchGUID:   d.MatchGuid,
			SourceEvent: env.Event,
		}}
		if d.Assister != nil {
			evs[0].AssisterID = d.Assister.PrimaryId
			evs = append(evs, NormalizedGameEvent{
				Type:        EventAssist,
				Team:        team,
				PlayerID:    d.Assister.PrimaryId,
				PlayerName:  d.Assister.Name,
				Time:        now,
				MatchClock:  &matchClock,
				MatchGUID:   d.MatchGuid,
				SourceEvent: env.Event,
			})
		}
		return evs
	case "StatfeedEvent":
		var d events.StatfeedEventData
		if json.Unmarshal(env.Data, &d) != nil {
			return nil
		}
		eventType, ok := statfeedType(d.EventName)
		if !ok {
			return nil
		}
		team, ok := TeamFromNum(d.MainTarget.TeamNum)
		if !ok {
			return nil
		}
		ev := NormalizedGameEvent{
			Type:        eventType,
			Team:        team,
			PlayerID:    d.MainTarget.PrimaryId,
			PlayerName:  d.MainTarget.Name,
			Time:        now,
			MatchGUID:   d.MatchGuid,
			SourceEvent: env.Event,
		}
		if d.SecondaryTarget != nil {
			ev.VictimID = d.SecondaryTarget.PrimaryId
		}
		return []NormalizedGameEvent{ev}
	default:
		return nil
	}
}

func statfeedType(name string) (EventType, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "shot":
		return EventShot, true
	case "save", "epicsave":
		return EventSave, true
	case "assist":
		return EventAssist, true
	case "demolish":
		return EventDemo, true
	case "goal", "owngoal":
		return EventGoal, true
	default:
		return "", false
	}
}
