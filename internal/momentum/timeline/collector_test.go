package timeline

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"OOF_RL/internal/momentum"
	"OOF_RL/internal/oofevents"
)

type fakeProvider struct {
	state  momentum.MomentumState
	status momentum.ServiceStatus
}

var _ SnapshotProvider = (*Collector)(nil)

func TestSnapshotProviderExposesReadOnlyMethods(t *testing.T) {
	providerType := reflect.TypeOf((*SnapshotProvider)(nil)).Elem()

	if _, ok := providerType.MethodByName("Snapshot"); !ok {
		t.Fatal("SnapshotProvider missing Snapshot method")
	}
	for _, name := range []string{
		"HandleGameAction",
		"Reset",
		"MarkMatchEnded",
		"Clear",
	} {
		if _, ok := providerType.MethodByName(name); ok {
			t.Fatalf("SnapshotProvider exposes mutating method %s", name)
		}
	}
}

func (f *fakeProvider) Snapshot() momentum.MomentumState {
	return f.state
}

func (f *fakeProvider) Status() momentum.ServiceStatus {
	return f.status
}

func TestCollectorRecordsSupportedGameAction(t *testing.T) {
	at := time.Unix(100, 0)
	event := withTime(oofevents.NewGameAction("match-1", oofevents.ActionGoal, oofevents.TeamBlue, "pid-a", "Alice"), at)
	provider := &fakeProvider{state: stateFor(event, 7)}
	collector := NewCollector(provider, Config{MaxEntries: 4})

	if !collector.HandleGameAction(event) {
		t.Fatal("HandleGameAction returned false for supported goal event")
	}

	snapshot := collector.Snapshot()
	if snapshot.MatchGUID != "match-1" {
		t.Fatalf("MatchGUID = %q, want match-1", snapshot.MatchGUID)
	}
	if len(snapshot.Entries) != 1 {
		t.Fatalf("entry count = %d, want 1", len(snapshot.Entries))
	}
	entry := snapshot.Entries[0]
	if entry.Index != 0 || entry.Action != oofevents.ActionGoal || entry.ActorTeam != oofevents.TeamBlue {
		t.Fatalf("entry metadata = %+v", entry)
	}
	if entry.PlayerID != "pid-a" || entry.PlayerName != "Alice" || !entry.OccurredAt.Equal(at) {
		t.Fatalf("player/time metadata = %+v", entry)
	}
	if entry.MomentumSequence != 7 {
		t.Fatalf("MomentumSequence = %d, want 7", entry.MomentumSequence)
	}
}

func TestCollectorSamplesMomentumValues(t *testing.T) {
	event := oofevents.NewGameAction("match-1", oofevents.ActionSave, oofevents.TeamOrange, "pid-b", "Bob", oofevents.WithEpicSave())
	provider := &fakeProvider{state: momentum.MomentumState{
		MatchGUID: "match-1",
		Sequence:  3,
		Teams: map[oofevents.Team]momentum.TeamSignal{
			oofevents.TeamBlue: {
				Pressure:            1,
				MomentumInfluence:   2,
				ContestInvolvement:  3,
				EventDerivedControl: 4,
				Confidence:          0.5,
				Volatility:          6,
			},
			oofevents.TeamOrange: {
				Pressure:            7,
				MomentumInfluence:   8,
				ContestInvolvement:  9,
				EventDerivedControl: 10,
				Confidence:          0.75,
				Volatility:          12,
			},
		},
		LastEvent: momentum.EventSignal{
			Action:     event.Action,
			ActorTeam:  event.Team,
			ImpactTeam: oofevents.TeamOrange,
			OccurredAt: event.OccurredAt(),
			MatchGUID:  event.MatchGUID(),
		},
	}}
	collector := NewCollector(provider, Config{})
	collector.HandleGameAction(event)

	entry := collector.Snapshot().Entries[0]
	if entry.Blue.Pressure != 1 || entry.Blue.EventDerivedControl != 4 || entry.Blue.Confidence != 0.5 {
		t.Fatalf("blue sample = %+v", entry.Blue)
	}
	if entry.Orange.Pressure != 7 || entry.Orange.Volatility != 12 || entry.Orange.Confidence != 0.75 {
		t.Fatalf("orange sample = %+v", entry.Orange)
	}
	if !entry.IsEpicSave || entry.Category != "contest" {
		t.Fatalf("save flags/category = %+v", entry)
	}
}

func TestCollectorBoundsBuffer(t *testing.T) {
	collector := NewCollector(&fakeProvider{}, Config{MaxEntries: 2})

	collector.HandleGameAction(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice"))
	collector.HandleGameAction(oofevents.NewGameAction("match-1", oofevents.ActionSave, oofevents.TeamOrange, "pid-b", "Bob"))
	collector.HandleGameAction(oofevents.NewGameAction("match-1", oofevents.ActionDemo, oofevents.TeamBlue, "pid-c", "Cora", oofevents.WithVictim("pid-b")))

	snapshot := collector.Snapshot()
	if len(snapshot.Entries) != 2 {
		t.Fatalf("entry count = %d, want bounded 2", len(snapshot.Entries))
	}
	if snapshot.Entries[0].Index != 1 || snapshot.Entries[1].Index != 2 {
		t.Fatalf("remaining indexes = %+v", snapshot.Entries)
	}
	if snapshot.NextIndex != 3 {
		t.Fatalf("NextIndex = %d, want 3", snapshot.NextIndex)
	}
}

func TestCollectorLifecycleResetEndAndClear(t *testing.T) {
	collector := NewCollector(&fakeProvider{}, Config{})
	collector.HandleGameAction(oofevents.NewGameAction("match-1", oofevents.ActionGoal, oofevents.TeamBlue, "pid-a", "Alice"))

	collector.MarkMatchEnded("match-1")
	if collector.HandleGameAction(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamOrange, "pid-b", "Bob")) {
		t.Fatal("collector recorded an event after match end")
	}
	ended := collector.Snapshot()
	if !ended.MatchEnded || ended.EndedReason != "match.ended" || len(ended.Entries) != 1 {
		t.Fatalf("ended snapshot = %+v", ended)
	}

	collector.Reset("match-2")
	reset := collector.Snapshot()
	if reset.MatchGUID != "match-2" || reset.MatchEnded || len(reset.Entries) != 0 || reset.NextIndex != 0 {
		t.Fatalf("reset snapshot = %+v", reset)
	}

	collector.HandleGameAction(oofevents.NewGameAction("match-2", oofevents.ActionShot, oofevents.TeamOrange, "pid-b", "Bob"))
	collector.Clear()
	cleared := collector.Snapshot()
	if cleared.MatchGUID != "" || cleared.MatchEnded || len(cleared.Entries) != 0 {
		t.Fatalf("cleared snapshot = %+v", cleared)
	}
}

func TestCollectorSnapshotReturnsCopy(t *testing.T) {
	collector := NewCollector(&fakeProvider{}, Config{})
	collector.HandleGameAction(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice"))

	snapshot := collector.Snapshot()
	snapshot.Entries[0].PlayerName = "mutated"
	snapshot.Entries = append(snapshot.Entries, TimelineEntry{PlayerName: "extra"})

	next := collector.Snapshot()
	if len(next.Entries) != 1 {
		t.Fatalf("entry count = %d, want 1", len(next.Entries))
	}
	if next.Entries[0].PlayerName != "Alice" {
		t.Fatalf("stored player name = %q, want Alice", next.Entries[0].PlayerName)
	}
}

func TestCollectorIgnoresUnsupportedEvents(t *testing.T) {
	collector := NewCollector(&fakeProvider{}, Config{})

	if collector.HandleGameAction(oofevents.NewGameAction("match-1", oofevents.ActionKind("boost_grab"), oofevents.TeamBlue, "pid-a", "Alice")) {
		t.Fatal("unsupported action was recorded")
	}
	if collector.HandleGameAction(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.Team("green"), "pid-a", "Alice")) {
		t.Fatal("unsupported team was recorded")
	}
	if len(collector.Snapshot().Entries) != 0 {
		t.Fatalf("unsupported events changed snapshot: %+v", collector.Snapshot())
	}
}

func TestCollectorResetsWhenMatchGUIDChanges(t *testing.T) {
	collector := NewCollector(&fakeProvider{}, Config{})
	collector.HandleGameAction(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice"))
	collector.HandleGameAction(oofevents.NewGameAction("match-2", oofevents.ActionGoal, oofevents.TeamOrange, "pid-b", "Bob"))

	snapshot := collector.Snapshot()
	if snapshot.MatchGUID != "match-2" || len(snapshot.Entries) != 1 || snapshot.Entries[0].Index != 0 {
		t.Fatalf("snapshot after GUID change = %+v", snapshot)
	}
}

func TestCollectorHasNoOverlayHUDDependency(t *testing.T) {
	source, err := os.ReadFile("collector.go")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.Contains(string(source), "internal/plugins/overlayhud") {
		t.Fatal("collector must not import Overlay HUD")
	}
}

func stateFor(event oofevents.GameActionEvent, sequence int) momentum.MomentumState {
	return momentum.MomentumState{
		MatchGUID: event.MatchGUID(),
		Sequence:  sequence,
		Teams: map[oofevents.Team]momentum.TeamSignal{
			oofevents.TeamBlue: {
				Pressure:            2,
				MomentumInfluence:   3,
				ContestInvolvement:  4,
				EventDerivedControl: 5,
				Confidence:          0.6,
				Volatility:          7,
			},
			oofevents.TeamOrange: {},
		},
		LastEvent: momentum.EventSignal{
			Action:     event.Action,
			ActorTeam:  event.Team,
			ImpactTeam: event.Team,
			PlayerID:   event.PlayerID,
			PlayerName: event.PlayerName,
			VictimID:   event.VictimID,
			IsOwnGoal:  event.IsOwnGoal,
			IsEpicSave: event.IsEpicSave,
			OccurredAt: event.OccurredAt(),
			MatchGUID:  event.MatchGUID(),
		},
	}
}

func withTime(event oofevents.GameActionEvent, at time.Time) oofevents.GameActionEvent {
	event.At = at
	return event
}
