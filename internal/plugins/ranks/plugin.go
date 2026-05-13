package ranks

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

type rankPlayer struct {
	PrimaryID string `json:"primary_id"`
	Name      string `json:"name"`
	TeamNum   int    `json:"team_num"`
}

type Plugin struct {
	plugin.BasePlugin
	mu      sync.RWMutex
	players []rankPlayer
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

func (p *Plugin) HandleEvent(env events.Envelope) {
	switch env.Event {
	case "UpdateState":
		var d events.UpdateStateData
		if err := json.Unmarshal(env.Data, &d); err != nil {
			return
		}
		players := make([]rankPlayer, 0, len(d.Players))
		for _, pl := range d.Players {
			players = append(players, rankPlayer{
				PrimaryID: pl.PrimaryId,
				Name:      pl.Name,
				TeamNum:   pl.TeamNum,
			})
		}
		p.mu.Lock()
		p.players = players
		p.mu.Unlock()
	case "MatchDestroyed":
		p.mu.Lock()
		p.players = nil
		p.mu.Unlock()
	}
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
