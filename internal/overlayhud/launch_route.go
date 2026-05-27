package overlayhud

import (
	"fmt"
	"net/http"
	"strings"

	"OOF_RL/internal/httputil"
	"OOF_RL/internal/overlay"
)

const launchRoutePath = "/internal/momentum-overlay-launch"

func (p *Plugin) handleLaunch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.JSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if p.cfg == nil {
		httputil.JSONError(w, http.StatusServiceUnavailable, "overlay config unavailable")
		return
	}

	spec, ok := MomentumNativeShellSpec(baseAppURLFromRequest(r))
	if !ok || !spec.Valid() {
		httputil.JSONError(w, http.StatusInternalServerError, "momentum overlay target unavailable")
		return
	}

	p.launchShellMu.Lock()
	defer p.launchShellMu.Unlock()

	if p.launchedShell != nil {
		writeLaunchStatus(w, "already-running", spec.URL)
		return
	}

	launcher := p.launchShell
	if launcher == nil {
		launcher = startManualShell
	}
	shell := launcher(spec.URL, p.cfg, overlay.ManualShellOptions{Title: spec.Title, Visible: true})
	if shell == nil {
		httputil.JSONError(w, http.StatusServiceUnavailable, "failed to launch momentum overlay shell")
		return
	}
	p.launchedShell = shell
	p.shellState = overlayShellVisible
	writeLaunchStatus(w, "launched", spec.URL)
}

func baseAppURLFromRequest(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := safeForwardedProto(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = forwarded
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

func safeForwardedProto(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "http", "https":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func writeLaunchStatus(w http.ResponseWriter, status string, targetURL string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintf(w, "momentum overlay shell %s\n%s\n", status, targetURL)
}
