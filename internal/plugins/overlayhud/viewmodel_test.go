package overlayhud

import (
	"math"
	"testing"
	"time"

	"OOF_RL/internal/momentum"
	"OOF_RL/internal/oofevents"
)

func TestMapMomentumViewModelWithBluePressure(t *testing.T) {
	now := time.Unix(100, 0)
	state := momentum.MomentumState{
		MatchGUID: "match-1",
		Sequence:  3,
		Teams: map[oofevents.Team]momentum.TeamSignal{
			oofevents.TeamBlue: {
				MomentumInfluence: 0.60,
				Confidence:        0.70,
				Volatility:        0.20,
			},
			oofevents.TeamOrange: {
				MomentumInfluence: 0.20,
				Confidence:        0.50,
				Volatility:        0.10,
			},
		},
		LastEvent: momentum.EventSignal{OccurredAt: now.Add(-time.Second)},
	}

	vm := mapMomentumViewModel(state, momentum.ServiceStatus{Active: true}, now)

	if !vm.MatchActive || !vm.HasData || vm.IsStale {
		t.Fatalf("unexpected activity/data state: %+v", vm)
	}
	if vm.StateLabel != "BLUE PRESSURE" {
		t.Fatalf("StateLabel = %q, want BLUE PRESSURE", vm.StateLabel)
	}
	if vm.BlueShare <= vm.OrangeShare {
		t.Fatalf("expected blue share to lead, got %+v", vm)
	}
	if !almostEqual(vm.Confidence, 0.60) || !almostEqual(vm.Volatility, 0.15) {
		t.Fatalf("confidence/volatility = %f/%f, want 0.60/0.15", vm.Confidence, vm.Volatility)
	}
}

func almostEqual(got, want float64) bool {
	return math.Abs(got-want) <= 1e-9
}

func TestMapMomentumViewModelHandlesEmptyState(t *testing.T) {
	vm := mapMomentumViewModel(momentum.MomentumState{}, momentum.ServiceStatus{}, time.Unix(100, 0))

	if vm.MatchActive || vm.HasData != false || !vm.IsStale {
		t.Fatalf("empty state flags = %+v", vm)
	}
	if vm.BlueShare != 0.5 || vm.OrangeShare != 0.5 {
		t.Fatalf("empty shares = %f/%f, want 0.5/0.5", vm.BlueShare, vm.OrangeShare)
	}
	if vm.StateLabel != "NO DATA" {
		t.Fatalf("StateLabel = %q, want NO DATA", vm.StateLabel)
	}
}

func TestMapMomentumViewModelMarksStaleState(t *testing.T) {
	now := time.Unix(100, 0)
	state := momentum.MomentumState{
		Sequence: 1,
		Teams: map[oofevents.Team]momentum.TeamSignal{
			oofevents.TeamBlue:   {MomentumInfluence: 0.2, Confidence: 0.5},
			oofevents.TeamOrange: {MomentumInfluence: 0.2, Confidence: 0.5},
		},
		LastEvent: momentum.EventSignal{OccurredAt: now.Add(-staleSnapshotAfter - time.Millisecond)},
	}

	vm := mapMomentumViewModel(state, momentum.ServiceStatus{Active: true}, now)

	if !vm.IsStale {
		t.Fatalf("expected stale view model, got %+v", vm)
	}
	if vm.StateLabel != "SHIFTING" {
		t.Fatalf("StateLabel = %q, want SHIFTING", vm.StateLabel)
	}
}

func TestMomentumViewModelDoesNotMutateMomentumState(t *testing.T) {
	now := time.Unix(100, 0)
	state := momentum.MomentumState{
		Sequence: 1,
		Teams: map[oofevents.Team]momentum.TeamSignal{
			oofevents.TeamBlue: {MomentumInfluence: 0.7, Confidence: 0.8},
		},
		LastEvent: momentum.EventSignal{OccurredAt: now},
	}
	provider := &fakeMomentumProvider{
		state:  state,
		status: momentum.ServiceStatus{Active: true},
	}
	p := New(provider)

	vm, ok := p.momentumViewModel(now)
	if !ok {
		t.Fatal("expected view model")
	}
	vm.BlueShare = 0

	next, ok := p.momentumViewModel(now)
	if !ok {
		t.Fatal("expected second view model")
	}
	if next.BlueShare == 0 {
		t.Fatal("mutating returned view model should not mutate provider state")
	}
	if provider.state.Teams[oofevents.TeamBlue].MomentumInfluence != 0.7 {
		t.Fatalf("provider state mutated: %+v", provider.state.Teams[oofevents.TeamBlue])
	}
}
