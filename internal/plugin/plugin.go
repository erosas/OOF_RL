package plugin

import (
	"io/fs"
	"maps"
	"net/http"
	"sync"

	"OOF_RL/internal/config"
	"OOF_RL/internal/db"
	"OOF_RL/internal/oofevents"
)

// Factory creates a new instance of a plugin.
type Factory func() Plugin

var (
	registryMu sync.RWMutex
	factories  = make(map[string]Factory)
)

// Register adds a plugin factory to the global registry.
// Called by plugins in their init() functions.
func Register(id string, f Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	factories[id] = f
}

// Unregister removes a factory from the global registry.
// Only for use in tests — use cfg.DisabledPlugins to toggle plugins at runtime.
func Unregister(id string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	delete(factories, id)
}

// Factories returns a map of all registered plugin factories.
func Factories() map[string]Factory {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return maps.Clone(factories)
}

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
	Config() *config.Config
}

// Plugin is the full contract for a UI plugin: lifecycle, routes, and assets.
type Plugin interface {
	ID() string
	Requires() []string
	Init(bus oofevents.PluginBus, registry Registry, db *db.DB) error
	Shutdown() error
	SettingsSchema() []Setting
	ApplySettings(map[string]string) error
	DeclaredEvents() []oofevents.EventDeclaration
	NavTab() NavTab
	Routes(mux *http.ServeMux)
	// RoutePaths returns the URL paths this plugin handles. Used for conflict
	// detection at registration time. WASM plugins return their declared routes;
	// native plugins that embed BasePlugin return nil (they are trusted).
	RoutePaths() []string
	Assets() fs.FS
}

// BasePlugin provides no-op implementations of the lifecycle methods.
// Embed this in plugins to satisfy the interface without boilerplate.
type BasePlugin struct {
	Subs []oofevents.Subscription
}

func (p *BasePlugin) RoutePaths() []string                                     { return nil }
func (p *BasePlugin) Init(_ oofevents.PluginBus, _ Registry, _ *db.DB) error  { return nil }

func (p *BasePlugin) Shutdown() error {
	for _, s := range p.Subs {
		s.Cancel()
	}
	p.Subs = nil
	return nil
}

func (p *BasePlugin) DeclaredEvents() []oofevents.EventDeclaration { return nil }

func (p *BasePlugin) AddSub(s oofevents.Subscription) {
	p.Subs = append(p.Subs, s)
}
