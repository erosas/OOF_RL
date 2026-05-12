package momentum

import (
	"encoding/json"
	"testing"
	"time"

	"OOF_RL/internal/events"
)

func updateStateEnvelope(t *testing.T, matchGUID string, players ...events.Player) events.Envelope {
	t.Helper()
	data, err := json.Marshal(events.UpdateStateData{
		MatchGuid: matchGUID,
		Players:   players,
		Game: events.GameState{
			TimeSeconds: 240,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return events.Envelope{Event: "UpdateState", Data: data}
}

func playerStats(id string, team, goals, shots, assists, saves, demos int) events.Player {
	return events.Player{
		Name:      id,
		PrimaryId: id,
		Shortcut:  1,
		TeamNum:   team,
		Goals:     goals,
		Shots:     shots,
		Assists:   assists,
		Saves:     saves,
		Demos:     demos,
	}
}

func TestUpdateStateFirstSnapshotIsBaseline(t *testing.T) {
	normalizer := NewUpdateStateDeltaNormalizer()
	env := updateStateEnvelope(t, "match-1", playerStats("blue-1", 0, 2, 3, 1, 4, 5))

	got := normalizer.NormalizeEnvelope(env, time.UnixMilli(1000))

	if len(got) != 0 {
		t.Fatalf("first snapshot should not emit events, got %+v", got)
	}
}

func TestUpdateStateStatDeltasEmitStrongEvents(t *testing.T) {
	normalizer := NewUpdateStateDeltaNormalizer()
	normalizer.NormalizeEnvelope(updateStateEnvelope(t, "match-1", playerStats("blue-1", 0, 0, 0, 0, 0, 0)), time.UnixMilli(1000))

	got := normalizer.NormalizeEnvelope(updateStateEnvelope(t, "match-1", playerStats("blue-1", 0, 1, 1, 1, 1, 1)), time.UnixMilli(1200))

	want := []EventType{EventShot, EventSave, EventDemo, EventAssist, EventGoal}
	if len(got) != len(want) {
		t.Fatalf("event count: got %d, want %d; events=%+v", len(got), len(want), got)
	}
	for i, wantType := range want {
		if got[i].Type != wantType {
			t.Fatalf("event %d type: got %s, want %s; events=%+v", i, got[i].Type, wantType, got)
		}
		if got[i].Team != TeamBlue {
			t.Fatalf("event %d team: got %s, want %s", i, got[i].Team, TeamBlue)
		}
		if got[i].SourceEvent != "UpdateStateDelta" {
			t.Fatalf("event %d source: got %s", i, got[i].SourceEvent)
		}
	}
}

func TestUpdateStateRepeatedSnapshotEmitsNoDuplicate(t *testing.T) {
	normalizer := NewUpdateStateDeltaNormalizer()
	env := updateStateEnvelope(t, "match-1", playerStats("blue-1", 0, 0, 0, 0, 0, 0))
	next := updateStateEnvelope(t, "match-1", playerStats("blue-1", 0, 0, 1, 0, 0, 0))
	normalizer.NormalizeEnvelope(env, time.UnixMilli(1000))
	first := normalizer.NormalizeEnvelope(next, time.UnixMilli(1200))
	second := normalizer.NormalizeEnvelope(next, time.UnixMilli(1400))

	if len(first) != 1 || first[0].Type != EventShot {
		t.Fatalf("first delta: got %+v, want one shot", first)
	}
	if len(second) != 0 {
		t.Fatalf("repeated snapshot should not duplicate, got %+v", second)
	}
}

func TestUpdateStateMatchChangeResetsToBaseline(t *testing.T) {
	normalizer := NewUpdateStateDeltaNormalizer()
	normalizer.NormalizeEnvelope(updateStateEnvelope(t, "match-1", playerStats("blue-1", 0, 0, 0, 0, 0, 0)), time.UnixMilli(1000))

	got := normalizer.NormalizeEnvelope(updateStateEnvelope(t, "match-2", playerStats("blue-1", 0, 1, 1, 0, 0, 0)), time.UnixMilli(1200))

	if len(got) != 0 {
		t.Fatalf("new match first snapshot should baseline, got %+v", got)
	}
}

func TestUpdateStateSuppressesRecentExplicitEvent(t *testing.T) {
	normalizer := NewUpdateStateDeltaNormalizer()
	normalizer.NormalizeEnvelope(updateStateEnvelope(t, "match-1", playerStats("blue-1", 0, 0, 0, 0, 0, 0)), time.UnixMilli(1000))
	normalizer.MarkExplicit(NormalizedGameEvent{
		Type:      EventShot,
		Team:      TeamBlue,
		PlayerID:  "blue-1",
		Time:      1100,
		MatchGUID: "match-1",
	})

	got := normalizer.NormalizeEnvelope(updateStateEnvelope(t, "match-1", playerStats("blue-1", 0, 0, 1, 0, 0, 0)), time.UnixMilli(1200))

	if len(got) != 0 {
		t.Fatalf("recent explicit shot should suppress update-state duplicate, got %+v", got)
	}
}

func TestUpdateStateSuppressesExplicitEventByPlayerName(t *testing.T) {
	normalizer := NewUpdateStateDeltaNormalizer()
	player := playerStats("steam-1", 1, 0, 0, 0, 0, 0)
	player.Name = "Mr Mung Beans"
	normalizer.NormalizeEnvelope(updateStateEnvelope(t, "match-1", player), time.UnixMilli(1000))
	normalizer.MarkExplicit(NormalizedGameEvent{
		Type:       EventGoal,
		Team:       TeamOrange,
		PlayerName: "Mr Mung Beans",
		Time:       1100,
		MatchGUID:  "match-1",
	})

	player.Goals = 1
	got := normalizer.NormalizeEnvelope(updateStateEnvelope(t, "match-1", player), time.UnixMilli(1200))

	if len(got) != 0 {
		t.Fatalf("recent explicit goal by player name should suppress update-state duplicate, got %+v", got)
	}
}
