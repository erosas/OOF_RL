package overlayhud

import (
	"time"

	"OOF_RL/internal/momentum"
	"OOF_RL/internal/oofevents"
)

// staleSnapshotAfter is a display-only fallback threshold for the HUD view model.
// It does not affect Momentum Engine state or persistence.
const staleSnapshotAfter = 10 * time.Second

const (
	displayStateNeutral        = "neutral"
	displayStateBluePressure   = "blue-pressure"
	displayStateOrangePressure = "orange-pressure"
	displayStateBlueControl    = "blue-control"
	displayStateOrangeControl  = "orange-control"
	displayStateVolatile       = "volatile"
	displayStateStale          = "stale"
	displayStateNoData         = "no-data"
	displayStateInactive       = "inactive"
)

const (
	displayNeutralSpreadMax = 0.08
	displayPressureShareMin = 0.58
	displayControlShareMin  = 0.70
	displayVolatilityMin    = 0.72
	displayConfidenceMin    = 0.20
)

// ViewModel is display-oriented Momentum state for future Overlay HUD rendering.
// It deliberately avoids exposing raw engine ownership or mutation details.
type ViewModel struct {
	MatchActive bool
	HasData     bool
	IsStale     bool

	BlueShare   float64
	OrangeShare float64

	DisplayState string
	StateLabel   string
	Confidence   float64
	Volatility   float64

	LastUpdated time.Time
}

func (p *Plugin) momentumViewModel(now time.Time) (ViewModel, bool) {
	return momentumViewModelFromProvider(p.momentum, now)
}

func mapMomentumViewModel(state momentum.MomentumState, status momentum.ServiceStatus, now time.Time) ViewModel {
	blue := state.Teams[oofevents.TeamBlue]
	orange := state.Teams[oofevents.TeamOrange]
	blueShare, orangeShare := shares(blue.MomentumInfluence, orange.MomentumInfluence)
	lastUpdated := state.LastEvent.OccurredAt
	confidence := clamp01(avg(blue.Confidence, orange.Confidence))
	volatility := clamp01(avg(blue.Volatility, orange.Volatility))
	isStale := isStale(lastUpdated, now)
	displayState := momentumDisplayState(blueShare, orangeShare, confidence, volatility, state.Sequence > 0, status.Active, isStale)

	return ViewModel{
		MatchActive:  status.Active,
		HasData:      state.Sequence > 0,
		IsStale:      isStale,
		BlueShare:    blueShare,
		OrangeShare:  orangeShare,
		DisplayState: displayState,
		StateLabel:   stateLabel(displayState),
		// Confidence and volatility are averaged for display smoothing only.
		// The engine remains the source of truth for team-level signals.
		Confidence:  confidence,
		Volatility:  volatility,
		LastUpdated: lastUpdated,
	}
}

func shares(blue, orange float64) (float64, float64) {
	blue = clamp01(blue)
	orange = clamp01(orange)
	total := blue + orange
	if total == 0 {
		return 0.5, 0.5
	}
	return blue / total, orange / total
}

func momentumDisplayState(blueShare, orangeShare, confidence, volatility float64, hasData, matchActive, stale bool) string {
	switch {
	case !hasData:
		return displayStateNoData
	case !matchActive:
		return displayStateInactive
	case stale:
		return displayStateStale
	case volatility >= displayVolatilityMin && confidence < displayConfidenceMin:
		return displayStateVolatile
	}

	diff := blueShare - orangeShare
	switch {
	case diff >= shareSpreadThreshold(displayControlShareMin):
		return displayStateBlueControl
	case diff <= -shareSpreadThreshold(displayControlShareMin):
		return displayStateOrangeControl
	case diff >= shareSpreadThreshold(displayPressureShareMin):
		return displayStateBluePressure
	case diff <= -shareSpreadThreshold(displayPressureShareMin):
		return displayStateOrangePressure
	case diff > displayNeutralSpreadMax:
		return displayStateBluePressure
	case diff < -displayNeutralSpreadMax:
		return displayStateOrangePressure
	default:
		return displayStateNeutral
	}
}

func shareSpreadThreshold(share float64) float64 {
	return share*2 - 1
}

func stateLabel(displayState string) string {
	switch displayState {
	case displayStateBluePressure:
		return "BLUE PRESSURE"
	case displayStateOrangePressure:
		return "ORANGE PRESSURE"
	case displayStateBlueControl:
		return "BLUE CONTROL"
	case displayStateOrangeControl:
		return "ORANGE CONTROL"
	case displayStateVolatile:
		return "VOLATILE"
	case displayStateStale:
		return "STALE"
	case displayStateNoData:
		return "NO DATA"
	case displayStateInactive:
		return "INACTIVE"
	default:
		return "NEUTRAL"
	}
}

func isStale(lastUpdated, now time.Time) bool {
	if lastUpdated.IsZero() {
		return true
	}
	return now.Sub(lastUpdated) > staleSnapshotAfter
}

func avg(a, b float64) float64 {
	return (a + b) / 2
}

func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}
