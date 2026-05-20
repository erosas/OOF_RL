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
				MomentumInfluence: 0.62,
				Confidence:        0.70,
				Volatility:        0.20,
			},
			oofevents.TeamOrange: {
				MomentumInfluence: 0.38,
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
	if vm.DisplayState != displayStateBluePressure {
		t.Fatalf("DisplayState = %q, want %q", vm.DisplayState, displayStateBluePressure)
	}
	if vm.BlueShare <= vm.OrangeShare {
		t.Fatalf("expected blue share to lead, got %+v", vm)
	}
	if !almostEqual(vm.Confidence, 0.70) || !almostEqual(vm.Volatility, 0.20) {
		t.Fatalf("confidence/volatility = %f/%f, want 0.70/0.20", vm.Confidence, vm.Volatility)
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
	if vm.DisplayState != displayStateNoData {
		t.Fatalf("DisplayState = %q, want %q", vm.DisplayState, displayStateNoData)
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
	if vm.StateLabel != "STALE" {
		t.Fatalf("StateLabel = %q, want STALE", vm.StateLabel)
	}
	if vm.DisplayState != displayStateStale {
		t.Fatalf("DisplayState = %q, want %q", vm.DisplayState, displayStateStale)
	}
}

func TestMapMomentumViewModelMapsControlStates(t *testing.T) {
	now := time.Unix(100, 0)
	tests := []struct {
		name      string
		blue      float64
		orange    float64
		wantState string
		wantLabel string
	}{
		{name: "blue control", blue: 0.72, orange: 0.28, wantState: displayStateBlueControl, wantLabel: "BLUE CONTROL"},
		{name: "orange control", blue: 0.28, orange: 0.72, wantState: displayStateOrangeControl, wantLabel: "ORANGE CONTROL"},
		{name: "neutral", blue: 0.52, orange: 0.48, wantState: displayStateNeutral, wantLabel: "NEUTRAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := mapMomentumViewModel(momentum.MomentumState{
				Sequence: 1,
				Teams: map[oofevents.Team]momentum.TeamSignal{
					oofevents.TeamBlue:   {MomentumInfluence: tt.blue, EventDerivedControl: tt.blue * 10, Confidence: 0.7},
					oofevents.TeamOrange: {MomentumInfluence: tt.orange, EventDerivedControl: tt.orange * 10, Confidence: 0.7},
				},
				LastEvent: momentum.EventSignal{OccurredAt: now},
			}, momentum.ServiceStatus{Active: true}, now)

			if vm.DisplayState != tt.wantState || vm.StateLabel != tt.wantLabel {
				t.Fatalf("state = %q label = %q, want %q/%q", vm.DisplayState, vm.StateLabel, tt.wantState, tt.wantLabel)
			}
		})
	}
}

func TestMapMomentumViewModelUsesInfluenceShareForWheelAndControlShareForState(t *testing.T) {
	now := time.Unix(100, 0)
	vm := mapMomentumViewModel(momentum.MomentumState{
		Sequence: 1,
		Teams: map[oofevents.Team]momentum.TeamSignal{
			oofevents.TeamBlue: {
				MomentumInfluence:   3.0,
				EventDerivedControl: 3.0,
				Pressure:            0.44,
				Confidence:          0.70,
			},
			oofevents.TeamOrange: {
				MomentumInfluence:   7.0,
				EventDerivedControl: 1.0,
				Pressure:            0.56,
				Confidence:          0.70,
			},
		},
		LastEvent: momentum.EventSignal{OccurredAt: now},
	}, momentum.ServiceStatus{Active: true}, now)

	if !almostEqual(vm.BlueShare, 0.30) || !almostEqual(vm.OrangeShare, 0.70) {
		t.Fatalf("wheel shares = %f/%f, want influence-derived 0.30/0.70", vm.BlueShare, vm.OrangeShare)
	}
	if !almostEqual(vm.BlueControlShare, 0.75) || !almostEqual(vm.OrangeControlShare, 0.25) {
		t.Fatalf("control shares = %f/%f, want 0.75/0.25", vm.BlueControlShare, vm.OrangeControlShare)
	}
	if !almostEqual(vm.BluePressureShare, 0.44) || !almostEqual(vm.OrangePressureShare, 0.56) {
		t.Fatalf("pressure shares = %f/%f, want 0.44/0.56", vm.BluePressureShare, vm.OrangePressureShare)
	}
	if vm.DisplayState != displayStateBlueControl {
		t.Fatalf("DisplayState = %q, want %q", vm.DisplayState, displayStateBlueControl)
	}
}

func TestMapMomentumViewModelFallsBackToInfluenceShare(t *testing.T) {
	now := time.Unix(100, 0)
	vm := mapMomentumViewModel(momentum.MomentumState{
		Sequence: 1,
		Teams: map[oofevents.Team]momentum.TeamSignal{
			oofevents.TeamBlue:   {MomentumInfluence: 0.63, Confidence: 0.70},
			oofevents.TeamOrange: {MomentumInfluence: 0.37, Confidence: 0.70},
		},
		LastEvent: momentum.EventSignal{OccurredAt: now},
	}, momentum.ServiceStatus{Active: true}, now)

	if !almostEqual(vm.BlueShare, 0.63) || !almostEqual(vm.OrangeShare, 0.37) {
		t.Fatalf("wheel shares = %f/%f, want influence fallback 0.63/0.37", vm.BlueShare, vm.OrangeShare)
	}
	if !almostEqual(vm.BlueControlShare, vm.BlueShare) || !almostEqual(vm.OrangeControlShare, vm.OrangeShare) {
		t.Fatalf("control shares should fall back to wheel shares, got %f/%f vs %f/%f", vm.BlueControlShare, vm.OrangeControlShare, vm.BlueShare, vm.OrangeShare)
	}
	if !almostEqual(vm.BluePressureShare, vm.BlueShare) || !almostEqual(vm.OrangePressureShare, vm.OrangeShare) {
		t.Fatalf("pressure shares should fall back to wheel shares, got %f/%f vs %f/%f", vm.BluePressureShare, vm.OrangePressureShare, vm.BlueShare, vm.OrangeShare)
	}
}

func TestMapMomentumViewModelAddsDominantStatesWithMaxConfidence(t *testing.T) {
	now := time.Unix(100, 0)
	tests := []struct {
		name      string
		blue      float64
		orange    float64
		wantState string
		wantLabel string
	}{
		{name: "dominant blue", blue: 0.86, orange: 0.14, wantState: displayStateDominantBlue, wantLabel: "BLUE CONTROL"},
		{name: "dominant orange", blue: 0.13, orange: 0.87, wantState: displayStateDominantOrange, wantLabel: "ORANGE CONTROL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := mapMomentumViewModel(momentum.MomentumState{
				Sequence: 1,
				Teams: map[oofevents.Team]momentum.TeamSignal{
					oofevents.TeamBlue:   {MomentumInfluence: tt.blue, EventDerivedControl: tt.blue * 10, Confidence: 0.90},
					oofevents.TeamOrange: {MomentumInfluence: tt.orange, EventDerivedControl: tt.orange * 10, Confidence: 0.90},
				},
				LastEvent: momentum.EventSignal{OccurredAt: now},
			}, momentum.ServiceStatus{Active: true}, now)

			if vm.DisplayState != tt.wantState || vm.StateLabel != tt.wantLabel {
				t.Fatalf("state = %q label = %q, want %q/%q", vm.DisplayState, vm.StateLabel, tt.wantState, tt.wantLabel)
			}
			if vm.ConfidenceBucket != confidenceBucketMax {
				t.Fatalf("ConfidenceBucket = %q, want %q", vm.ConfidenceBucket, confidenceBucketMax)
			}
		})
	}
}

func TestMapMomentumViewModelDoesNotPromoteDominantWithoutMaxConfidence(t *testing.T) {
	now := time.Unix(100, 0)
	vm := mapMomentumViewModel(momentum.MomentumState{
		Sequence: 1,
		Teams: map[oofevents.Team]momentum.TeamSignal{
			oofevents.TeamBlue:   {MomentumInfluence: 0.86, EventDerivedControl: 8.6, Confidence: 0.70},
			oofevents.TeamOrange: {MomentumInfluence: 0.14, EventDerivedControl: 1.4, Confidence: 0.70},
		},
		LastEvent: momentum.EventSignal{OccurredAt: now},
	}, momentum.ServiceStatus{Active: true}, now)

	if vm.DisplayState != displayStateBlueControl {
		t.Fatalf("DisplayState = %q, want %q", vm.DisplayState, displayStateBlueControl)
	}
	if vm.ConfidenceBucket != confidenceBucketHigh {
		t.Fatalf("ConfidenceBucket = %q, want %q", vm.ConfidenceBucket, confidenceBucketHigh)
	}
}

func TestMapMomentumViewModelMapsConfidenceBuckets(t *testing.T) {
	now := time.Unix(100, 0)
	tests := []struct {
		name       string
		confidence float64
		want       string
	}{
		{name: "low", confidence: 0.20, want: confidenceBucketLow},
		{name: "medium", confidence: 0.36, want: confidenceBucketMedium},
		{name: "high", confidence: 0.62, want: confidenceBucketHigh},
		{name: "max", confidence: 0.82, want: confidenceBucketMax},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vm := mapMomentumViewModel(momentum.MomentumState{
				Sequence: 1,
				Teams: map[oofevents.Team]momentum.TeamSignal{
					oofevents.TeamBlue:   {MomentumInfluence: 0.50, Confidence: tt.confidence},
					oofevents.TeamOrange: {MomentumInfluence: 0.50, Confidence: tt.confidence},
				},
				LastEvent: momentum.EventSignal{OccurredAt: now},
			}, momentum.ServiceStatus{Active: true}, now)

			if vm.ConfidenceBucket != tt.want {
				t.Fatalf("ConfidenceBucket = %q, want %q", vm.ConfidenceBucket, tt.want)
			}
		})
	}
}

func TestMapMomentumViewModelMapsRecentEventDisplayContract(t *testing.T) {
	now := time.Unix(100, 0)
	vm := mapMomentumViewModel(momentum.MomentumState{
		Sequence: 1,
		Teams: map[oofevents.Team]momentum.TeamSignal{
			oofevents.TeamBlue:   {MomentumInfluence: 0.50, Confidence: 0.70},
			oofevents.TeamOrange: {MomentumInfluence: 0.50, Confidence: 0.70},
		},
		LastEvent: momentum.EventSignal{
			Action:     oofevents.ActionGoal,
			ActorTeam:  oofevents.TeamBlue,
			ImpactTeam: oofevents.TeamOrange,
			OccurredAt: now.Add(-recentEventEnergyWindow / 4),
		},
	}, momentum.ServiceStatus{Active: true}, now)

	if vm.RecentEventType != string(oofevents.ActionGoal) {
		t.Fatalf("RecentEventType = %q, want %q", vm.RecentEventType, oofevents.ActionGoal)
	}
	if vm.RecentEventTeam != string(oofevents.TeamOrange) {
		t.Fatalf("RecentEventTeam = %q, want impact team orange", vm.RecentEventTeam)
	}
	if !almostEqual(vm.RecentEventEnergy, 0.75) {
		t.Fatalf("RecentEventEnergy = %f, want 0.75", vm.RecentEventEnergy)
	}
}

func TestMapMomentumViewModelClearsExpiredRecentEventEnergy(t *testing.T) {
	now := time.Unix(100, 0)
	vm := mapMomentumViewModel(momentum.MomentumState{
		Sequence: 1,
		Teams: map[oofevents.Team]momentum.TeamSignal{
			oofevents.TeamBlue:   {MomentumInfluence: 0.50, Confidence: 0.70},
			oofevents.TeamOrange: {MomentumInfluence: 0.50, Confidence: 0.70},
		},
		LastEvent: momentum.EventSignal{
			Action:     oofevents.ActionShot,
			ActorTeam:  oofevents.TeamBlue,
			OccurredAt: now.Add(-recentEventEnergyWindow - time.Millisecond),
		},
	}, momentum.ServiceStatus{Active: true}, now)

	if vm.RecentEventEnergy != 0 {
		t.Fatalf("RecentEventEnergy = %f, want 0 after window", vm.RecentEventEnergy)
	}
	if vm.RecentEventTeam != "" || vm.RecentEventType != "" {
		t.Fatalf("expired recent event should clear team/type, got %q/%q", vm.RecentEventTeam, vm.RecentEventType)
	}
}

func TestMapMomentumViewModelMapsVolatileLowConfidenceState(t *testing.T) {
	now := time.Unix(100, 0)
	vm := mapMomentumViewModel(momentum.MomentumState{
		Sequence: 1,
		Teams: map[oofevents.Team]momentum.TeamSignal{
			oofevents.TeamBlue:   {MomentumInfluence: 0.55, Confidence: 0.14, Volatility: 0.85},
			oofevents.TeamOrange: {MomentumInfluence: 0.45, Confidence: 0.14, Volatility: 0.85},
		},
		LastEvent: momentum.EventSignal{OccurredAt: now},
	}, momentum.ServiceStatus{Active: true}, now)

	if vm.DisplayState != displayStateVolatile || vm.StateLabel != "CONTESTED" {
		t.Fatalf("state = %q label = %q, want volatile/CONTESTED", vm.DisplayState, vm.StateLabel)
	}
}

func TestMapMomentumViewModelMapsInactiveState(t *testing.T) {
	now := time.Unix(100, 0)
	vm := mapMomentumViewModel(momentum.MomentumState{
		Sequence: 1,
		Teams: map[oofevents.Team]momentum.TeamSignal{
			oofevents.TeamBlue:   {MomentumInfluence: 0.62, Confidence: 0.7},
			oofevents.TeamOrange: {MomentumInfluence: 0.38, Confidence: 0.7},
		},
		LastEvent: momentum.EventSignal{OccurredAt: now},
	}, momentum.ServiceStatus{Active: false}, now)

	if vm.DisplayState != displayStateInactive || vm.StateLabel != "INACTIVE" {
		t.Fatalf("state = %q label = %q, want inactive/INACTIVE", vm.DisplayState, vm.StateLabel)
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
