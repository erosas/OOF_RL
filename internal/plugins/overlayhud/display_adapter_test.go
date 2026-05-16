package overlayhud

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"OOF_RL/internal/momentum"
	"OOF_RL/internal/oofevents"
)

func TestDisplayAdapterNilProvider(t *testing.T) {
	adapter := NewDisplayAdapter(nil)

	if svg, ok := adapter.RenderSVG(time.Unix(100, 0)); ok || svg != "" {
		t.Fatalf("RenderSVG = %q, %v, want unavailable empty", svg, ok)
	}
	if html, ok := adapter.RenderHTML(time.Unix(100, 0)); ok || html != "" {
		t.Fatalf("RenderHTML = %q, %v, want unavailable empty", html, ok)
	}
}

func TestDisplayAdapterRendersNoDataHTMLAndSVG(t *testing.T) {
	adapter := NewDisplayAdapter(&fakeMomentumProvider{})
	now := time.Unix(100, 0)

	svg, ok := adapter.RenderSVG(now)
	if !ok {
		t.Fatal("RenderSVG should be available with provider")
	}
	html, ok := adapter.RenderHTML(now)
	if !ok {
		t.Fatal("RenderHTML should be available with provider")
	}

	for _, output := range []string{svg, html} {
		for _, want := range []string{
			`viewBox="0 0 320 320"`,
			`class="overlayhud-render-model is-inactive has-no-data is-stale"`,
			`>NO DATA</text>`,
		} {
			if !strings.Contains(output, want) {
				t.Fatalf("output missing %q: %s", want, output)
			}
		}
	}
	if !strings.Contains(html, "<!doctype html>") {
		t.Fatalf("RenderHTML missing wrapper: %s", html)
	}
}

func TestDisplayAdapterRendersActiveHTMLAndSVG(t *testing.T) {
	adapter := NewDisplayAdapter(activeMomentumProvider(time.Unix(100, 0)))
	now := time.Unix(101, 0)

	svg, ok := adapter.RenderSVG(now)
	if !ok {
		t.Fatal("RenderSVG should be available")
	}
	html, ok := adapter.RenderHTML(now)
	if !ok {
		t.Fatal("RenderHTML should be available")
	}

	for _, output := range []string{svg, html} {
		for _, want := range []string{
			`class="overlayhud-render-model is-active has-data"`,
			`>BLUE PRESSURE</text>`,
			`id="hud-momentum-blue"`,
		} {
			if !strings.Contains(output, want) {
				t.Fatalf("output missing %q: %s", want, output)
			}
		}
	}
}

func TestDisplayAdapterRendersStaleHTMLAndSVG(t *testing.T) {
	adapter := NewDisplayAdapter(activeMomentumProvider(time.Unix(100, 0)))
	now := time.Unix(100, 0).Add(staleSnapshotAfter + time.Second)

	svg, ok := adapter.RenderSVG(now)
	if !ok {
		t.Fatal("RenderSVG should be available")
	}
	html, ok := adapter.RenderHTML(now)
	if !ok {
		t.Fatal("RenderHTML should be available")
	}

	for _, output := range []string{svg, html} {
		for _, want := range []string{
			`class="overlayhud-render-model is-active has-data is-stale"`,
			`>STALE</text>`,
		} {
			if !strings.Contains(output, want) {
				t.Fatalf("output missing %q: %s", want, output)
			}
		}
	}
}

func TestDisplayAdapterDeterministicForSameProviderAndTime(t *testing.T) {
	adapter := NewDisplayAdapter(activeMomentumProvider(time.Unix(100, 0)))
	now := time.Unix(101, 0)

	first, ok := adapter.RenderHTML(now)
	if !ok {
		t.Fatal("RenderHTML should be available")
	}
	second, ok := adapter.RenderHTML(now)
	if !ok {
		t.Fatal("RenderHTML should be available")
	}
	if first != second {
		t.Fatalf("RenderHTML should be deterministic for same provider/time")
	}
}

func TestPreviewRouteUsesDisplayAdapterOutput(t *testing.T) {
	provider := activeMomentumProvider(time.Now())
	plugin := New(provider)
	mux := http.NewServeMux()
	plugin.Routes(mux)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, previewRoutePath, nil)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		"<!doctype html>",
		`viewBox="0 0 320 320"`,
		`>BLUE PRESSURE</text>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("route body missing adapter output %q: %s", want, body)
		}
	}
}

func activeMomentumProvider(occurredAt time.Time) *fakeMomentumProvider {
	return &fakeMomentumProvider{
		state: momentum.MomentumState{
			Sequence: 1,
			Teams: map[oofevents.Team]momentum.TeamSignal{
				oofevents.TeamBlue: {
					MomentumInfluence: 0.72,
					Confidence:        0.76,
					Volatility:        0.24,
				},
				oofevents.TeamOrange: {
					MomentumInfluence: 0.28,
					Confidence:        0.76,
					Volatility:        0.24,
				},
			},
			LastEvent: momentum.EventSignal{OccurredAt: occurredAt},
		},
		status: momentum.ServiceStatus{Active: true},
	}
}
