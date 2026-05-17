package overlayhud

import (
	"strings"
	"testing"
)

type svgFixture struct {
	name                   string
	view                   ViewModel
	expectedClasses        []string
	expectedLabel          string
	expectedConfidence     string
	expectedBlueSegments   int
	expectedStaleIndicator bool
}

func TestSVGFixtureStatesRenderExpectedOutput(t *testing.T) {
	fixtures := []svgFixture{
		{
			name: "neutral empty",
			view: ViewModel{
				MatchActive: false,
				HasData:     false,
				IsStale:     true,
				BlueShare:   0.5,
				OrangeShare: 0.5,
				StateLabel:  "NO DATA",
				Confidence:  0,
				Volatility:  0,
			},
			expectedClasses:        []string{"is-inactive", "has-no-data", "is-stale", "mcw-state-no-data"},
			expectedLabel:          "NO DATA",
			expectedConfidence:     "overlayhud-confidence is-empty",
			expectedBlueSegments:   48,
			expectedStaleIndicator: true,
		},
		{
			name: "active neutral",
			view: ViewModel{
				MatchActive:  true,
				HasData:      true,
				IsStale:      false,
				BlueShare:    0.52,
				OrangeShare:  0.48,
				DisplayState: displayStateNeutral,
				StateLabel:   "NEUTRAL",
				Confidence:   0.42,
				Volatility:   0.38,
			},
			expectedClasses:      []string{"is-active", "has-data", "mcw-state-neutral"},
			expectedLabel:        "NEUTRAL",
			expectedConfidence:   "overlayhud-confidence is-medium",
			expectedBlueSegments: 50,
		},
		{
			name: "blue pressure",
			view: ViewModel{
				MatchActive:  true,
				HasData:      true,
				IsStale:      false,
				BlueShare:    0.72,
				OrangeShare:  0.28,
				DisplayState: displayStateBlueControl,
				StateLabel:   "BLUE CONTROL",
				Confidence:   0.76,
				Volatility:   0.24,
			},
			expectedClasses:      []string{"is-active", "has-data", "mcw-state-blue-control"},
			expectedLabel:        "BLUE CONTROL",
			expectedConfidence:   "overlayhud-confidence is-high",
			expectedBlueSegments: 70,
		},
		{
			name: "volatile contested",
			view: ViewModel{
				MatchActive:  true,
				HasData:      true,
				IsStale:      false,
				BlueShare:    0.31,
				OrangeShare:  0.69,
				DisplayState: displayStateVolatile,
				StateLabel:   "VOLATILE",
				Confidence:   0.67,
				Volatility:   0.86,
			},
			expectedClasses:      []string{"is-active", "has-data", "mcw-state-volatile"},
			expectedLabel:        "VOLATILE",
			expectedConfidence:   "overlayhud-confidence is-high",
			expectedBlueSegments: 30,
		},
		{
			name: "stale last known",
			view: ViewModel{
				MatchActive:  true,
				HasData:      true,
				IsStale:      true,
				BlueShare:    0.64,
				OrangeShare:  0.36,
				DisplayState: displayStateStale,
				StateLabel:   "STALE",
				Confidence:   0.51,
				Volatility:   0.17,
			},
			expectedClasses:        []string{"is-active", "has-data", "is-stale", "mcw-state-stale"},
			expectedLabel:          "STALE",
			expectedConfidence:     "overlayhud-confidence is-medium",
			expectedBlueSegments:   62,
			expectedStaleIndicator: true,
		},
	}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			model := buildRenderModel(fixture.view)
			svg := RenderSVG(model)

			assertFixtureSVG(t, svg, fixture)

			if second := RenderSVG(model); svg != second {
				t.Fatal("fixture SVG output should be deterministic")
			}
		})
	}
}

func assertFixtureSVG(t *testing.T, svg string, fixture svgFixture) {
	t.Helper()

	for _, want := range []string{
		`viewBox="0 0 1024 1024"`,
		`id="momentum-wheel-root"`,
		`id="segment-ring-underlay"`,
		`id="segment-ring-blue-active"`,
		`id="segment-ring-orange-active"`,
		`id="inner-tick-ring"`,
		`>--:--</text>`,
		`>` + fixture.expectedLabel + `</text>`,
		`class="` + fixture.expectedConfidence + `"`,
	} {
		if !strings.Contains(svg, want) {
			t.Fatalf("fixture %q missing %q: %s", fixture.name, want, svg)
		}
	}

	for _, className := range fixture.expectedClasses {
		if !strings.Contains(svg, className) {
			t.Fatalf("fixture %q missing state class %q: %s", fixture.name, className, svg)
		}
	}

	segments := strings.Count(svg, `class="mcw-segment mcw-segment-inactive"`)
	if segments != 96 {
		t.Fatalf("fixture %q inactive segments = %d, want 96", fixture.name, segments)
	}
	ticks := strings.Count(svg, `class="mcw-tick `)
	if ticks != 120 {
		t.Fatalf("fixture %q ticks = %d, want 120", fixture.name, ticks)
	}
	blueSegments := strings.Count(svg, `class="mcw-segment mcw-segment-blue`)
	if blueSegments != fixture.expectedBlueSegments {
		t.Fatalf("fixture %q blue segments = %d, want %d", fixture.name, blueSegments, fixture.expectedBlueSegments)
	}

	hasStaleIndicator := strings.Contains(svg, `>STALE</text>`)
	if hasStaleIndicator != fixture.expectedStaleIndicator {
		t.Fatalf("fixture %q stale indicator = %v, want %v", fixture.name, hasStaleIndicator, fixture.expectedStaleIndicator)
	}
}
