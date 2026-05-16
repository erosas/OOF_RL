// Package overlayhud contains the display-only Overlay HUD consumer rebuild.
// This slice intentionally contains no routes, assets, rendering, or overlay
// lifecycle behavior yet. It only proves read-only Momentum consumption.
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
func (p *Plugin) Routes(_ *http.ServeMux)                 {}
func (p *Plugin) SettingsSchema() []plugin.Setting        { return nil }
func (p *Plugin) ApplySettings(_ map[string]string) error { return nil }
func (p *Plugin) Assets() fs.FS                           { return nil }

func (p *Plugin) momentumSnapshot() (momentum.MomentumState, momentum.ServiceStatus, bool) {
	if p.momentum == nil {
		return momentum.MomentumState{}, momentum.ServiceStatus{}, false
	}
	return p.momentum.Snapshot(), p.momentum.Status(), true
}
