package ranks

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

type rankPlayer struct {
	PrimaryID string `json:"primary_id"`
	Name      string `json:"name"`
	TeamNum   int    `json:"team_num"`
}

type Plugin struct {
	plugin.BasePlugin
	mu      sync.RWMutex
	players []rankPlayer
	subs    []oofevents.Subscription
}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string         { return "ranks" }
func (p *Plugin) DBPrefix() string   { return "" }
func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) NavTab() plugin.NavTab {
	return plugin.NavTab{ID: "ranks", Label: "Ranks", Order: 15}
}

func (p *Plugin) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/ranks/players", p.handlePlayers)
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
	players := make([]rankPlayer, 0, len(ev.Players))
	for _, pl := range ev.Players {
		players = append(players, rankPlayer{
			PrimaryID: pl.PrimaryID,
			Name:      pl.Name,
			TeamNum:   pl.TeamNum,
		})
	}
	p.mu.Lock()
	p.players = players
	p.mu.Unlock()
}

func (p *Plugin) onMatchDestroyed(_ oofevents.OOFEvent) {
	p.mu.Lock()
	p.players = nil
	p.mu.Unlock()
}

func (p *Plugin) handlePlayers(w http.ResponseWriter, r *http.Request) {
	p.mu.RLock()
	players := p.players
	p.mu.RUnlock()
	if players == nil {
		httputil.WriteJSON(w, []rankPlayer{})
		return
	}
	httputil.WriteJSON(w, players)
}