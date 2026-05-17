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
		`viewBox="0 0 1024 1024"`,
		`id="momentum-wheel-root"`,
		`>--:--</text>`,
		`>BLUE CONTROL</text>`,
		previewSVGRoutePath,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("preview body missing %q: %s", want, body)
		}
	}
}

func TestPreviewSVGRouteRendersCurrentMomentumSVG(t *testing.T) {
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
	req := httptest.NewRequest(http.MethodGet, previewSVGRoutePath, nil)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if contentType := w.Header().Get("Content-Type"); !strings.Contains(contentType, "image/svg+xml") {
		t.Fatalf("Content-Type = %q, want image/svg+xml", contentType)
	}
	if cache := w.Header().Get("Cache-Control"); cache != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", cache)
	}
	body := w.Body.String()
	for _, want := range []string{
		`<svg `,
		`viewBox="0 0 1024 1024"`,
		`data-state="blue-control"`,
		`>BLUE CONTROL</text>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("preview svg body missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "<!doctype html>") {
		t.Fatalf("preview svg route should not return HTML wrapper: %s", body)
	}
}

func TestPreviewRouteRendersNoDataState(t *testing.T) {
	plugin := New(&fakeMomentumProvider{})
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
		`viewBox="0 0 1024 1024"`,
		`mcw-state-no-data`,
		`is-inactive`,
		`has-no-data`,
		`is-stale`,
		`>NO DATA</text>`,
		`>STALE</text>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("no-data preview body missing %q: %s", want, body)
		}
	}
}

func TestPreviewRouteRendersStaleState(t *testing.T) {
	provider := &fakeMomentumProvider{
		state: momentum.MomentumState{
			Sequence: 1,
			Teams: map[oofevents.Team]momentum.TeamSignal{
				oofevents.TeamBlue: {
					MomentumInfluence: 0.64,
					Confidence:        0.51,
					Volatility:        0.17,
				},
				oofevents.TeamOrange: {
					MomentumInfluence: 0.36,
					Confidence:        0.51,
					Volatility:        0.17,
				},
			},
			LastEvent: momentum.EventSignal{OccurredAt: time.Now().Add(-staleSnapshotAfter - time.Second)},
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
	body := w.Body.String()
	for _, want := range []string{
		`mcw-state-stale`,
		`is-active`,
		`has-data`,
		`is-stale`,
		`>STALE</text>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("stale preview body missing %q: %s", want, body)
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

func TestPreviewSVGRouteRequiresMomentumProvider(t *testing.T) {
	plugin := New(nil)
	mux := http.NewServeMux()
	plugin.Routes(mux)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, previewSVGRoutePath, nil)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestPreviewSVGRouteRejectsNonGET(t *testing.T) {
	plugin := New(&fakeMomentumProvider{})
	mux := http.NewServeMux()
	plugin.Routes(mux)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, previewSVGRoutePath, nil)
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
	if schema := plugin.SettingsSchema(); schema != nil {
		t.Fatalf("SettingsSchema = %+v, want nil for hidden internal preview", schema)
	}
}
