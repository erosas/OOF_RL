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

func TestPreviewRouteRendersCurrentMomentumSVG(t *testing.T) {
	provider := &fakeMomentumProvider{
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
			LastEvent: momentum.EventSignal{OccurredAt: time.Now()},
		},
		status: momentum.ServiceStatus{Active: true},
	}
	plugin := New(provider)
	mux := http.NewServeMux()
	plugin.Routes(mux)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, previewRoutePath, nil)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if contentType := w.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", contentType)
	}
	body := w.Body.String()
	for _, want := range []string{
		"<!doctype html>",
		`viewBox="0 0 320 320"`,
		`id="hud-root"`,
		`>--:--</text>`,
		`>BLUE PRESSURE</text>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("preview body missing %q: %s", want, body)
		}
	}
}

func TestPreviewRouteRequiresMomentumProvider(t *testing.T) {
	plugin := New(nil)
	mux := http.NewServeMux()
	plugin.Routes(mux)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, previewRoutePath, nil)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestPreviewRouteRejectsNonGET(t *testing.T) {
	plugin := New(&fakeMomentumProvider{})
	mux := http.NewServeMux()
	plugin.Routes(mux)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, previewRoutePath, nil)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

func TestOverlayHUDPluginRemainsHiddenFromNav(t *testing.T) {
	plugin := New(&fakeMomentumProvider{})

	if tab := plugin.NavTab(); tab.ID != "" || tab.Label != "" || tab.Order != 0 {
		t.Fatalf("NavTab = %+v, want empty hidden tab", tab)
	}
}
