// Package overlayhud contains the display-only Overlay HUD consumer rebuild.
// It consumes read-only Momentum state and exposes only narrowly scoped display
// surfaces; native overlay/window lifecycle behavior is intentionally separate.
package overlayhud

import (
	"io/fs"
	"net/http"

	"OOF_RL/internal/momentum"
	"OOF_RL/internal/plugin"
)

// Plugin is a display-only Overlay HUD consumer skeleton. It can read Momentum
// snapshots through a read-only provider, but it does not own or mutate
// Momentum runtime state.
type Plugin struct {
	plugin.BasePlugin
	momentum momentum.SnapshotProvider
}

func New(provider momentum.SnapshotProvider) *Plugin {
	return &Plugin{momentum: provider}
}

func (p *Plugin) ID() string         { return "overlayhud" }
func (p *Plugin) DBPrefix() string   { return "" }
func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) NavTab() plugin.NavTab                   { return plugin.NavTab{} }
func (p *Plugin) Routes(mux *http.ServeMux)               { mux.HandleFunc(previewRoutePath, p.handlePreview) }
func (p *Plugin) SettingsSchema() []plugin.Setting        { return nil }
func (p *Plugin) ApplySettings(_ map[string]string) error { return nil }
func (p *Plugin) Assets() fs.FS                           { return nil }

func (p *Plugin) momentumSnapshot() (momentum.MomentumState, momentum.ServiceStatus, bool) {
	if p.momentum == nil {
		return momentum.MomentumState{}, momentum.ServiceStatus{}, false
	}
	return p.momentum.Snapshot(), p.momentum.Status(), true
}
