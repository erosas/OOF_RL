package overlayhud

import (
	"fmt"
	"net/http"
)

const controlRoutePath = "/internal/momentum-overlay-control"

func (p *Plugin) handleControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	action := r.URL.Query().Get("action")
	if action == "" {
		action = "status"
	}

	p.launchShellMu.Lock()
	defer p.launchShellMu.Unlock()

	switch action {
	case "status":
	case "show":
		if p.launchedShell != nil {
			p.launchedShell.Show()
			p.shellState = overlayShellVisible
		}
	case "hide":
		if p.launchedShell != nil {
			p.launchedShell.Hide()
			p.shellState = overlayShellHidden
		}
	case "dormant":
		if p.launchedShell != nil {
			p.launchedShell.Dormant()
			p.shellState = overlayShellDormant
		}
	default:
		http.Error(w, "unknown action", http.StatusBadRequest)
		return
	}

	writeControlStatus(w, action, p.currentShellState())
}

func (p *Plugin) currentShellState() overlayShellState {
	if p.launchedShell == nil {
		return overlayShellNotLaunched
	}
	if p.shellState == "" {
		return overlayShellHidden
	}
	return p.shellState
}

func writeControlStatus(w http.ResponseWriter, action string, state overlayShellState) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintf(w, "momentum overlay control action=%s\nstate=%s\n", action, state)
}
