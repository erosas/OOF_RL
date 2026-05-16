package overlayhud

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildRenderModelUsesSpecGeometry(t *testing.T) {
	model := buildRenderModel(ViewModel{})

	if model.ViewBox != "0 0 320 320" {
		t.Fatalf("ViewBox = %q, want 0 0 320 320", model.ViewBox)
	}
	if model.CenterX != 160 || model.CenterY != 160 {
		t.Fatalf("center = %f,%f, want 160,160", model.CenterX, model.CenterY)
	}
	if model.Groups.Root != "hud-root" || model.Groups.StateOverlay != "hud-status-overlay" {
		t.Fatalf("unexpected groups: %+v", model.Groups)
	}
}

func TestBuildRenderModelMapsSharesToArcSweeps(t *testing.T) {
	model := buildRenderModel(ViewModel{
		MatchActive: true,
		HasData:     true,
		BlueShare:   0.75,
		OrangeShare: 0.25,
		StateLabel:  "BLUE PRESSURE",
	})

	if !almostEqual(model.BlueArc.StartDeg, -90) || !almostEqual(model.BlueArc.SweepDeg, 270) {
		t.Fatalf("blue arc = %+v, want start -90 sweep 270", model.BlueArc)
	}
	if !almostEqual(model.BlueArc.EndDeg, 180) {
		t.Fatalf("blue end = %f, want 180", model.BlueArc.EndDeg)
	}
	if !almostEqual(model.OrangeArc.StartDeg, model.BlueArc.EndDeg) || !almostEqual(model.OrangeArc.SweepDeg, 90) {
		t.Fatalf("orange arc = %+v, want start at blue end and sweep 90", model.OrangeArc)
	}
}

func TestBuildRenderModelUsesNeutralNoDataState(t *testing.T) {
	model := buildRenderModel(ViewModel{
		StateLabel: "NO DATA",
	})

	if !almostEqual(model.BlueArc.Share, 0.5) || !almostEqual(model.OrangeArc.Share, 0.5) {
		t.Fatalf("shares = %f/%f, want 0.5/0.5", model.BlueArc.Share, model.OrangeArc.Share)
	}
	for _, className := range []string{"is-inactive", "has-no-data"} {
		if !hasStateClass(model.StateClasses, className) {
			t.Fatalf("StateClasses = %v, want %s", model.StateClasses, className)
		}
	}
	if model.Center.StateLabel != "NO DATA" {
		t.Fatalf("StateLabel = %q, want NO DATA", model.Center.StateLabel)
	}
}

func TestBuildRenderModelAddsStaleAndInactiveClasses(t *testing.T) {
	model := buildRenderModel(ViewModel{
		HasData:    true,
		IsStale:    true,
		StateLabel: "SHIFTING",
	})

	for _, className := range []string{"is-inactive", "has-data", "is-stale"} {
		if !hasStateClass(model.StateClasses, className) {
			t.Fatalf("StateClasses = %v, want %s", model.StateClasses, className)
		}
	}
	if model.MatchActive || !model.HasData || !model.IsStale {
		t.Fatalf("state flags = active:%v hasData:%v stale:%v", model.MatchActive, model.HasData, model.IsStale)
	}
}

func TestBuildRenderModelClampsConfidence(t *testing.T) {
	low := buildRenderModel(ViewModel{Confidence: -0.5})
	high := buildRenderModel(ViewModel{Confidence: 1.5})

	if low.Confidence.Value != 0 || low.Confidence.Intensity != 0 {
		t.Fatalf("low confidence = %+v, want zero", low.Confidence)
	}
	if high.Confidence.Value != 1 || high.Confidence.Intensity != 1 {
		t.Fatalf("high confidence = %+v, want one", high.Confidence)
	}
	if !strings.Contains(high.Confidence.ClassName, "is-high") {
		t.Fatalf("high confidence class = %q, want is-high", high.Confidence.ClassName)
	}
}

func TestBuildRenderModelCreatesVolatilitySegments(t *testing.T) {
	model := buildRenderModel(ViewModel{Volatility: 0.50})

	if len(model.Volatility) != 24 {
		t.Fatalf("volatility segments = %d, want 24", len(model.Volatility))
	}

	active := 0
	for _, segment := range model.Volatility {
		if segment.Active {
			active++
			if segment.Intensity != 0.50 {
				t.Fatalf("active segment intensity = %f, want 0.50", segment.Intensity)
			}
		}
		if !almostEqual(segment.EndDeg-segment.StartDeg, 11) {
			t.Fatalf("segment %d span = %f, want 11", segment.Index, segment.EndDeg-segment.StartDeg)
		}
	}
	if active != 12 {
		t.Fatalf("active segments = %d, want 12", active)
	}
}

func TestBuildRenderModelScalesVolatilityActivation(t *testing.T) {
	model := buildRenderModel(ViewModel{Volatility: 0.01})

	active := 0
	for _, segment := range model.Volatility {
		if segment.Active {
			active++
		}
	}
	if active != 1 {
		t.Fatalf("active segments = %d, want 1 for non-zero volatility", active)
	}
}

func TestBuildRenderModelUsesTimerPlaceholder(t *testing.T) {
	model := buildRenderModel(ViewModel{StateLabel: "SHIFTING"})

	if model.Center.PrimaryText != "--:--" {
		t.Fatalf("PrimaryText = %q, want --:--", model.Center.PrimaryText)
	}
}

func TestBuildRenderModelDoesNotExposeSVGStrings(t *testing.T) {
	modelType := reflect.TypeOf(RenderModel{})
	disallowedNames := []string{"Path", "PathData", "Markup", "SVG", "HTML"}

	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		for _, disallowed := range disallowedNames {
			if strings.Contains(field.Name, disallowed) {
				t.Fatalf("RenderModel field %s suggests SVG string rendering", field.Name)
			}
		}
	}
}

func hasStateClass(classes []string, className string) bool {
	for _, existing := range classes {
		if existing == className {
			return true
		}
	}
	return false
}
