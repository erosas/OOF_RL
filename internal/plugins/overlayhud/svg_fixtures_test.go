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
	expectedActiveTicks    int
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
			expectedClasses:        []string{"is-inactive", "has-no-data", "is-stale"},
			expectedLabel:          "NO DATA",
			expectedConfidence:     "overlayhud-confidence is-empty",
			expectedActiveTicks:    0,
			expectedStaleIndicator: true,
		},
		{
			name: "active shifting",
			view: ViewModel{
				MatchActive: true,
				HasData:     true,
				IsStale:     false,
				BlueShare:   0.52,
				OrangeShare: 0.48,
				StateLabel:  "SHIFTING",
				Confidence:  0.42,
				Volatility:  0.38,
			},
			expectedClasses:     []string{"is-active", "has-data"},
			expectedLabel:       "SHIFTING",
			expectedConfidence:  "overlayhud-confidence is-medium",
			expectedActiveTicks: 10,
		},
		{
			name: "blue pressure",
			view: ViewModel{
				MatchActive: true,
				HasData:     true,
				IsStale:     false,
				BlueShare:   0.72,
				OrangeShare: 0.28,
				StateLabel:  "BLUE PRESSURE",
				Confidence:  0.76,
				Volatility:  0.24,
			},
			expectedClasses:     []string{"is-active", "has-data"},
			expectedLabel:       "BLUE PRESSURE",
			expectedConfidence:  "overlayhud-confidence is-high",
			expectedActiveTicks: 6,
		},
		{
			name: "orange pressure volatile",
			view: ViewModel{
				MatchActive: true,
				HasData:     true,
				IsStale:     false,
				BlueShare:   0.31,
				OrangeShare: 0.69,
				StateLabel:  "ORANGE PRESSURE",
				Confidence:  0.67,
				Volatility:  0.86,
			},
			expectedClasses:     []string{"is-active", "has-data"},
			expectedLabel:       "ORANGE PRESSURE",
			expectedConfidence:  "overlayhud-confidence is-high",
			expectedActiveTicks: 21,
		},
		{
			name: "stale last known",
			view: ViewModel{
				MatchActive: true,
				HasData:     true,
				IsStale:     true,
				BlueShare:   0.64,
				OrangeShare: 0.36,
				StateLabel:  "BLUE PRESSURE",
				Confidence:  0.51,
				Volatility:  0.17,
			},
			expectedClasses:        []string{"is-active", "has-data", "is-stale"},
			expectedLabel:          "BLUE PRESSURE",
			expectedConfidence:     "overlayhud-confidence is-medium",
			expectedActiveTicks:    5,
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
		`viewBox="0 0 320 320"`,
		`id="hud-root"`,
		`id="hud-momentum-blue"`,
		`id="hud-momentum-orange"`,
		`id="hud-volatility-segments"`,
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

	activeTicks := strings.Count(svg, `data-active="true"`)
	if activeTicks != fixture.expectedActiveTicks {
		t.Fatalf("fixture %q active ticks = %d, want %d", fixture.name, activeTicks, fixture.expectedActiveTicks)
	}

	hasStaleIndicator := strings.Contains(svg, `>STALE</text>`)
	if hasStaleIndicator != fixture.expectedStaleIndicator {
		t.Fatalf("fixture %q stale indicator = %v, want %v", fixture.name, hasStaleIndicator, fixture.expectedStaleIndicator)
	}
}
