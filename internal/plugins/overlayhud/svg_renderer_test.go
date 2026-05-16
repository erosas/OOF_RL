package overlayhud

import (
	"reflect"
	"strings"
	"testing"
)

func TestRenderSVGIncludesFixedViewBox(t *testing.T) {
	svg := RenderSVG(testRenderModel())

	if !strings.Contains(svg, `viewBox="0 0 320 320"`) {
		t.Fatalf("svg missing fixed viewBox: %s", svg)
	}
}

func TestRenderSVGIncludesExpectedGroupsAndClasses(t *testing.T) {
	svg := RenderSVG(testRenderModel())

	for _, want := range []string{
		`id="hud-root"`,
		`id="hud-background"`,
		`id="hud-confidence-track"`,
		`id="hud-momentum-arcs"`,
		`id="hud-momentum-track"`,
		`id="hud-momentum-blue"`,
		`id="hud-momentum-orange"`,
		`id="hud-confidence-ring"`,
		`id="hud-volatility-track"`,
		`id="hud-volatility-segments"`,
		`id="hud-center-panel"`,
		`id="hud-timer-text"`,
		`id="hud-state-label"`,
		`id="hud-confidence-label"`,
		`id="hud-labels"`,
		`id="hud-status-overlay"`,
		`class="overlayhud-render-model is-active has-data is-stale"`,
		`class="overlayhud-arc-blue"`,
		`class="overlayhud-arc-orange"`,
	} {
		if !strings.Contains(svg, want) {
			t.Fatalf("svg missing %q: %s", want, svg)
		}
	}
}

func TestRenderSVGIncludesInactiveNoDataClasses(t *testing.T) {
	model := buildRenderModel(ViewModel{StateLabel: "NO DATA"})

	svg := RenderSVG(model)

	if !strings.Contains(svg, `class="overlayhud-render-model is-inactive has-no-data"`) {
		t.Fatalf("svg missing inactive/no-data classes: %s", svg)
	}
}

func TestRenderSVGIncludesTwentyFourVolatilitySegments(t *testing.T) {
	svg := RenderSVG(testRenderModel())

	if got := strings.Count(svg, `data-segment="`); got != 24 {
		t.Fatalf("volatility segment count = %d, want 24", got)
	}
	if !strings.Contains(svg, `data-segment="0"`) || !strings.Contains(svg, `data-segment="23"`) {
		t.Fatalf("svg missing first or last volatility segment: %s", svg)
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

func TestRenderSVGFullBlueArcIsNonDegenerate(t *testing.T) {
	model := testRenderModel()
	model.BlueArc = ArcModel{
		Share:     1,
		StartDeg:  -90,
		EndDeg:    270,
		SweepDeg:  360,
		ClassName: "overlayhud-arc-blue",
	}
	model.OrangeArc = ArcModel{
		Share:     0,
		StartDeg:  270,
		EndDeg:    270,
		SweepDeg:  0,
		ClassName: "overlayhud-arc-orange",
	}

	svg := RenderSVG(model)

	assertFullCirclePath(t, svg, `class="overlayhud-arc-blue"`)
}

func TestRenderSVGFullOrangeArcIsNonDegenerate(t *testing.T) {
	model := testRenderModel()
	model.BlueArc = ArcModel{
		Share:     0,
		StartDeg:  -90,
		EndDeg:    -90,
		SweepDeg:  0,
		ClassName: "overlayhud-arc-blue",
	}
	model.OrangeArc = ArcModel{
		Share:     1,
		StartDeg:  -90,
		EndDeg:    270,
		SweepDeg:  360,
		ClassName: "overlayhud-arc-orange",
	}

	svg := RenderSVG(model)

	assertFullCirclePath(t, svg, `class="overlayhud-arc-orange"`)
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
		MatchActive: true,
		HasData:     true,
		IsStale:     true,
		BlueShare:   0.70,
		OrangeShare: 0.30,
		StateLabel:  "BLUE PRESSURE",
		Confidence:  0.75,
		Volatility:  0.50,
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
