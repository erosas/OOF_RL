package overlayhud

import (
	"reflect"
	"testing"

	"OOF_RL/internal/momentum"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/plugin"
)

var _ plugin.Plugin = (*Plugin)(nil)

type fakeMomentumProvider struct {
	state  momentum.MomentumState
	status momentum.ServiceStatus
}

func (f *fakeMomentumProvider) Snapshot() momentum.MomentumState { return f.state }
func (f *fakeMomentumProvider) Status() momentum.ServiceStatus   { return f.status }

func TestNewStoresReadOnlyMomentumProvider(t *testing.T) {
	provider := &fakeMomentumProvider{
		state: momentum.MomentumState{
			MatchGUID: "match-1",
			Sequence:  7,
			Teams: map[oofevents.Team]momentum.TeamSignal{
				oofevents.TeamBlue: {Pressure: 0.25},
			},
		},
		status: momentum.ServiceStatus{Active: true, Reason: "test"},
	}
	p := New(provider)

	state, status, ok := p.momentumSnapshot()
	if !ok {
		t.Fatal("momentumSnapshot should report provider availability")
	}
	if state.MatchGUID != "match-1" || state.Sequence != 7 {
		t.Fatalf("snapshot = %+v, want match-1 sequence 7", state)
	}
	if !status.Active || status.Reason != "test" {
		t.Fatalf("status = %+v, want active test", status)
	}
}

func TestPluginDoesNotExposeMomentumMutation(t *testing.T) {
	providerType := reflect.TypeOf((*momentum.SnapshotProvider)(nil)).Elem()
	for _, name := range []string{
		"HandleGameAction",
		"HandleMatchStarted",
		"HandleMatchRestarted",
		"HandleMatchEnded",
		"HandleMatchDestroyed",
		"Reset",
		"MarkMatchEnded",
	} {
		if _, ok := providerType.MethodByName(name); ok {
			t.Fatalf("SnapshotProvider exposes mutating method %s", name)
		}
	}
}

func TestNilProviderIsSafe(t *testing.T) {
	p := New(nil)

	if _, _, ok := p.momentumSnapshot(); ok {
		t.Fatal("nil provider should report unavailable snapshot")
	}
}
