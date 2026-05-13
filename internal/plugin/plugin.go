package plugin

import (
	"io/fs"
	"net/http"

	"OOF_RL/internal/db"
	"OOF_RL/internal/events"
	"OOF_RL/internal/oofevents"
)

// NavTab describes a navigation entry the plugin contributes to the UI.
type NavTab struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Order int    `json:"order"`
}

// Registry allows plugins to look up other registered plugins.
type Registry interface {
	Get(id string) (Plugin, bool)
	List() []Plugin
}

// Analyzer is a background computation unit. No UI, no routes.
// Subscribe to events, emit events, optionally persist data.
type Analyzer interface {
	ID() string
	DBPrefix() string
	Requires() []string
	Init(bus oofevents.PluginBus, registry Registry, db *db.DB) error
	Shutdown() error
	SettingsSchema() []Setting
	ApplySettings(map[string]string) error
	DeclaredEvents() []oofevents.EventDeclaration
}

// Plugin extends Analyzer with a UI tab, HTTP routes, and static assets.
type Plugin interface {
	Analyzer
	NavTab() NavTab
	Routes(mux *http.ServeMux)
	Assets() fs.FS

	// HandleEvent delivers a raw RL envelope. Deprecated: plugins should
	// subscribe in Init instead. Kept as an adapter during migration.
	HandleEvent(env events.Envelope)
}

// BasePlugin provides no-op implementations of the lifecycle methods added
// in the event-bus migration. Embed this in existing plugins to satisfy the
// updated interface without changing any existing logic.
type BasePlugin struct{}

func (BasePlugin) Init(_ oofevents.PluginBus, _ Registry, _ *db.DB) error { return nil }
func (BasePlugin) Shutdown() error                                         { return nil }
func (BasePlugin) DeclaredEvents() []oofevents.EventDeclaration            { return nil }
func (BasePlugin) HandleEvent(_ events.Envelope)                           {}