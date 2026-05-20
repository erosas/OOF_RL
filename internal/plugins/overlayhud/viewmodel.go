package overlayhud

import (
	"time"

	"OOF_RL/internal/momentum"
	"OOF_RL/internal/oofevents"
)

// staleSnapshotAfter is a display-only fallback threshold for the HUD view model.
// It does not affect Momentum Engine state or persistence.
const staleSnapshotAfter = 10 * time.Second

// recentEventEnergyWindow is a display-only pulse decay window for the wheel.
// It does not change Momentum Engine weighting or runtime state.
const recentEventEnergyWindow = 4 * time.Second

const (
	displayStateNeutral        = "neutral"
	displayStateBluePressure   = "blue-pressure"
	displayStateOrangePressure = "orange-pressure"
	displayStateBlueControl    = "blue-control"
	displayStateOrangeControl  = "orange-control"
	displayStateDominantBlue   = "dominant-blue"
	displayStateDominantOrange = "dominant-orange"
	displayStateVolatile       = "volatile"
	displayStateStale          = "stale"
	displayStateNoData         = "no-data"
	displayStateInactive       = "inactive"
)

const (
	displayNeutralSpreadMax = 0.08
	displayPressureShareMin = 0.62
	displayPressureValueMin = 3.2
	displayControlValueMin  = 1.8
	displayControlShareMin  = 0.60
	displayDominantShareMin = 0.84
	displayVolatilityMin    = 0.65
)

const (
	confidenceBucketLow    = "low"
	confidenceBucketMedium = "medium"
	confidenceBucketHigh   = "high"
	confidenceBucketMax    = "max"
)

const (
	displayConfidenceMediumMin = 0.36
	displayConfidenceHighMin   = 0.62
	displayConfidenceMaxMin    = 0.82
)

const (
	recentEventGoalEnergy    = 1.00
	recentEventSaveEnergy    = 0.82
	recentEventShotEnergy    = 0.72
	recentEventDemoEnergy    = 0.72
	recentEventAssistEnergy  = 0.62
	recentEventBallHitEnergy = 0.28
)

// ViewModel is display-oriented Momentum state for future Overlay HUD rendering.
// It deliberately avoids exposing raw engine ownership or mutation details.
type ViewModel struct {
	MatchActive bool
	HasData     bool
	IsStale     bool

	BlueShare   float64
	OrangeShare float64

	BlueControlShare    float64
	OrangeControlShare  float64
	BluePressureShare   float64
	OrangePressureShare float64

	DisplayState     string
	StateLabel       string
	Confidence       float64
	ConfidenceBucket string
	Volatility       float64

	RecentEventEnergy float64
	RecentEventTeam   string
	RecentEventType   string

	LastUpdated time.Time
}

func (p *Plugin) momentumViewModel(now time.Time) (ViewModel, bool) {
	return momentumViewModelFromProvider(p.momentum, now)
}

func mapMomentumViewModel(state momentum.MomentumState, status momentum.ServiceStatus, now time.Time) ViewModel {
	blue := state.Teams[oofevents.TeamBlue]
	orange := state.Teams[oofevents.TeamOrange]
	fallbackBlueShare, fallbackOrangeShare := shares(blue.MomentumInfluence, orange.MomentumInfluence)
	blueControlShare, orangeControlShare := sharesWithFallback(blue.EventDerivedControl, orange.EventDerivedControl, fallbackBlueShare, fallbackOrangeShare)
	bluePressureShare, orangePressureShare := sharesWithFallback(blue.Pressure, orange.Pressure, fallbackBlueShare, fallbackOrangeShare)
	blueShare, orangeShare := fallbackBlueShare, fallbackOrangeShare
	lastUpdated := state.LastEvent.OccurredAt
	confidence := clamp01(max(blue.Confidence, orange.Confidence))
	confidenceBucket := confidenceBucket(confidence)
	volatility := clamp01(max(blue.Volatility, orange.Volatility))
	isStale := isStale(lastUpdated, now)
	recentEventEnergy, recentEventTeam, recentEventType := recentEventDisplay(state.LastEvent, now)
	displayState := momentumDisplayState(
		blueShare,
		orangeShare,
		blueControlShare,
		orangeControlShare,
		bluePressureShare,
		orangePressureShare,
		blue.Pressure,
		orange.Pressure,
		blue.EventDerivedControl,
		orange.EventDerivedControl,
		confidenceBucket,
		volatility,
		recentEventEnergy,
		recentEventTeam,
		recentEventType,
		state.Sequence > 0,
		status.Active,
		isStale,
	)

	return ViewModel{
		MatchActive:         status.Active,
		HasData:             state.Sequence > 0,
		IsStale:             isStale,
		BlueShare:           blueShare,
		OrangeShare:         orangeShare,
		BlueControlShare:    blueControlShare,
		OrangeControlShare:  orangeControlShare,
		BluePressureShare:   bluePressureShare,
		OrangePressureShare: orangePressureShare,
		DisplayState:        displayState,
		StateLabel:          stateLabel(displayState),
		// Confidence and volatility are averaged for display smoothing only.
		// The engine remains the source of truth for team-level signals.
		Confidence:       confidence,
		ConfidenceBucket: confidenceBucket,
		Volatility:       volatility,

		RecentEventEnergy: recentEventEnergy,
		RecentEventTeam:   recentEventTeam,
		RecentEventType:   recentEventType,

		LastUpdated: lastUpdated,
	}
}

func shares(blue, orange float64) (float64, float64) {
	blue = max0(blue)
	orange = max0(orange)
	total := blue + orange
	if total == 0 {
		return 0.5, 0.5
	}
	return blue / total, orange / total
}

func sharesWithFallback(blue, orange, fallbackBlue, fallbackOrange float64) (float64, float64) {
	blue = max0(blue)
	orange = max0(orange)
	if blue+orange == 0 {
		return fallbackBlue, fallbackOrange
	}
	return shares(blue, orange)
}

func momentumDisplayState(blueShare, orangeShare, blueControlShare, orangeControlShare, bluePressureShare, orangePressureShare, bluePressure, orangePressure, blueControl, orangeControl float64, confidenceBucket string, volatility float64, recentEventEnergy float64, recentEventTeam, recentEventType string, hasData, matchActive, stale bool) string {
	switch {
	case !hasData:
		return displayStateNoData
	case !matchActive:
		return displayStateInactive
	case stale:
		return displayStateStale
	case volatility >= displayVolatilityMin:
		return displayStateVolatile
	case recentEventEnergy > 0 && isPressureEvent(recentEventType):
		if state := recentPressureState(recentEventType, recentEventTeam, bluePressureShare, orangePressureShare); state != "" {
			return state
		}
	case blueShare >= displayDominantShareMin && confidenceBucket == confidenceBucketMax:
		return displayStateDominantBlue
	case orangeShare >= displayDominantShareMin && confidenceBucket == confidenceBucketMax:
		return displayStateDominantOrange
	case isPressureDominant(bluePressure, bluePressureShare):
		return displayStateBluePressure
	case isPressureDominant(orangePressure, orangePressureShare):
		return displayStateOrangePressure
	}

	switch {
	case blueControlShare >= displayControlShareMin && blueControl > displayControlValueMin:
		return displayStateBlueControl
	case orangeControlShare >= displayControlShareMin && orangeControl > displayControlValueMin:
		return displayStateOrangeControl
	}

	diff := blueShare - orangeShare
	switch {
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

func isPressureEvent(eventType string) bool {
	switch oofevents.ActionKind(eventType) {
	case oofevents.ActionGoal, oofevents.ActionShot, oofevents.ActionSave:
		return true
	default:
		return false
	}
}

func recentPressureState(eventType, eventTeam string, bluePressureShare, orangePressureShare float64) string {
	if oofevents.ActionKind(eventType) == oofevents.ActionSave {
		if bluePressureShare >= orangePressureShare {
			return displayStateBluePressure
		}
		return displayStateOrangePressure
	}
	switch oofevents.Team(eventTeam) {
	case oofevents.TeamBlue:
		return displayStateBluePressure
	case oofevents.TeamOrange:
		return displayStateOrangePressure
	default:
		if bluePressureShare >= orangePressureShare {
			return displayStateBluePressure
		}
		return displayStateOrangePressure
	}
}

func isPressureDominant(pressure, pressureShare float64) bool {
	return pressure >= displayPressureValueMin &&
		pressureShare >= displayPressureShareMin
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
	case displayStateDominantBlue:
		return "BLUE CONTROL"
	case displayStateDominantOrange:
		return "ORANGE CONTROL"
	case displayStateVolatile:
		return "CONTESTED"
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

func confidenceBucket(confidence float64) string {
	confidence = clamp01(confidence)
	switch {
	case confidence >= displayConfidenceMaxMin:
		return confidenceBucketMax
	case confidence >= displayConfidenceHighMin:
		return confidenceBucketHigh
	case confidence >= displayConfidenceMediumMin:
		return confidenceBucketMedium
	default:
		return confidenceBucketLow
	}
}

func recentEventDisplay(event momentum.EventSignal, now time.Time) (float64, string, string) {
	if event.Action == "" || event.OccurredAt.IsZero() {
		return 0, "", ""
	}
	age := now.Sub(event.OccurredAt)
	if age < 0 {
		age = 0
	}
	if age > recentEventEnergyWindow {
		return 0, "", ""
	}
	team := event.ImpactTeam
	if team == "" {
		team = event.ActorTeam
	}
	energy := recentEventBaseEnergy(event.Action)
	if energy <= 0 {
		return 0, "", ""
	}
	return clamp01(energy * (1 - age.Seconds()/recentEventEnergyWindow.Seconds())), string(team), string(event.Action)
}

func recentEventBaseEnergy(action oofevents.ActionKind) float64 {
	switch action {
	case oofevents.ActionGoal:
		return recentEventGoalEnergy
	case oofevents.ActionSave:
		return recentEventSaveEnergy
	case oofevents.ActionShot:
		return recentEventShotEnergy
	case oofevents.ActionDemo:
		return recentEventDemoEnergy
	case oofevents.ActionAssist:
		return recentEventAssistEnergy
	case oofevents.ActionBallHit:
		return recentEventBallHitEnergy
	default:
		return 0
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

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func max0(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
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
