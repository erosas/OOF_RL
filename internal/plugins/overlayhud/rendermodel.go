package overlayhud

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

// RenderModel contains SVG/display primitives for the future Overlay HUD renderer.
// It intentionally avoids SVG path strings, markup, runtime ownership, or rendering.
type RenderModel struct {
	ViewBox string
	CenterX float64
	CenterY float64

	Groups RenderGroups

	BlueArc   ArcModel
	OrangeArc ArcModel

	Confidence ConfidenceModel
	Segments   []WheelSegment
	Ticks      []WheelTick

	Center CenterModel

	DisplayState string
	StateClasses []string

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

type CenterModel struct {
	PrimaryText string
	StateLabel  string
}

func buildRenderModel(vm ViewModel) RenderModel {
	blueShare, orangeShare := shares(vm.BlueShare, vm.OrangeShare)
	blueArc := buildArc(blueShare, renderStartDeg, "overlayhud-arc-blue")
	orangeArc := buildArc(orangeShare, blueArc.EndDeg, "overlayhud-arc-orange")

	confidence := clamp01(vm.Confidence)
	volatility := clamp01(vm.Volatility)
	displayState := renderDisplayState(vm)

	return RenderModel{
		ViewBox: renderViewBox,
		CenterX: renderCenterX,
		CenterY: renderCenterY,
		Groups:  defaultRenderGroups(),

		BlueArc:   blueArc,
		OrangeArc: orangeArc,

		Confidence: ConfidenceModel{
			Value:     confidence,
			Intensity: confidence,
			ClassName: confidenceClass(confidence),
		},
		Segments: buildWheelSegments(blueShare, orangeShare, volatility, displayState),
		Ticks:    buildWheelTicks(blueShare, orangeShare),

		Center: CenterModel{
			PrimaryText: renderTimerPlaceholder,
			StateLabel:  vm.StateLabel,
		},

		DisplayState: displayState,
		StateClasses: stateClasses(vm, displayState),

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

func confidenceClass(confidence float64) string {
	switch {
	case confidence <= 0:
		return "overlayhud-confidence is-empty"
	case confidence < 0.35:
		return "overlayhud-confidence is-low"
	case confidence < 0.67:
		return "overlayhud-confidence is-medium"
	default:
		return "overlayhud-confidence is-high"
	}
}

func stateClasses(vm ViewModel, displayState string) []string {
	classes := []string{
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
	case "VOLATILE":
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
