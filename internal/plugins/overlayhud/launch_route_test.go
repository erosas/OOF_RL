package overlayhud

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"OOF_RL/internal/config"
	"OOF_RL/internal/overlay"
)

func TestLaunchRouteStartsManualShell(t *testing.T) {
	cfg := config.Defaults()
	plugin := NewWithConfig(&fakeMomentumProvider{}, &cfg)

	var calls int
	var gotURL string
	var gotTitle string
	var gotVisible bool
	plugin.launchShell = func(url string, cfg *config.Config, opts overlay.ManualShellOptions) any {
		calls++
		gotURL = url
		gotTitle = opts.Title
		gotVisible = opts.Visible
		return struct{}{}
	}

	mux := http.NewServeMux()
	plugin.Routes(mux)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080"+launchRoutePath, nil)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if calls != 1 {
		t.Fatalf("launch calls = %d, want 1", calls)
	}
	if gotURL != "http://localhost:8080/internal/momentum-overlay-preview" {
		t.Fatalf("launch URL = %q, want preview URL", gotURL)
	}
	if gotTitle != "Momentum Overlay" {
		t.Fatalf("title = %q, want Momentum Overlay", gotTitle)
	}
	if !gotVisible {
		t.Fatal("manual launch should request visible shell")
	}
	body := w.Body.String()
	if !strings.Contains(body, "momentum overlay shell launched") || !strings.Contains(body, gotURL) {
		t.Fatalf("body = %q, want launch status and URL", body)
	}
}

func TestLaunchRouteReusesExistingShell(t *testing.T) {
	cfg := config.Defaults()
	plugin := NewWithConfig(&fakeMomentumProvider{}, &cfg)

	var calls int
	plugin.launchShell = func(url string, cfg *config.Config, opts overlay.ManualShellOptions) any {
		calls++
		return struct{}{}
	}

	mux := http.NewServeMux()
	plugin.Routes(mux)

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "http://localhost:8080"+launchRoutePath, nil)
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("launch %d status = %d, body: %s", i+1, w.Code, w.Body.String())
		}
	}

	if calls != 1 {
		t.Fatalf("launch calls = %d, want 1", calls)
	}
}

func TestLaunchRouteRequiresConfig(t *testing.T) {
	plugin := New(&fakeMomentumProvider{})
	mux := http.NewServeMux()
	plugin.Routes(mux)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080"+launchRoutePath, nil)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestLaunchRouteRejectsNonGET(t *testing.T) {
	cfg := config.Defaults()
	plugin := NewWithConfig(&fakeMomentumProvider{}, &cfg)
	mux := http.NewServeMux()
	plugin.Routes(mux)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://localhost:8080"+launchRoutePath, nil)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

func TestBaseAppURLFromRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080"+launchRoutePath, nil)
	if got := baseAppURLFromRequest(req); got != "http://localhost:8080" {
		t.Fatalf("baseAppURLFromRequest = %q, want http://localhost:8080", got)
	}

	req.Header.Set("X-Forwarded-Proto", "https")
	if got := baseAppURLFromRequest(req); got != "https://localhost:8080" {
		t.Fatalf("baseAppURLFromRequest forwarded = %q, want https://localhost:8080", got)
	}

	req.Header.Set("X-Forwarded-Proto", "http")
	if got := baseAppURLFromRequest(req); got != "http://localhost:8080" {
		t.Fatalf("baseAppURLFromRequest forwarded http = %q, want http://localhost:8080", got)
	}
}

func TestBaseAppURLFromRequestRejectsUnsafeForwardedProto(t *testing.T) {
	for _, proto := range []string{"javascript", "file", "ftp", "ws", "://bad"} {
		req := httptest.NewRequest(http.MethodGet, "http://localhost:8080"+launchRoutePath, nil)
		req.Header.Set("X-Forwarded-Proto", proto)

		if got := baseAppURLFromRequest(req); got != "http://localhost:8080" {
			t.Fatalf("baseAppURLFromRequest proto %q = %q, want http fallback", proto, got)
		}
	}
}

func TestBaseAppURLFromRequestEmptyForwardedProtoFallsBack(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080"+launchRoutePath, nil)
	req.Header.Set("X-Forwarded-Proto", "   ")
	if got := baseAppURLFromRequest(req); got != "http://localhost:8080" {
		t.Fatalf("baseAppURLFromRequest empty forwarded = %q, want http fallback", got)
	}
}

func TestBaseAppURLFromRequestEmptyHostIsPredictable(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080"+launchRoutePath, nil)
	req.Host = ""

	if got := baseAppURLFromRequest(req); got != "http://" {
		t.Fatalf("baseAppURLFromRequest empty host = %q, want predictable empty-host URL", got)
	}
}
