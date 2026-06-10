package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func guardedRecorder(t *testing.T, method, target string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	called := false
	h := LocalhostGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(method, target, nil)
	for k, v := range headers {
		if k == "Host" {
			req.Host = v
			continue
		}
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code == http.StatusOK && !called {
		t.Fatal("handler not called despite 200")
	}
	return w
}

func TestLocalhostGuardAllowsLoopbackHosts(t *testing.T) {
	for _, host := range []string{"localhost:8080", "127.0.0.1:8080", "localhost", "127.0.0.1", "[::1]:8080"} {
		w := guardedRecorder(t, http.MethodGet, "/api/config", map[string]string{"Host": host})
		if w.Code != http.StatusOK {
			t.Errorf("host %q: got %d, want 200", host, w.Code)
		}
	}
}

func TestLocalhostGuardRejectsForeignHost(t *testing.T) {
	// DNS rebinding: attacker's hostname resolves to 127.0.0.1 but the
	// browser still sends the attacker's domain in Host.
	for _, host := range []string{"evil.example:8080", "evil.example", "192.168.1.5:8080"} {
		w := guardedRecorder(t, http.MethodGet, "/api/config", map[string]string{"Host": host})
		if w.Code != http.StatusForbidden {
			t.Errorf("host %q: got %d, want 403", host, w.Code)
		}
	}
}

func TestLocalhostGuardRejectsCrossOriginPost(t *testing.T) {
	w := guardedRecorder(t, http.MethodPost, "/api/config", map[string]string{
		"Host":   "localhost:8080",
		"Origin": "https://evil.example",
	})
	if w.Code != http.StatusForbidden {
		t.Errorf("got %d, want 403", w.Code)
	}
}

func TestLocalhostGuardAllowsLoopbackOriginPost(t *testing.T) {
	for _, origin := range []string{"http://localhost:8080", "http://127.0.0.1:8081"} {
		w := guardedRecorder(t, http.MethodPost, "/api/settings", map[string]string{
			"Host":   "localhost:8080",
			"Origin": origin,
		})
		if w.Code != http.StatusOK {
			t.Errorf("origin %q: got %d, want 200", origin, w.Code)
		}
	}
}

func TestLocalhostGuardAllowsOriginlessPost(t *testing.T) {
	// WebView2 and CLI clients send no Origin header.
	w := guardedRecorder(t, http.MethodPost, "/api/settings", map[string]string{"Host": "localhost:8080"})
	if w.Code != http.StatusOK {
		t.Errorf("got %d, want 200", w.Code)
	}
}

func TestLocalhostGuardIgnoresOriginOnGet(t *testing.T) {
	// Cross-origin GETs can't be blocked meaningfully (no CORS read access
	// anyway once Host is enforced); only state-changing methods check Origin.
	w := guardedRecorder(t, http.MethodGet, "/api/config", map[string]string{
		"Host":   "localhost:8080",
		"Origin": "https://evil.example",
	})
	if w.Code != http.StatusOK {
		t.Errorf("got %d, want 200", w.Code)
	}
}