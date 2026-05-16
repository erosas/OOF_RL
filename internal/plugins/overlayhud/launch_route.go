package overlayhud

import (
	"fmt"
	"net/http"
	"strings"

	"OOF_RL/internal/overlay"
)

const launchRoutePath = "/internal/momentum-overlay-launch"

func (p *Plugin) handleLaunch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if p.cfg == nil {
		http.Error(w, "overlay config unavailable", http.StatusServiceUnavailable)
		return
	}

	spec, ok := MomentumNativeShellSpec(baseAppURLFromRequest(r))
	if !ok || !spec.Valid() {
		http.Error(w, "momentum overlay target unavailable", http.StatusInternalServerError)
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
		http.Error(w, "failed to launch momentum overlay shell", http.StatusServiceUnavailable)
		return
	}
	p.launchedShell = shell
	writeLaunchStatus(w, "launched", spec.URL)
}

func baseAppURLFromRequest(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = forwarded
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

func writeLaunchStatus(w http.ResponseWriter, status string, targetURL string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintf(w, "momentum overlay shell %s\n%s\n", status, targetURL)
}
