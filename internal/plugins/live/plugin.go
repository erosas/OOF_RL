package live

import (
	"embed"
	"io/fs"
	"net/http"
	"sync"

	"OOF_RL/internal/db"
	"OOF_RL/internal/httputil"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/plugin"
)

//go:embed view.html view.js
var viewFS embed.FS

type Plugin struct {
	plugin.BasePlugin
	mu    sync.RWMutex
	state *oofevents.StateUpdatedEvent // nil = no active match
	subs  []oofevents.Subscription
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

func (p *Plugin) Init(bus oofevents.PluginBus, _ plugin.Registry, _ *db.DB) error {
	p.subs = []oofevents.Subscription{
		bus.Subscribe(oofevents.TypeStateUpdated, p.onStateUpdated),
		bus.Subscribe(oofevents.TypeMatchDestroyed, p.onMatchDestroyed),
	}
	return nil
}

func (p *Plugin) Shutdown() error {
	for _, s := range p.subs {
		s.Cancel()
	}
	return nil
}

func (p *Plugin) onStateUpdated(e oofevents.OOFEvent) {
	ev, ok := e.(oofevents.StateUpdatedEvent)
	if !ok {
		return
	}
	p.mu.Lock()
	p.state = &ev
	p.mu.Unlock()
}

func (p *Plugin) onMatchDestroyed(_ oofevents.OOFEvent) {
	p.mu.Lock()
	p.state = nil
	p.mu.Unlock()
}

// handleState returns the last known game state so the frontend can hydrate
// the live view immediately on page load without waiting for the next tick.
func (p *Plugin) handleState(w http.ResponseWriter, r *http.Request) {
	p.mu.RLock()
	state := p.state
	p.mu.RUnlock()

	if state == nil {
		httputil.WriteJSON(w, map[string]any{"active": false})
		return
	}
	httputil.WriteJSON(w, map[string]any{
		"active": true,
		"state": map[string]any{
			"MatchGuid": state.MatchGUID(),
			"Players":   state.Players,
			"Game":      state.Game,
		},
	})
}