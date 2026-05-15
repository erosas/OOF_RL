package momentumtimeline

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
	if !strings.Contains(body, "Momentum Timeline B-lite") {
		t.Fatalf("preview page missing title: %s", body)
	}
	if !strings.Contains(body, "frontend fixture data only") {
		t.Fatalf("preview page missing fixture-only boundary copy: %s", body)
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
	if !strings.Contains(rr.Body.String(), "pressureContribution") {
		t.Fatalf("fixture missing contribution fields: %s", rr.Body.String())
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
