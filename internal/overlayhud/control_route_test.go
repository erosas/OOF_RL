package overlayhud

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"OOF_RL/internal/config"
	"OOF_RL/internal/overlay"
)

type fakeManualShell struct {
	showCalls    int
	hideCalls    int
	dormantCalls int
}

func (s *fakeManualShell) Show()    { s.showCalls++ }
func (s *fakeManualShell) Hide()    { s.hideCalls++ }
func (s *fakeManualShell) Dormant() { s.dormantCalls++ }

func TestControlRouteReportsNotLaunched(t *testing.T) {
	plugin := New(&fakeMomentumProvider{})
	mux := http.NewServeMux()
	plugin.Routes(mux)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, controlRoutePath+"?action=status", nil)
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); !strings.Contains(body, "action=status") || !strings.Contains(body, "state=not-launched") {
		t.Fatalf("body = %q, want not-launched status", body)
	}
}

func TestControlRouteDoesNotCreateShellOnHideOrDormant(t *testing.T) {
	cfg := config.Defaults()
	plugin := NewWithConfig(&fakeMomentumProvider{}, &cfg)

	var calls int
	plugin.launchShell = func(url string, cfg *config.Config, opts overlay.ManualShellOptions) manualOverlayShell {
		calls++
		return &fakeManualShell{}
	}

	mux := http.NewServeMux()
	plugin.Routes(mux)

	for _, action := range []string{"hide", "dormant"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, controlRoutePath+"?action="+action, nil)
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("%s status = %d, body: %s", action, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "state=not-launched") {
			t.Fatalf("%s body = %q, want not-launched", action, w.Body.String())
		}
	}

	if calls != 0 {
		t.Fatalf("launch calls = %d, want 0", calls)
	}
}

func TestControlRouteShowHideDormantExistingShell(t *testing.T) {
	shell := &fakeManualShell{}
	plugin := New(&fakeMomentumProvider{})
	plugin.launchedShell = shell
	plugin.shellState = overlayShellHidden

	mux := http.NewServeMux()
	plugin.Routes(mux)

	tests := []struct {
		action string
		state  string
	}{
		{action: "show", state: "visible"},
		{action: "hide", state: "hidden"},
		{action: "dormant", state: "dormant"},
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, controlRoutePath+"?action="+tt.action, nil)
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("%s status = %d, body: %s", tt.action, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "state="+tt.state) {
			t.Fatalf("%s body = %q, want state %s", tt.action, w.Body.String(), tt.state)
		}
	}

	if shell.showCalls != 1 {
		t.Fatalf("showCalls = %d, want 1", shell.showCalls)
	}
	if shell.hideCalls != 1 {
		t.Fatalf("hideCalls = %d, want 1", shell.hideCalls)
	}
	if shell.dormantCalls != 1 {
		t.Fatalf("dormantCalls = %d, want 1", shell.dormantCalls)
	}
}

func TestControlRouteRepeatedShowDoesNotCreateShell(t *testing.T) {
	cfg := config.Defaults()
	shell := &fakeManualShell{}
	plugin := NewWithConfig(&fakeMomentumProvider{}, &cfg)
	plugin.launchedShell = shell
	plugin.shellState = overlayShellVisible

	var calls int
	plugin.launchShell = func(url string, cfg *config.Config, opts overlay.ManualShellOptions) manualOverlayShell {
		calls++
		return &fakeManualShell{}
	}

	mux := http.NewServeMux()
	plugin.Routes(mux)

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, controlRoutePath+"?action=show", nil)
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("show %d status = %d, body: %s", i+1, w.Code, w.Body.String())
		}
	}

	if calls != 0 {
		t.Fatalf("launch calls = %d, want 0", calls)
	}
	if shell.showCalls != 2 {
		t.Fatalf("showCalls = %d, want 2", shell.showCalls)
	}
}

func TestControlRouteRejectsUnknownActionAndNonGET(t *testing.T) {
	plugin := New(&fakeMomentumProvider{})
	mux := http.NewServeMux()
	plugin.Routes(mux)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, controlRoutePath+"?action=launch", nil)
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("unknown action status = %d, want 400", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, controlRoutePath+"?action=status", nil)
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("post status = %d, want 405", w.Code)
	}
}
