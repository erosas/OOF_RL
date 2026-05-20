package overlayhud

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildRenderModelUsesSpecGeometry(t *testing.T) {
	model := buildRenderModel(ViewModel{})

	if model.ViewBox != "0 0 1024 1024" {
		t.Fatalf("ViewBox = %q, want 0 0 1024 1024", model.ViewBox)
	}
	if model.CenterX != 512 || model.CenterY != 512 {
		t.Fatalf("center = %f,%f, want 512,512", model.CenterX, model.CenterY)
	}
	if model.Groups.Root != "momentum-wheel-root" || model.Groups.SegmentActive != "segment-ring-active" || model.Groups.InnerTickRing != "inner-tick-ring" {
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
	for _, className := range []string{"is-inactive", "has-no-data", "mcw-state-no-data", "is-state-no-data"} {
		if !hasStateClass(model.StateClasses, className) {
			t.Fatalf("StateClasses = %v, want %s", model.StateClasses, className)
		}
	}
	if model.DisplayState != displayStateNoData {
		t.Fatalf("DisplayState = %q, want %q", model.DisplayState, displayStateNoData)
	}
	if model.Center.StateLabel != "NO DATA" {
		t.Fatalf("StateLabel = %q, want NO DATA", model.Center.StateLabel)
	}
}

func TestBuildRenderModelAddsMomentumControlWheelRootClassAndStyleVars(t *testing.T) {
	model := buildRenderModel(ViewModel{
		MatchActive:         true,
		HasData:             true,
		BlueShare:           0.72,
		OrangeShare:         0.28,
		BluePressureShare:   0.44,
		OrangePressureShare: 0.56,
		DisplayState:        displayStateBlueControl,
		StateLabel:          "BLUE CONTROL",
		Confidence:          0.76,
		ConfidenceBucket:    confidenceBucketHigh,
		Volatility:          0.24,
	})

	for _, className := range []string{"momentum-control-wheel-svg", "mcw-state-blue-control", "is-state-blue-control"} {
		if !hasStateClass(model.StateClasses, className) {
			t.Fatalf("StateClasses = %v, want %s", model.StateClasses, className)
		}
	}
	for _, want := range []string{
		"--mcw-blue-pressure:0.480",
		"--mcw-orange-pressure:0.610",
		"--mcw-confidence:0.760",
		"--mcw-volatility:0.272",
		"--mcw-state-intensity:1.750",
		"--mcw-center-blue-wash:0.280",
		"--mcw-blue-aura-opacity:0.420",
	} {
		if !strings.Contains(model.StyleVars, want) {
			t.Fatalf("StyleVars = %q, want %q", model.StyleVars, want)
		}
	}
	if !almostEqual(model.SeamAngleDeg, 79.2) {
		t.Fatalf("SeamAngleDeg = %f, want 79.2", model.SeamAngleDeg)
	}
}

func TestBuildRenderModelPreservesDominantStateClasses(t *testing.T) {
	model := buildRenderModel(ViewModel{
		MatchActive:      true,
		HasData:          true,
		BlueShare:        0.86,
		OrangeShare:      0.14,
		DisplayState:     displayStateDominantBlue,
		StateLabel:       "BLUE CONTROL",
		Confidence:       0.90,
		ConfidenceBucket: confidenceBucketMax,
	})

	if model.DisplayState != displayStateDominantBlue {
		t.Fatalf("DisplayState = %q, want %q", model.DisplayState, displayStateDominantBlue)
	}
	for _, className := range []string{"mcw-state-dominant-blue", "is-state-dominant-blue"} {
		if !hasStateClass(model.StateClasses, className) {
			t.Fatalf("StateClasses = %v, want %s", model.StateClasses, className)
		}
	}
	if model.Confidence.Bucket != confidenceBucketMax {
		t.Fatalf("Confidence bucket = %q, want %q", model.Confidence.Bucket, confidenceBucketMax)
	}
}

func TestBuildRenderModelMapsConfidenceMaxBucket(t *testing.T) {
	model := buildRenderModel(ViewModel{
		Confidence:       0.95,
		ConfidenceBucket: confidenceBucketMax,
	})

	if model.Confidence.Bucket != confidenceBucketMax {
		t.Fatalf("Confidence bucket = %q, want %q", model.Confidence.Bucket, confidenceBucketMax)
	}
	if !strings.Contains(model.Confidence.ClassName, "is-max") {
		t.Fatalf("confidence class = %q, want is-max", model.Confidence.ClassName)
	}
}

func TestBuildRenderModelPreservesRecentEventContract(t *testing.T) {
	model := buildRenderModel(ViewModel{
		MatchActive:       true,
		HasData:           true,
		RecentEventEnergy: 0.64,
		RecentEventTeam:   "blue",
		RecentEventType:   "goal",
	})

	if !almostEqual(model.RecentEvent.Energy, 0.9728) {
		t.Fatalf("RecentEvent energy = %f, want 0.9728", model.RecentEvent.Energy)
	}
	if model.RecentEvent.Team != "blue" || model.RecentEvent.Type != "goal" {
		t.Fatalf("RecentEvent team/type = %q/%q, want blue/goal", model.RecentEvent.Team, model.RecentEvent.Type)
	}
	if model.RecentEvent.ClassName != "mcw-recent-event has-recent-event is-recent-team-blue is-recent-event-goal" {
		t.Fatalf("RecentEvent class = %q", model.RecentEvent.ClassName)
	}
	for _, className := range []string{"has-recent-event", "recent-event-team-blue", "recent-event-goal"} {
		if !hasStateClass(model.StateClasses, className) {
			t.Fatalf("StateClasses = %v, want %s", model.StateClasses, className)
		}
	}
}

func TestBuildRenderModelAddsContestedLineContract(t *testing.T) {
	model := buildRenderModel(ViewModel{
		MatchActive:       true,
		HasData:           true,
		BlueShare:         0.70,
		OrangeShare:       0.30,
		DisplayState:      displayStateBlueControl,
		StateLabel:        "BLUE CONTROL",
		Volatility:        0.40,
		RecentEventEnergy: 0.50,
		RecentEventTeam:   "blue",
		RecentEventType:   "shot",
	})

	if !almostEqual(model.ContestedLine.AngleDeg, model.SeamAngleDeg) || !almostEqual(model.ContestedLine.AngleDeg, 72) {
		t.Fatalf("ContestedLine angle = %f seam = %f, want 72", model.ContestedLine.AngleDeg, model.SeamAngleDeg)
	}
	if !model.ContestedLine.Active {
		t.Fatal("expected contested line active for live data")
	}
	if !almostEqual(model.ContestedLine.BandDeg, 7.5) {
		t.Fatalf("ContestedLine BandDeg = %f, want 7.5", model.ContestedLine.BandDeg)
	}
	if !almostEqual(model.ContestedLine.Intensity, 0.76144) {
		t.Fatalf("ContestedLine Intensity = %f, want 0.76144", model.ContestedLine.Intensity)
	}
	for _, want := range []string{"mcw-contested-front-line", "is-contested-line-active", "is-contested-line-blue-control"} {
		if !strings.Contains(model.ContestedLine.ClassName, want) {
			t.Fatalf("ContestedLine class = %q, want %q", model.ContestedLine.ClassName, want)
		}
	}
}

func TestBuildRenderModelWidensContestedLineForVolatileState(t *testing.T) {
	model := buildRenderModel(ViewModel{
		MatchActive:  true,
		HasData:      true,
		BlueShare:    0.50,
		OrangeShare:  0.50,
		DisplayState: displayStateVolatile,
		StateLabel:   "CONTESTED",
		Volatility:   0.90,
	})

	if !almostEqual(model.ContestedLine.BandDeg, 11.25) {
		t.Fatalf("ContestedLine BandDeg = %f, want 11.25", model.ContestedLine.BandDeg)
	}
	if !strings.Contains(model.ContestedLine.ClassName, "is-contested-line-volatile") {
		t.Fatalf("ContestedLine class = %q, want volatile class", model.ContestedLine.ClassName)
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

func TestBuildRenderModelAddsMomentumControlWheelStateClasses(t *testing.T) {
	tests := []struct {
		name        string
		view        ViewModel
		wantState   string
		wantClasses []string
	}{
		{
			name:      "explicit blue control",
			view:      ViewModel{DisplayState: displayStateBlueControl, StateLabel: "BLUE CONTROL"},
			wantState: displayStateBlueControl,
			wantClasses: []string{
				"mcw-state-blue-control",
				"is-state-blue-control",
			},
		},
		{
			name:      "label fallback volatile",
			view:      ViewModel{StateLabel: "CONTESTED"},
			wantState: displayStateVolatile,
			wantClasses: []string{
				"mcw-state-volatile",
				"is-state-volatile",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := buildRenderModel(tt.view)
			if model.DisplayState != tt.wantState {
				t.Fatalf("DisplayState = %q, want %q", model.DisplayState, tt.wantState)
			}
			for _, className := range tt.wantClasses {
				if !hasStateClass(model.StateClasses, className) {
					t.Fatalf("StateClasses = %v, want %s", model.StateClasses, className)
				}
			}
		})
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
	if !strings.Contains(high.Confidence.ClassName, "is-max") {
		t.Fatalf("high confidence class = %q, want is-max", high.Confidence.ClassName)
	}
}

func TestBuildRenderModelCreatesMomentumControlWheelSegments(t *testing.T) {
	model := buildRenderModel(ViewModel{Volatility: 0.50})

	if len(model.Segments) != 96 {
		t.Fatalf("segments = %d, want 96", len(model.Segments))
	}
	for i, segment := range model.Segments {
		if segment.Index != i {
			t.Fatalf("segment index = %d, want %d", segment.Index, i)
		}
		if !almostEqual(segment.AngleDeg, float64(i)*3.75) {
			t.Fatalf("segment %d angle = %f, want %f", i, segment.AngleDeg, float64(i)*3.75)
		}
		if segment.Owner != "blue" && segment.Owner != "orange" {
			t.Fatalf("segment %d owner = %q, want blue/orange", i, segment.Owner)
		}
	}
}

func TestBuildRenderModelCreatesMomentumControlWheelTicks(t *testing.T) {
	model := buildRenderModel(ViewModel{})

	if len(model.Ticks) != 120 {
		t.Fatalf("ticks = %d, want 120", len(model.Ticks))
	}
	for i, tick := range model.Ticks {
		if tick.Index != i {
			t.Fatalf("tick index = %d, want %d", tick.Index, i)
		}
		if !almostEqual(tick.AngleDeg, float64(i)*3) {
			t.Fatalf("tick %d angle = %f, want %f", i, tick.AngleDeg, float64(i)*3)
		}
	}
}

func TestBuildRenderModelAssignsSegmentOwnersFromOldWheelOrigin(t *testing.T) {
	model := buildRenderModel(ViewModel{BlueShare: 0.50, OrangeShare: 0.50})

	if model.Segments[0].Owner != "orange" {
		t.Fatalf("segment 0 owner = %q, want orange for old 180deg blue origin", model.Segments[0].Owner)
	}
	if model.Segments[48].Owner != "blue" {
		t.Fatalf("segment 48 owner = %q, want blue at 180deg origin", model.Segments[48].Owner)
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
