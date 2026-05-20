package overlayhud

import "math"

const (
	renderViewBox          = "0 0 1024 1024"
	renderCenterX          = 512
	renderCenterY          = 512
	renderStartDeg         = -90
	renderFullSweepDeg     = 360
	renderWheelSegments    = 96
	renderWheelTicks       = 120
	renderSegmentStepDeg   = float64(renderFullSweepDeg) / float64(renderWheelSegments)
	renderTickStepDeg      = float64(renderFullSweepDeg) / float64(renderWheelTicks)
	renderTimerPlaceholder = "--:--"
)

const (
	displayResponsePressureSensitivity   = 1.3
	displayResponseVolatilitySensitivity = 1.3
	displayResponseEventReactiveness     = 1.8
	displayResponseHoldDuration          = 1.0
	displayResponseDecaySpeed            = 1.0
	displayResponseReactionSpeed         = 1.4
	displayResponsePulseDuration         = 1.2

	displayVisualStateNeutral  = 1.1
	displayVisualStatePressure = 1.95
	displayVisualStateControl  = 1.75
	displayVisualStateVolatile = 1.8
	displayVisualStateDominant = 1.35
)

// RenderModel contains SVG/display primitives for the future Overlay HUD renderer.
// It intentionally avoids SVG path strings, markup, runtime ownership, or rendering.
type RenderModel struct {
	ViewBox string
	CenterX float64
	CenterY float64

	Groups RenderGroups

	BlueArc   ArcModel
	OrangeArc ArcModel

	Confidence    ConfidenceModel
	Segments      []WheelSegment
	Ticks         []WheelTick
	RecentEvent   RecentEventModel
	ContestedLine ContestedLineModel
	Diagnostics   SignalDiagnosticsModel

	Center CenterModel

	DisplayState string
	StateClasses []string
	StyleVars    string
	SeamAngleDeg float64

	MatchActive bool
	HasData     bool
	IsStale     bool
}

type RenderGroups struct {
	Root               string
	Background         string
	OuterAura          string
	OuterStreaks       string
	OuterSparks        string
	Frame              string
	SegmentUnderlay    string
	SegmentActive      string
	SegmentBevels      string
	InnerTickRing      string
	CenterColorWashes  string
	CenterTexture      string
	CenterRim          string
	ContestedFrontLine string
	TextLayer          string
	Badge              string
	DebugOverlays      string
	MomentumArcs       string
	ConfidenceRing     string
	VolatilityRing     string
	CenterCore         string
	Labels             string
	StateOverlay       string
}

type ArcModel struct {
	Share     float64
	StartDeg  float64
	EndDeg    float64
	SweepDeg  float64
	ClassName string
}

type ConfidenceModel struct {
	Value     float64
	Bucket    string
	Intensity float64
	ClassName string
}

type WheelSegment struct {
	Index     int
	Active    bool
	Intensity float64
	AngleDeg  float64
	Owner     string
	IsSeam    bool
	ClassName string
}

type WheelTick struct {
	Index     int
	AngleDeg  float64
	Owner     string
	ClassName string
}

type RecentEventModel struct {
	Energy    float64
	Team      string
	Type      string
	ClassName string
}

type ContestedLineModel struct {
	Active    bool
	AngleDeg  float64
	BandDeg   float64
	Intensity float64
	ClassName string
}

type SignalDiagnosticsModel struct {
	BlueShare           float64
	OrangeShare         float64
	BluePressureShare   float64
	OrangePressureShare float64
	BlueControlShare    float64
	OrangeControlShare  float64
	ConfidenceBucket    string
	Volatility          float64
}

type CenterModel struct {
	PrimaryText string
	StateLabel  string
}

func buildRenderModel(vm ViewModel) RenderModel {
	blueShare, orangeShare := shares(vm.BlueShare, vm.OrangeShare)
	bluePressureShare, orangePressureShare := sharesWithFallback(vm.BluePressureShare, vm.OrangePressureShare, blueShare, orangeShare)
	visualBluePressure, visualOrangePressure := applyPressureResponse(bluePressureShare, orangePressureShare)
	blueArc := buildArc(blueShare, renderStartDeg, "overlayhud-arc-blue")
	orangeArc := buildArc(orangeShare, blueArc.EndDeg, "overlayhud-arc-orange")

	confidence := clamp01(vm.Confidence)
	confidenceBucket := renderConfidenceBucket(vm.ConfidenceBucket, confidence)
	volatility := applyVolatilityResponse(vm.Volatility)
	recentEvent := buildRecentEventModel(vm)
	displayState := renderDisplayState(vm)
	seamAngle := calculateSeamAngle(blueShare, orangeShare)

	return RenderModel{
		ViewBox: renderViewBox,
		CenterX: renderCenterX,
		CenterY: renderCenterY,
		Groups:  defaultRenderGroups(),

		BlueArc:   blueArc,
		OrangeArc: orangeArc,

		Confidence: ConfidenceModel{
			Value:     confidence,
			Bucket:    confidenceBucket,
			Intensity: confidence,
			ClassName: confidenceClass(confidenceBucket, confidence),
		},
		Segments:      buildWheelSegments(blueShare, orangeShare, volatility, displayState),
		Ticks:         buildWheelTicks(blueShare, orangeShare),
		RecentEvent:   recentEvent,
		ContestedLine: buildContestedLineModel(seamAngle, displayState, volatility, recentEvent.Energy, vm.HasData, vm.MatchActive, vm.IsStale),
		Diagnostics: SignalDiagnosticsModel{
			BlueShare:           blueShare,
			OrangeShare:         orangeShare,
			BluePressureShare:   bluePressureShare,
			OrangePressureShare: orangePressureShare,
			BlueControlShare:    vm.BlueControlShare,
			OrangeControlShare:  vm.OrangeControlShare,
			ConfidenceBucket:    confidenceBucket,
			Volatility:          volatility,
		},

		Center: CenterModel{
			PrimaryText: renderTimerPlaceholder,
			StateLabel:  vm.StateLabel,
		},

		DisplayState: displayState,
		StateClasses: stateClasses(vm, displayState),
		StyleVars:    renderStyleVars(visualBluePressure, visualOrangePressure, confidence, volatility, displayState),
		SeamAngleDeg: seamAngle,

		MatchActive: vm.MatchActive,
		HasData:     vm.HasData,
		IsStale:     vm.IsStale,
	}
}

func defaultRenderGroups() RenderGroups {
	return RenderGroups{
		Root:               "momentum-wheel-root",
		Background:         "background",
		OuterAura:          "outer-aura",
		OuterStreaks:       "outer-energy-streaks",
		OuterSparks:        "outer-sparks",
		Frame:              "outer-mechanical-frame",
		SegmentUnderlay:    "segment-ring-underlay",
		SegmentActive:      "segment-ring-active",
		SegmentBevels:      "segment-ring-bevels",
		InnerTickRing:      "inner-tick-ring",
		CenterCore:         "center-disc",
		CenterColorWashes:  "center-color-washes",
		CenterTexture:      "center-texture",
		CenterRim:          "center-rim",
		ContestedFrontLine: "contested-front-line",
		TextLayer:          "text-layer",
		Badge:              "oof-badge",
		DebugOverlays:      "debug-overlays",
		MomentumArcs:       "hud-momentum-arcs",
		ConfidenceRing:     "hud-confidence-ring",
		VolatilityRing:     "hud-volatility-segments",
		Labels:             "hud-labels",
		StateOverlay:       "hud-status-overlay",
	}
}

func buildArc(share, startDeg float64, className string) ArcModel {
	share = clamp01(share)
	sweep := share * renderFullSweepDeg
	return ArcModel{
		Share:     share,
		StartDeg:  startDeg,
		EndDeg:    startDeg + sweep,
		SweepDeg:  sweep,
		ClassName: className,
	}
}

func buildWheelSegments(blueShare, orangeShare, volatility float64, displayState string) []WheelSegment {
	volatility = clamp01(volatility)
	bluePercent := blueShare * 100
	orangePercent := orangeShare * 100
	seamAngle := calculateSeamAngle(blueShare, orangeShare)
	seamBand := 7.5
	if displayState == displayStateVolatile {
		seamBand = 11.25
	}

	segments := make([]WheelSegment, renderWheelSegments)
	for i := range segments {
		angle := float64(i) * renderSegmentStepDeg
		owner := segmentOwner(angle, bluePercent, orangePercent)
		isSeam := circularDistance(angle, seamAngle) <= seamBand/2
		className := "mcw-segment"
		if owner == "blue" {
			className += " mcw-segment-blue"
		} else {
			className += " mcw-segment-orange"
		}
		if isSeam {
			className += " is-seam"
		}
		segments[i] = WheelSegment{
			Index:     i,
			Active:    true,
			Intensity: volatility,
			AngleDeg:  angle,
			Owner:     owner,
			IsSeam:    isSeam,
			ClassName: className,
		}
	}
	return segments
}

func buildWheelTicks(blueShare, orangeShare float64) []WheelTick {
	bluePercent := blueShare * 100
	orangePercent := orangeShare * 100
	ticks := make([]WheelTick, renderWheelTicks)
	for i := range ticks {
		angle := float64(i) * renderTickStepDeg
		owner := segmentOwner(angle, bluePercent, orangePercent)
		ticks[i] = WheelTick{
			Index:     i,
			AngleDeg:  angle,
			Owner:     owner,
			ClassName: "mcw-tick mcw-tick-" + owner,
		}
	}
	return ticks
}

func segmentOwner(angle, bluePercent, orangePercent float64) string {
	if bluePercent >= 99.999 {
		return "blue"
	}
	if orangePercent >= 99.999 {
		return "orange"
	}
	blueSpanDegrees := bluePercent * 3.6
	if angularDistanceClockwise(180, normalizeAngle(angle)) < blueSpanDegrees {
		return "blue"
	}
	return "orange"
}

func calculateSeamAngle(blueShare, orangeShare float64) float64 {
	bluePercent := blueShare * 100
	orangePercent := orangeShare * 100
	if bluePercent >= 99.999 || orangePercent >= 99.999 {
		return 180
	}
	return normalizeAngle(180 + bluePercent*3.6)
}

func angularDistanceClockwise(from, to float64) float64 {
	return normalizeAngle(to - from)
}

func circularDistance(a, b float64) float64 {
	diff := normalizeAngle(a - b)
	if diff > 180 {
		return 360 - diff
	}
	return diff
}

func normalizeAngle(angle float64) float64 {
	for angle < 0 {
		angle += 360
	}
	for angle >= 360 {
		angle -= 360
	}
	return angle
}

func confidenceClass(bucket string, confidence float64) string {
	if confidence <= 0 {
		return "overlayhud-confidence is-empty"
	}
	switch bucket {
	case confidenceBucketLow:
		return "overlayhud-confidence is-low"
	case confidenceBucketMedium:
		return "overlayhud-confidence is-medium"
	case confidenceBucketHigh:
		return "overlayhud-confidence is-high"
	case confidenceBucketMax:
		return "overlayhud-confidence is-max"
	default:
		return "overlayhud-confidence is-low"
	}
}

func renderConfidenceBucket(bucket string, confidence float64) string {
	switch bucket {
	case confidenceBucketLow, confidenceBucketMedium, confidenceBucketHigh, confidenceBucketMax:
		return bucket
	default:
		return confidenceBucket(confidence)
	}
}

func buildRecentEventModel(vm ViewModel) RecentEventModel {
	energy := applyRecentEventResponse(vm.RecentEventEnergy)
	team := safeRecentEventToken(vm.RecentEventTeam)
	eventType := safeRecentEventToken(vm.RecentEventType)
	className := "mcw-recent-event"
	if energy > 0 && team != "" && eventType != "" {
		className += " has-recent-event is-recent-team-" + team + " is-recent-event-" + eventType
	}
	return RecentEventModel{
		Energy:    energy,
		Team:      team,
		Type:      eventType,
		ClassName: className,
	}
}

func buildContestedLineModel(angle float64, displayState string, volatility, recentEventEnergy float64, hasData, matchActive, stale bool) ContestedLineModel {
	// This exists as a display contract: the seam/front line follows the
	// ownership split, while intensity remains visual-only and runtime-only.
	band := 7.5
	if displayState == displayStateVolatile {
		band = 11.25
	}
	active := hasData && matchActive && !stale
	intensity := 0.34 + clamp01(volatility)*0.56 + clamp01(recentEventEnergy)*0.22
	if !active {
		intensity *= 0.45
	}
	return ContestedLineModel{
		Active:    active,
		AngleDeg:  normalizeAngle(angle),
		BandDeg:   band,
		Intensity: clamp01(intensity),
		ClassName: contestedLineClassName(displayState, active),
	}
}

func contestedLineClassName(displayState string, active bool) string {
	className := "mcw-contested-front-line is-contested-line-" + displayState
	if active {
		className += " is-contested-line-active"
	} else {
		className += " is-contested-line-inactive"
	}
	return className
}

func safeRecentEventToken(value string) string {
	switch value {
	case "blue", "orange", "ball_hit", "shot", "save", "goal", "assist", "demo":
		return value
	default:
		return ""
	}
}

func renderStyleVars(blueShare, orangeShare, confidence, volatility float64, displayState string) string {
	bluePressure := clamp01(blueShare)
	orangePressure := clamp01(orangeShare)
	volatility = clamp01(volatility)
	confidence = clamp01(confidence)

	blueWash, orangeWash, purpleWash := 0.12, 0.12, 0.04
	blueAura, orangeAura, contestAura := 0.11, 0.11, 0.14
	stateIntensity := 0.35

	switch displayState {
	case displayStateBluePressure:
		blueWash, orangeWash, purpleWash = 0.18, 0.06, 0.06
		blueAura, orangeAura, contestAura = 0.30, 0.05, 0.26
		stateIntensity = displayVisualStatePressure
	case displayStateOrangePressure:
		blueWash, orangeWash, purpleWash = 0.06, 0.18, 0.06
		blueAura, orangeAura, contestAura = 0.05, 0.30, 0.26
		stateIntensity = displayVisualStatePressure
	case displayStateBlueControl:
		blueWash, orangeWash, purpleWash = 0.28, 0.04, 0.05
		blueAura, orangeAura, contestAura = 0.42, 0.03, 0.20
		stateIntensity = displayVisualStateControl
	case displayStateOrangeControl:
		blueWash, orangeWash, purpleWash = 0.04, 0.28, 0.05
		blueAura, orangeAura, contestAura = 0.03, 0.42, 0.20
		stateIntensity = displayVisualStateControl
	case displayStateDominantBlue:
		blueWash, orangeWash, purpleWash = 0.40, 0.02, 0.04
		blueAura, orangeAura, contestAura = 0.58, 0.02, 0.16
		stateIntensity = displayVisualStateDominant
	case displayStateDominantOrange:
		blueWash, orangeWash, purpleWash = 0.02, 0.40, 0.04
		blueAura, orangeAura, contestAura = 0.02, 0.58, 0.16
		stateIntensity = displayVisualStateDominant
	case displayStateVolatile:
		blueWash, orangeWash, purpleWash = 0.12, 0.12, 0.24
		blueAura, orangeAura, contestAura = 0.16, 0.16, 0.64
		stateIntensity = displayVisualStateVolatile
	case displayStateNoData, displayStateStale, displayStateInactive:
		blueWash, orangeWash, purpleWash = 0.04, 0.04, 0.02
		blueAura, orangeAura, contestAura = 0.04, 0.04, 0.05
		stateIntensity = 0.25
	default:
		stateIntensity = displayVisualStateNeutral
	}

	return "--mcw-blue-pressure:" + styleNum(bluePressure) +
		";--mcw-orange-pressure:" + styleNum(orangePressure) +
		";--mcw-confidence:" + styleNum(confidence) +
		";--mcw-volatility:" + styleNum(volatility) +
		";--mcw-state-intensity:" + styleNum(stateIntensity) +
		";--mcw-center-blue-wash:" + styleNum(blueWash) +
		";--mcw-center-orange-wash:" + styleNum(orangeWash) +
		";--mcw-center-purple-wash:" + styleNum(purpleWash) +
		";--mcw-blue-aura-opacity:" + styleNum(blueAura) +
		";--mcw-orange-aura-opacity:" + styleNum(orangeAura) +
		";--mcw-contest-aura-opacity:" + styleNum(contestAura) +
		";--mcw-seam-intensity:" + styleNum(0.34+volatility*0.56) +
		";"
}

func applyPressureResponse(bluePressure, orangePressure float64) (float64, float64) {
	scale := 0.7 + displayResponsePressureSensitivity*0.3
	return clamp01(bluePressure * scale), clamp01(orangePressure * scale)
}

func applyVolatilityResponse(volatility float64) float64 {
	scale := 0.55 + displayResponseVolatilitySensitivity*0.45
	return clamp01(volatility * scale)
}

func applyRecentEventResponse(energy float64) float64 {
	eventScale := 0.35 + displayResponseEventReactiveness*0.65
	raw := clamp01(energy * eventScale)
	exponent := displayResponseDecaySpeed / max(0.25, displayResponseHoldDuration)
	return clamp01(math.Pow(raw, exponent))
}

func styleNum(value float64) string {
	if value < 0 {
		value = 0
	}
	return num(value)
}

func stateClasses(vm ViewModel, displayState string) []string {
	classes := []string{
		"momentum-control-wheel-svg",
		"overlayhud-render-model",
		"mcw-state-" + displayState,
		"is-state-" + displayState,
	}
	if vm.MatchActive {
		classes = append(classes, "is-active")
	} else {
		classes = append(classes, "is-inactive")
	}
	if vm.HasData {
		classes = append(classes, "has-data")
	} else {
		classes = append(classes, "has-no-data")
	}
	if vm.IsStale {
		classes = append(classes, "is-stale")
	}
	if vm.RecentEventEnergy > 0 && safeRecentEventToken(vm.RecentEventTeam) != "" && safeRecentEventToken(vm.RecentEventType) != "" {
		classes = append(classes,
			"has-recent-event",
			"recent-event-team-"+safeRecentEventToken(vm.RecentEventTeam),
			"recent-event-"+safeRecentEventToken(vm.RecentEventType),
		)
	}
	return classes
}

func renderDisplayState(vm ViewModel) string {
	if vm.DisplayState != "" {
		return vm.DisplayState
	}
	return displayStateFromLabel(vm.StateLabel)
}

func displayStateFromLabel(label string) string {
	switch label {
	case "BLUE PRESSURE":
		return displayStateBluePressure
	case "ORANGE PRESSURE":
		return displayStateOrangePressure
	case "BLUE CONTROL":
		return displayStateBlueControl
	case "ORANGE CONTROL":
		return displayStateOrangeControl
	case "CONTESTED", "VOLATILE":
		return displayStateVolatile
	case "STALE":
		return displayStateStale
	case "NO DATA":
		return displayStateNoData
	case "INACTIVE":
		return displayStateInactive
	default:
		return displayStateNeutral
	}
}
