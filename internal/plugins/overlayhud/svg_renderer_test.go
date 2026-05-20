package overlayhud

import (
	"reflect"
	"strings"
	"testing"
)

func TestRenderSVGIncludesFixedViewBox(t *testing.T) {
	svg := RenderSVG(testRenderModel())

	if !strings.Contains(svg, `viewBox="0 0 1024 1024"`) {
		t.Fatalf("svg missing fixed viewBox: %s", svg)
	}
}

func TestRenderSVGIncludesExpectedGroupsAndClasses(t *testing.T) {
	svg := RenderSVG(testRenderModel())

	for _, want := range []string{
		`id="momentum-wheel-root"`,
		`id="background"`,
		`id="outer-aura"`,
		`id="outer-energy-streaks"`,
		`id="outer-sparks"`,
		`id="outer-mechanical-frame"`,
		`id="segment-ring-underlay"`,
		`id="segment-ring-active"`,
		`id="segment-ring-blue-active"`,
		`id="segment-ring-orange-active"`,
		`id="segment-ring-neutral-caps"`,
		`id="segment-ring-bevels"`,
		`id="inner-tick-ring"`,
		`id="inner-tick-ring-muted"`,
		`id="center-disc"`,
		`id="center-color-washes"`,
		`id="center-texture"`,
		`id="center-rim"`,
		`id="contested-front-line"`,
		`id="text-layer"`,
		`id="oof-badge"`,
		`id="debug-overlays"`,
		`id="hud-confidence-track"`,
		`id="hud-confidence-ring"`,
		`id="hud-timer-text"`,
		`id="hud-state-label"`,
		`id="hud-confidence-label"`,
		`id="hud-status-overlay"`,
		`id="momentum-control-wheel"`,
		`class="momentum-control-wheel-svg overlayhud-render-model mcw-state-blue-pressure is-state-blue-pressure is-active has-data is-stale"`,
		`data-state="blue-pressure"`,
		`data-seam-angle="72.000"`,
	} {
		if !strings.Contains(svg, want) {
			t.Fatalf("svg missing %q: %s", want, svg)
		}
	}
}

func TestRenderSVGIncludesRecentEventDataAttributes(t *testing.T) {
	model := testRenderModel()
	model.RecentEvent = RecentEventModel{
		Energy:    0.75,
		Team:      "orange",
		Type:      "goal",
		ClassName: "mcw-recent-event has-recent-event is-recent-team-orange is-recent-event-goal",
	}
	model.StateClasses = append(model.StateClasses, "has-recent-event", "recent-event-team-orange", "recent-event-goal")

	svg := RenderSVG(model)

	for _, want := range []string{
		`data-recent-event-energy="0.750"`,
		`data-recent-event-team="orange"`,
		`data-recent-event-type="goal"`,
		`has-recent-event`,
	} {
		if !strings.Contains(svg, want) {
			t.Fatalf("svg missing %q: %s", want, svg)
		}
	}
}

func TestRenderSVGIncludesContestedLineContractAttributes(t *testing.T) {
	model := testRenderModel()
	model.ContestedLine = ContestedLineModel{
		Active:    true,
		AngleDeg:  72,
		BandDeg:   7.5,
		Intensity: 0.67,
		ClassName: "mcw-contested-front-line is-contested-line-active is-contested-line-blue-control",
	}

	svg := RenderSVG(model)

	for _, want := range []string{
		`data-contested-line-angle="72.000"`,
		`data-contested-line-band="7.500"`,
		`data-contested-line-intensity="0.670"`,
		`is-contested-line-active`,
		`is-contested-line-blue-control`,
	} {
		if !strings.Contains(svg, want) {
			t.Fatalf("svg missing %q: %s", want, svg)
		}
	}
}

func TestRenderSVGIncludesSignalDiagnosticAttributes(t *testing.T) {
	model := testRenderModel()
	model.Diagnostics = SignalDiagnosticsModel{
		BlueShare:           0.72,
		OrangeShare:         0.28,
		BluePressureShare:   0.44,
		OrangePressureShare: 0.56,
		BlueControlShare:    0.72,
		OrangeControlShare:  0.28,
		ConfidenceBucket:    confidenceBucketHigh,
		Volatility:          0.24,
	}

	svg := RenderSVG(model)

	for _, want := range []string{
		`data-blue-share="0.720"`,
		`data-orange-share="0.280"`,
		`data-blue-pressure-share="0.440"`,
		`data-orange-pressure-share="0.560"`,
		`data-blue-control-share="0.720"`,
		`data-orange-control-share="0.280"`,
		`data-confidence-bucket="high"`,
		`data-volatility="0.240"`,
	} {
		if !strings.Contains(svg, want) {
			t.Fatalf("svg missing diagnostic attr %q: %s", want, svg)
		}
	}
}

func TestRenderSVGIncludesInactiveNoDataClasses(t *testing.T) {
	model := buildRenderModel(ViewModel{StateLabel: "NO DATA"})

	svg := RenderSVG(model)

	if !strings.Contains(svg, `class="momentum-control-wheel-svg overlayhud-render-model mcw-state-no-data is-state-no-data is-inactive has-no-data"`) {
		t.Fatalf("svg missing inactive/no-data classes: %s", svg)
	}
}

func TestRenderSVGIncludesMomentumControlWheelResponseHooks(t *testing.T) {
	svg := RenderSVG(testRenderModel())

	for _, want := range []string{
		`<defs>`,
		`id="mcw-soft-blur"`,
		`id="outer-aura-blue"`,
		`id="outer-aura-orange"`,
		`id="outer-aura-purple-contest"`,
		`id="center-disc-blue-wash"`,
		`id="center-disc-orange-wash"`,
		`id="center-disc-purple-contest-wash"`,
		`id="contest-top-core"`,
		`id="contest-top-purple-glow"`,
		`--mcw-blue-pressure:0.763`,
		`--mcw-orange-pressure:0.327`,
		`--mcw-volatility:0.568`,
	} {
		if !strings.Contains(svg, want) {
			t.Fatalf("svg missing response hook %q: %s", want, svg)
		}
	}
}

func TestRenderSVGUsesMomentumControlWheelSeamOrientation(t *testing.T) {
	tests := []struct {
		name             string
		blueShare        float64
		orangeShare      float64
		wantSeamAngle    string
		wantContestPoint string
		wantTransform    string
	}{
		{
			name:             "neutral seam starts at 12 oclock",
			blueShare:        0.50,
			orangeShare:      0.50,
			wantSeamAngle:    `data-seam-angle="0.000"`,
			wantContestPoint: `<circle id="outer-aura-purple-contest" class="mcw-aura mcw-aura-contest" cx="512.000" cy="84.000" r="34"/>`,
			wantTransform:    `transform="rotate(0.000 512 512)"`,
		},
		{
			name:             "blue control seam follows clockwise split",
			blueShare:        0.70,
			orangeShare:      0.30,
			wantSeamAngle:    `data-seam-angle="72.000"`,
			wantContestPoint: `<circle id="outer-aura-purple-contest" class="mcw-aura mcw-aura-contest" cx="919.052" cy="379.741" r="34"/>`,
			wantTransform:    `transform="rotate(72.000 512 512)"`,
		},
		{
			name:             "orange control seam follows clockwise split",
			blueShare:        0.30,
			orangeShare:      0.70,
			wantSeamAngle:    `data-seam-angle="288.000"`,
			wantContestPoint: `<circle id="outer-aura-purple-contest" class="mcw-aura mcw-aura-contest" cx="104.948" cy="379.741" r="34"/>`,
			wantTransform:    `transform="rotate(288.000 512 512)"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := buildRenderModel(ViewModel{
				MatchActive:  true,
				HasData:      true,
				BlueShare:    tt.blueShare,
				OrangeShare:  tt.orangeShare,
				DisplayState: displayStateBlueControl,
				StateLabel:   "BLUE CONTROL",
				Confidence:   0.75,
				Volatility:   0.25,
			})

			svg := RenderSVG(model)
			for _, want := range []string{tt.wantSeamAngle, tt.wantContestPoint, tt.wantTransform} {
				if !strings.Contains(svg, want) {
					t.Fatalf("svg missing seam parity marker %q: %s", want, svg)
				}
			}
		})
	}
}

func TestRenderSVGIncludesNinetySixWheelSegments(t *testing.T) {
	svg := RenderSVG(testRenderModel())

	if got := strings.Count(svg, `class="mcw-segment mcw-segment-inactive"`); got != 96 {
		t.Fatalf("inactive segment count = %d, want 96", got)
	}
	if !strings.Contains(svg, `data-segment="0"`) || !strings.Contains(svg, `data-segment="95"`) {
		t.Fatalf("svg missing expected segment data attributes: %s", svg)
	}
}

func TestRenderSVGIncludesOneHundredTwentyTicks(t *testing.T) {
	svg := RenderSVG(testRenderModel())

	if got := strings.Count(svg, `class="mcw-tick `); got != 120 {
		t.Fatalf("tick count = %d, want 120", got)
	}
	if !strings.Contains(svg, `data-tick="0"`) || !strings.Contains(svg, `data-tick="119"`) {
		t.Fatalf("svg missing first or last tick: %s", svg)
	}
}

func TestRenderSVGIncludesCenterText(t *testing.T) {
	svg := RenderSVG(testRenderModel())

	if !strings.Contains(svg, `>--:--</text>`) {
		t.Fatalf("svg missing timer placeholder: %s", svg)
	}
	if !strings.Contains(svg, `>BLUE PRESSURE</text>`) {
		t.Fatalf("svg missing state label: %s", svg)
	}
}

func TestRenderSVGFullConfidenceRingIsNonDegenerate(t *testing.T) {
	model := testRenderModel()
	model.Confidence = ConfidenceModel{
		Value:     1,
		Intensity: 1,
		ClassName: "overlayhud-confidence is-high",
	}

	svg := RenderSVG(model)

	assertFullCirclePath(t, svg, `class="overlayhud-confidence is-high"`)
}

func TestRenderSVGEscapesUnsafeText(t *testing.T) {
	model := testRenderModel()
	model.Center.PrimaryText = `<timer & "value">`
	model.Center.StateLabel = `<SHIFTING & "unsafe">`

	svg := RenderSVG(model)

	if strings.Contains(svg, model.Center.PrimaryText) || strings.Contains(svg, model.Center.StateLabel) {
		t.Fatalf("svg contains unsafe raw text: %s", svg)
	}
	if !strings.Contains(svg, `&lt;timer &amp; &#34;value&#34;&gt;`) {
		t.Fatalf("svg missing escaped primary text: %s", svg)
	}
	if !strings.Contains(svg, `&lt;SHIFTING &amp; &#34;unsafe&#34;&gt;`) {
		t.Fatalf("svg missing escaped state label: %s", svg)
	}
}

func TestRenderSVGConsumesRenderModelOnly(t *testing.T) {
	renderType := reflect.TypeOf(RenderSVG)

	if renderType.NumIn() != 1 || renderType.In(0) != reflect.TypeOf(RenderModel{}) {
		t.Fatalf("RenderSVG input = %v, want exactly RenderModel", renderType)
	}
	if renderType.NumOut() != 1 || renderType.Out(0).Kind() != reflect.String {
		t.Fatalf("RenderSVG output = %v, want string", renderType)
	}
}

func TestRenderSVGDeterministicForSameRenderModel(t *testing.T) {
	model := testRenderModel()

	first := RenderSVG(model)
	second := RenderSVG(model)

	if first != second {
		t.Fatalf("RenderSVG output is not deterministic:\nfirst: %s\nsecond: %s", first, second)
	}
}

func testRenderModel() RenderModel {
	return buildRenderModel(ViewModel{
		MatchActive:  true,
		HasData:      true,
		IsStale:      true,
		BlueShare:    0.70,
		OrangeShare:  0.30,
		DisplayState: displayStateBluePressure,
		StateLabel:   "BLUE PRESSURE",
		Confidence:   0.75,
		Volatility:   0.50,
	})
}

func assertFullCirclePath(t *testing.T, svg, className string) {
	t.Helper()

	path := pathWithClass(svg, className)
	if path == "" {
		t.Fatalf("svg missing path with %s: %s", className, svg)
	}
	if strings.Count(path, " A ") != 2 {
		t.Fatalf("full circle path should use two arc commands, got %q", path)
	}
}

func pathWithClass(svg, className string) string {
	start := strings.Index(svg, `<path `+className)
	if start == -1 {
		return ""
	}
	end := strings.Index(svg[start:], `/>`)
	if end == -1 {
		return ""
	}
	return svg[start : start+end+2]
}
