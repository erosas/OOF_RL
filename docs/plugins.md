# Writing a Plugin

Plugins are the primary extension point. A plugin can contribute:

- background logic via the OOF event bus
- optional HTTP routes
- optional settings fields
- one browser view (`view.html` + `view.js`)

## Identity Contract (Important)

OOF RL now uses two identities with different purposes:

- `PluginID`: runtime/API/assets identity (from `Plugin.ID()`)
- `ViewID`: frontend navigation slug (from `Plugin.NavTab().ID`)

Use them like this:

- View HTML endpoint: `/api/plugins/{pluginID}/view`
- View script endpoint: `/plugins/{pluginID}/view.js`
- Frontend init hook: `window.pluginInit_<pluginID>()`
- Navigation state (`showView`, nav button `data-view`): `ViewID`



---

## Plugin Interface

Every plugin implements `plugin.Plugin` in `internal/plugin/plugin.go`.

```go
type Plugin interface {
    ID() string
    DBPrefix() string
    Requires() []string
    Init(bus oofevents.PluginBus, registry Registry, db *db.DB) error
    Shutdown() error
    SettingsSchema() []Setting
    ApplySettings(map[string]string) error
    DeclaredEvents() []oofevents.EventDeclaration

    NavTab() NavTab
    Routes(mux *http.ServeMux)
    Assets() fs.FS
}
```

Most plugins embed `plugin.BasePlugin` and only override what they use.

---

## Minimal Single-Page Plugin Example

### 1) `plugin.go`

```go
package pings

import (
    "embed"
    "io/fs"
    "net/http"
    "sync/atomic"

    "OOF_RL/internal/db"
    "OOF_RL/internal/httputil"
    "OOF_RL/internal/oofevents"
    "OOF_RL/internal/plugin"
    "OOF_RL/internal/rlevents"
)

//go:embed assets/view.html assets/view.js
var assetsFS embed.FS

type Plugin struct {
    plugin.BasePlugin
    count atomic.Int64
}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string         { return "pings" } // PluginID
func (p *Plugin) DBPrefix() string   { return "" }
func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) NavTab() plugin.NavTab {
    return plugin.NavTab{ID: "pings", Label: "Pings", Order: 50} // ViewID
}

func (p *Plugin) Init(bus oofevents.PluginBus, _ plugin.Registry, _ *db.DB) error {
    p.AddSub(bus.Subscribe(rlevents.TypeBallHit, func(oofevents.OOFEvent) {
        p.count.Add(1)
    }))
    p.AddSub(bus.Subscribe(rlevents.TypeMatchDestroyed, func(oofevents.OOFEvent) {
        p.count.Store(0)
    }))
    return nil
}

func (p *Plugin) Routes(mux *http.ServeMux) {
    mux.HandleFunc("/api/pings/count", func(w http.ResponseWriter, _ *http.Request) {
        httputil.WriteJSON(w, map[string]int64{"count": p.count.Load()})
    })
}

func (p *Plugin) Assets() fs.FS { return assetsFS }

func (p *Plugin) SettingsSchema() []plugin.Setting        { return nil }
func (p *Plugin) ApplySettings(map[string]string) error   { return nil }
```

### 2) `assets/view.html`

```html
<div class="text-center py-20">
  <div class="text-7xl font-extrabold tabular-nums" id="pings-count">0</div>
  <div class="text-sm text-gray-500 mt-2">ball hits this session</div>
</div>
```

### 3) `assets/view.js`

`pluginInit_<pluginID>` is called once when the plugin view is injected.

```js
'use strict';

window.pluginInit_pings = async function() {
  refreshPings();
};

async function refreshPings() {
  try {
    const data = await fetch('/api/pings/count').then(r => r.json());
    const el = document.getElementById('pings-count');
    if (el) el.textContent = data.count;
  } catch (_) {}
}
```

---

## Settings

Settings are declared in `SettingsSchema()` and applied in `ApplySettings()`.

```go
func (p *Plugin) SettingsSchema() []plugin.Setting {
    return []plugin.Setting{
        {
            Key:         "pings_threshold",
            Label:       "Alert threshold",
            Type:        plugin.SettingTypeNumber,
            Default:     "100",
            Description: "Show an alert banner after this many ball hits.",
        },
    }
}
```

Disabled plugins still appear in settings schema with `enabled=false` so users can re-enable them.

---

## Dependencies

Declare dependencies with `Requires()` using plugin IDs:

```go
func (p *Plugin) Requires() []string { return []string{"history"} }
```

Runtime policy is strict: if an enabled plugin requires a disabled plugin, startup fails with a dependency error.

---

## Quick Checklist

- Pick stable `PluginID` (`ID()`)
- Pick a `ViewID` (`NavTab().ID`) for navigation
- Export `window.pluginInit_<pluginID>` in `assets/view.js`
- Serve plugin APIs under your own namespace (for example `/api/<pluginID>/...`)
- Keep dependency IDs in `Requires()` aligned with plugin IDs
