# Writing a Plugin

Plugins are the primary extension point. Each plugin owns a nav tab, a set of HTTP routes, an optional database schema, and a view rendered in the browser. The session tracker, ballchasing integration, and match history are all plugins.

---

## The Plugin Interface

Every plugin implements `plugin.Plugin` (`internal/plugin/plugin.go`):

```go
type Plugin interface {
    ID() string                                   // unique identifier, e.g. "myplugin"
    DBPrefix() string                             // table name prefix, e.g. "mp" → tables named "mp_*"
    Requires() []string                           // IDs of plugins this depends on; nil = no deps
    NavTab() NavTab                               // tab label and sort order
    Routes(mux *http.ServeMux)                    // register HTTP handlers
    SettingsSchema() []Setting                    // settings fields shown in the Settings tab
    ApplySettings(values map[string]string) error // called when the user saves settings
    HandleEvent(env events.Envelope)              // called for every RL event (MatchCreated, UpdateState, …)
    Assets() fs.FS                               // embedded FS containing view.html and view.js
}
```

---

## Step-by-Step Example

This example builds a minimal "ping counter" plugin that counts how many times the ball has been hit in the current session.

### 1. Create the package

```
internal/plugins/pings/
    plugin.go
    view.html
    view.js
```

### 2. Write `plugin.go`

```go
package pings

import (
    "embed"
    "encoding/json"
    "io/fs"
    "net/http"
    "sync/atomic"

    "OOF_RL/internal/events"
    "OOF_RL/internal/httputil"
    "OOF_RL/internal/plugin"
)

//go:embed view.html view.js
var viewFS embed.FS

type Plugin struct {
    count atomic.Int64
}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string         { return "pings" }
func (p *Plugin) DBPrefix() string   { return "" }      // no database tables needed
func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) NavTab() plugin.NavTab {
    return plugin.NavTab{ID: "pings", Label: "Pings", Order: 50}
}

func (p *Plugin) Routes(mux *http.ServeMux) {
    mux.HandleFunc("/api/pings/count", func(w http.ResponseWriter, r *http.Request) {
        httputil.WriteJSON(w, map[string]int64{"count": p.count.Load()})
    })
}

func (p *Plugin) HandleEvent(env events.Envelope) {
    switch env.Event {
    case "BallHit":
        p.count.Add(1)
    case "MatchDestroyed":
        p.count.Store(0)
    }
}

func (p *Plugin) SettingsSchema() []plugin.Setting        { return nil }
func (p *Plugin) ApplySettings(_ map[string]string) error { return nil }
func (p *Plugin) Assets() fs.FS                           { return viewFS }
```

### 3. Write `view.html`

This fragment is injected into the page when the tab is shown.

```html
<div class="text-center py-20">
  <div class="text-7xl font-extrabold tabular-nums" id="pings-count">0</div>
  <div class="text-sm text-gray-500 mt-2">ball hits this session</div>
</div>
```

### 4. Write `view.js`

The function `pluginInit_<id>` is called once when the tab's HTML is first injected. The name must match the plugin ID with hyphens replaced by underscores.

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

To update in real time when a `BallHit` event arrives over WebSocket, add a handler in `web/app.js` — or call `refreshPings()` from the WS dispatch block inside the existing `ws.onmessage` handler.

### 5. Register in `main.go`

```go
import "OOF_RL/internal/plugins/pings"

// after the other srv.Use() calls:
srv.Use(pings.New())
```

That's it. The nav tab appears, `/api/pings/count` is live, and `HandleEvent` receives every RL event as it arrives.

---

## Database Tables

If your plugin needs persistent storage, declare a schema in `New()` and prefix every table name with your `DBPrefix()`:

```go
func New(database *db.DB) *Plugin {
    if err := database.RunMigration(`
        CREATE TABLE IF NOT EXISTS mp_sessions (
            id         INTEGER PRIMARY KEY AUTOINCREMENT,
            started_at DATETIME NOT NULL
        );
    `); err != nil {
        log.Printf("[myplugin] migrate: %v", err)
    }
    return &Plugin{db: database}
}
```

`RunMigration` executes the SQL verbatim. Run it once at startup; `CREATE TABLE IF NOT EXISTS` is safe to repeat. The prefix keeps table names unambiguous across plugins.

---

## Settings

Declare settings in `SettingsSchema()` and read them back in `ApplySettings()`:

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

func (p *Plugin) ApplySettings(values map[string]string) error {
    if v, ok := values["pings_threshold"]; ok {
        if n, err := strconv.Atoi(v); err == nil {
            p.threshold.Store(int64(n))
        }
    }
    return nil
}
```

Setting types: `SettingTypeText`, `SettingTypePassword`, `SettingTypeNumber`, `SettingTypeCheckbox`.

---

## Plugin Dependencies

If your plugin reads data written by another plugin (e.g. the `hist_*` tables from the history plugin), declare that dependency so the Settings UI can warn the user if the required plugin is disabled:

```go
func (p *Plugin) Requires() []string { return []string{"history"} }
```

This is advisory — the runtime does not enforce load order. Make sure the dependency is registered with `srv.Use()` before your plugin if you need its DB tables to exist first.

---

## RL Events Reference

These are the events OOF RL receives from Rocket League (defined in `internal/events/`):

| Event | When it fires |
|-------|---------------|
| `MatchCreated` | A new match GUID is assigned |
| `MatchInitialized` | Match is fully ready |
| `UpdateState` | Every game tick (~60 Hz) — players, ball, scores, boost |
| `GoalScored` | A goal is scored |
| `BallHit` | A player touches the ball |
| `StatfeedEvent` | Award/accolade feed entry |
| `MatchEnded` | Winning team determined |
| `MatchDestroyed` | Match fully torn down (post-game screen dismissed) |

The `Data` field of each envelope is already unwrapped from its double-encoded form — you can `json.Unmarshal` it directly into the corresponding struct in `internal/events/`.

---

## WebSocket Push

To push data to the browser from `HandleEvent`, inject it into the WebSocket hub:

```go
type Plugin struct {
    hub *hub.Hub
    // ...
}

func (p *Plugin) HandleEvent(env events.Envelope) {
    if env.Event == "BallHit" {
        evt, _ := json.Marshal(map[string]any{
            "Event": "pings:hit",
            "Data":  map[string]any{"count": p.count.Add(1)},
        })
        p.hub.Broadcast(evt)
    }
}
```

In `view.js`, listen for it in the global WS `onmessage` handler (in `web/app.js`) by adding a `case "pings:hit":` branch, or use a `window.handlePingsHit` convention that `app.js` already dispatches for you if you add it to the dispatch table.
