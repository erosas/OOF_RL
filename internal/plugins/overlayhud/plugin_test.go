package overlayhud

import (
	"reflect"
	"testing"

	"OOF_RL/internal/config"
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

type fakeRegistry struct {
	provider momentum.SnapshotProvider
}

func (f *fakeRegistry) Get(_ string) (plugin.Plugin, bool)  { return nil, false }
func (f *fakeRegistry) List() []plugin.Plugin                { return nil }
func (f *fakeRegistry) Momentum() momentum.SnapshotProvider { return f.provider }
func (f *fakeRegistry) Config() *config.Config              { return &config.Config{} }

func TestInitWiresMomentumAndConfig(t *testing.T) {
	provider := &fakeMomentumProvider{
		state: momentum.MomentumState{MatchGUID: "init-test"},
	}
	p := &Plugin{launchShell: startManualShell}
	reg := &fakeRegistry{provider: provider}
	if err := p.Init(nil, reg, nil); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if p.momentum == nil {
		t.Fatal("Init should set momentum from registry")
	}
	if p.cfg == nil {
		t.Fatal("Init should set cfg from registry")
	}
	state, _, ok := p.momentumSnapshot()
	if !ok || state.MatchGUID != "init-test" {
		t.Fatalf("momentum not wired correctly: ok=%v state=%+v", ok, state)
	}
}

func TestNilProviderIsSafe(t *testing.T) {
	p := New(nil)

	if _, _, ok := p.momentumSnapshot(); ok {
		t.Fatal("nil provider should report unavailable snapshot")
	}
}
