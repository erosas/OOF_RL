package overlayhud

import (
	"time"

	"OOF_RL/internal/momentum"
	"OOF_RL/internal/oofevents"
)

// staleSnapshotAfter is a display-only fallback threshold for the HUD view model.
// It does not affect Momentum Engine state or persistence.
const staleSnapshotAfter = 10 * time.Second

// ViewModel is display-oriented Momentum state for future Overlay HUD rendering.
// It deliberately avoids exposing raw engine ownership or mutation details.
type ViewModel struct {
	MatchActive bool
	HasData     bool
	IsStale     bool

	BlueShare   float64
	OrangeShare float64

	StateLabel string
	Confidence float64
	Volatility float64

	LastUpdated time.Time
}

func (p *Plugin) momentumViewModel(now time.Time) (ViewModel, bool) {
	state, status, ok := p.momentumSnapshot()
	if !ok {
		return ViewModel{}, false
	}
	return mapMomentumViewModel(state, status, now), true
}

func mapMomentumViewModel(state momentum.MomentumState, status momentum.ServiceStatus, now time.Time) ViewModel {
	blue := state.Teams[oofevents.TeamBlue]
	orange := state.Teams[oofevents.TeamOrange]
	blueShare, orangeShare := shares(blue.MomentumInfluence, orange.MomentumInfluence)
	lastUpdated := state.LastEvent.OccurredAt

	return ViewModel{
		MatchActive: status.Active,
		HasData:     state.Sequence > 0,
		IsStale:     isStale(lastUpdated, now),
		BlueShare:   blueShare,
		OrangeShare: orangeShare,
		StateLabel:  stateLabel(blueShare, orangeShare, avg(blue.Confidence, orange.Confidence)),
		// Confidence and volatility are averaged for display smoothing only.
		// The engine remains the source of truth for team-level signals.
		Confidence:  clamp01(avg(blue.Confidence, orange.Confidence)),
		Volatility:  clamp01(avg(blue.Volatility, orange.Volatility)),
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

func stateLabel(blueShare, orangeShare, confidence float64) string {
	if confidence <= 0 {
		return "NO DATA"
	}
	diff := blueShare - orangeShare
	switch {
	case diff >= 0.12:
		return "BLUE PRESSURE"
	case diff <= -0.12:
		return "ORANGE PRESSURE"
	default:
		return "SHIFTING"
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
