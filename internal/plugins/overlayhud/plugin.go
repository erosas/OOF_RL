// Package overlayhud contains the display-only Overlay HUD consumer rebuild.
// It consumes read-only Momentum state and exposes only narrowly scoped display
// surfaces; native overlay/window lifecycle behavior is intentionally separate.
package overlayhud

import (
	"io/fs"
	"net/http"
	"sync"

	"OOF_RL/internal/config"
	"OOF_RL/internal/db"
	"OOF_RL/internal/momentum"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/overlay"
	"OOF_RL/internal/plugin"
)

func init() {
	plugin.Register("overlayhud", func() plugin.Plugin {
		return &Plugin{
			launchShell: startManualShell,
		}
	})
}

// Plugin is a display-only Overlay HUD consumer skeleton. It can read Momentum
// snapshots through a read-only provider, but it does not own or mutate
// Momentum runtime state.
type Plugin struct {
	plugin.BasePlugin
	momentum      momentum.SnapshotProvider
	cfg           *config.Config
	launchShell   manualShellLauncher
	launchShellMu sync.Mutex
	launchedShell any
}

func New(provider momentum.SnapshotProvider) *Plugin {
	return NewWithConfig(provider, nil)
}

func NewWithConfig(provider momentum.SnapshotProvider, cfg *config.Config) *Plugin {
	return &Plugin{
		momentum:    provider,
		cfg:         cfg,
		launchShell: startManualShell,
	}
}

func (p *Plugin) ID() string         { return "overlayhud" }
func (p *Plugin) DBPrefix() string   { return "" }
func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) NavTab() plugin.NavTab { return plugin.NavTab{} }
func (p *Plugin) Routes(mux *http.ServeMux) {
	mux.HandleFunc(previewRoutePath, p.handlePreview)
	mux.HandleFunc(launchRoutePath, p.handleLaunch)
}
func (p *Plugin) SettingsSchema() []plugin.Setting        { return nil }
func (p *Plugin) ApplySettings(_ map[string]string) error { return nil }
func (p *Plugin) Assets() fs.FS                           { return nil }

func (p *Plugin) Init(_ oofevents.PluginBus, reg plugin.Registry, _ *db.DB) error {
	p.momentum = reg.Momentum()
	p.cfg = reg.Config()
	return nil
}

func (p *Plugin) momentumSnapshot() (momentum.MomentumState, momentum.ServiceStatus, bool) {
	if p.momentum == nil {
		return momentum.MomentumState{}, momentum.ServiceStatus{}, false
	}
	return p.momentum.Snapshot(), p.momentum.Status(), true
}

type manualShellLauncher func(url string, cfg *config.Config, opts overlay.ManualShellOptions) any

func startManualShell(url string, cfg *config.Config, opts overlay.ManualShellOptions) any {
	return overlay.StartManualShell(url, cfg, opts)
}
