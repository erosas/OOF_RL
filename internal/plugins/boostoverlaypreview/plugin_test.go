package boostoverlaypreview

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPreviewRouteServesFixturePage(t *testing.T) {
	p := New()
	mux := http.NewServeMux()
	p.Routes(mux)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, previewRoute, nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("preview route status = %d, want 200: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Fixture Boost Preview") {
		t.Fatalf("preview page missing title: %s", body)
	}
	if !strings.Contains(body, "Not live match data") {
		t.Fatalf("preview page missing live-data boundary copy: %s", body)
	}
}

func TestPreviewRouteServesFixtureJSON(t *testing.T) {
	p := New()
	mux := http.NewServeMux()
	p.Routes(mux)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, previewRoute+"/fixture.json", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("fixture status = %d, want 200: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("fixture content type = %q, want application/json", ct)
	}
	if !strings.Contains(rr.Body.String(), "boostOverlaySignal") {
		t.Fatalf("fixture missing boost overlay signal: %s", rr.Body.String())
	}
}

func TestPreviewRouteRejectsUnknownAssets(t *testing.T) {
	p := New()
	mux := http.NewServeMux()
	p.Routes(mux)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, previewRoute+"/missing.json", nil))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("unknown asset status = %d, want 404", rr.Code)
	}
}

func TestPreviewPluginStaysHiddenFromNavAndSettings(t *testing.T) {
	p := New()
	if tab := p.NavTab(); tab.ID != "" || tab.Label != "" {
		t.Fatalf("NavTab = %+v, want empty hidden tab", tab)
	}
	if settings := p.SettingsSchema(); len(settings) != 0 {
		t.Fatalf("SettingsSchema length = %d, want 0", len(settings))
	}
}
