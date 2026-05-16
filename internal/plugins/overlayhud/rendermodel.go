package overlayhud

const (
	renderViewBox              = "0 0 320 320"
	renderCenterX              = 160
	renderCenterY              = 160
	renderStartDeg             = -90
	renderFullSweepDeg         = 360
	renderVolatilitySegments   = 24
	renderVolatilitySegmentGap = 4
	renderTimerPlaceholder     = "--:--"
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
	Volatility []VolatilitySegment

	Center CenterModel

	StateClasses []string

	MatchActive bool
	HasData     bool
	IsStale     bool
}

type RenderGroups struct {
	Root           string
	Background     string
	MomentumArcs   string
	ConfidenceRing string
	VolatilityRing string
	CenterCore     string
	Labels         string
	StateOverlay   string
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

type VolatilitySegment struct {
	Index     int
	Active    bool
	Intensity float64
	StartDeg  float64
	EndDeg    float64
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
		Volatility: buildVolatilitySegments(volatility),

		Center: CenterModel{
			PrimaryText: renderTimerPlaceholder,
			StateLabel:  vm.StateLabel,
		},

		StateClasses: stateClasses(vm),

		MatchActive: vm.MatchActive,
		HasData:     vm.HasData,
		IsStale:     vm.IsStale,
	}
}

func defaultRenderGroups() RenderGroups {
	return RenderGroups{
		Root:           "hud-root",
		Background:     "hud-background",
		MomentumArcs:   "hud-momentum-arcs",
		ConfidenceRing: "hud-confidence-ring",
		VolatilityRing: "hud-volatility-segments",
		CenterCore:     "hud-center-panel",
		Labels:         "hud-labels",
		StateOverlay:   "hud-status-overlay",
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

func buildVolatilitySegments(volatility float64) []VolatilitySegment {
	volatility = clamp01(volatility)
	activeCount := ceilSegments(volatility * renderVolatilitySegments)
	segments := make([]VolatilitySegment, renderVolatilitySegments)
	step := float64(renderFullSweepDeg) / float64(renderVolatilitySegments)

	for i := range segments {
		active := i < activeCount
		className := "overlayhud-volatility-segment"
		if active {
			className += " is-active"
		}
		start := renderStartDeg + float64(i)*step
		segments[i] = VolatilitySegment{
			Index:     i,
			Active:    active,
			Intensity: segmentIntensity(active, volatility),
			StartDeg:  start,
			EndDeg:    start + step - renderVolatilitySegmentGap,
			ClassName: className,
		}
	}

	return segments
}

func ceilSegments(v float64) int {
	count := int(v)
	if float64(count) < v {
		count++
	}
	if count < 0 {
		return 0
	}
	if count > renderVolatilitySegments {
		return renderVolatilitySegments
	}
	return count
}

func segmentIntensity(active bool, volatility float64) float64 {
	if !active {
		return 0
	}
	return volatility
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

func stateClasses(vm ViewModel) []string {
	classes := []string{"overlayhud-render-model"}
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
