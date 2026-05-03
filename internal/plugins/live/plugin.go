package live

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"sync"

	"OOF_RL/internal/events"
	"OOF_RL/internal/httputil"
	"OOF_RL/internal/plugin"
)

//go:embed view.html view.js
var viewFS embed.FS

// Plugin caches the current game state and serves it to new page loads.
// It receives updates via HandleEvent once the Bus is wired to rl.Client (Phase 10).
type Plugin struct {
	mu    sync.RWMutex
	state *events.UpdateStateData // nil = no active match
}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string         { return "live" }
func (p *Plugin) DBPrefix() string   { return "" }
func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) NavTab() plugin.NavTab {
	return plugin.NavTab{ID: "live", Label: "Live", Order: 10}
}

func (p *Plugin) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/live/state", p.handleState)
}

func (p *Plugin) SettingsSchema() []plugin.Setting        { return nil }
func (p *Plugin) ApplySettings(_ map[string]string) error { return nil }
func (p *Plugin) Assets() fs.FS                           { return viewFS }

func (p *Plugin) HandleEvent(env events.Envelope) {
	switch env.Event {
	case "UpdateState":
		var d events.UpdateStateData
		if err := json.Unmarshal(env.Data, &d); err == nil {
			p.mu.Lock()
			p.state = &d
			p.mu.Unlock()
		}
	case "MatchDestroyed":
		p.mu.Lock()
		p.state = nil
		p.mu.Unlock()
	}
}

// handleState returns the last known game state so the frontend can restore
// the live view immediately on page load without waiting for the next tick.
func (p *Plugin) handleState(w http.ResponseWriter, r *http.Request) {
	p.mu.RLock()
	state := p.state
	p.mu.RUnlock()

	if state == nil {
		httputil.WriteJSON(w, map[string]any{"active": false})
		return
	}
	httputil.WriteJSON(w, map[string]any{"active": true, "state": state})
}