package history

import (
	"embed"
	"io/fs"
	"net/http"

	"OOF_RL/internal/db"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/plugin"
)

//go:embed view.html view.js
var viewFS embed.FS

func init() {
	plugin.Register("history", func() plugin.Plugin { return &Plugin{} })
}

// Plugin provides the History nav tab and its embedded assets.
// All event ingestion and REST API are handled by internal/histstore wired into core.
type Plugin struct {
	plugin.BasePlugin
}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string         { return "history" }
func (p *Plugin) DBPrefix() string   { return "" }
func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) NavTab() plugin.NavTab {
	return plugin.NavTab{ID: "history", Label: "History", Order: 20}
}

func (p *Plugin) Routes(_ *http.ServeMux)              {} // routes owned by core
func (p *Plugin) SettingsSchema() []plugin.Setting     { return nil }
func (p *Plugin) ApplySettings(_ map[string]string) error { return nil }
func (p *Plugin) Assets() fs.FS                        { return viewFS }

func (p *Plugin) Init(_ oofevents.PluginBus, _ plugin.Registry, _ *db.DB) error { return nil }