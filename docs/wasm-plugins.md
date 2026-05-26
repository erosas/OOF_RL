# WASM Plugin System

Plugins can be compiled to `.wasm` and dropped into `%LOCALAPPDATA%\OOF_RL\plugins\` without recompiling the host. The host loads them at startup and treats them identically to built-in Go plugins — same nav tab, same routing, same event bus.

## Identity contract

WASM plugins use the same identity rules as native plugins:

- `PluginID` is canonical runtime/API/assets identity (`PluginMeta.ID`).
- `ViewID` is frontend navigation identity only (`PluginMeta.NavTab.ID`).

Use them like this:

- View HTML endpoint: `/api/plugins/{pluginID}/view`
- View script endpoint: `/plugins/{pluginID}/view.js`
- Frontend init hook: `window.pluginInit_<pluginID>()`
- Nav/showView state: `view_id`

There are no compatibility aliases for ViewID-based backend loading.

## Building a plugin

```sh
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o live.wasm .
```

`-buildmode=c-shared` produces a WASI *reactor* module. Without it you get a command module: `_initialize` only partially starts the Go runtime, and every exported call fails with `runtime.notInitialized`.

The Makefile target `make wasm/<name>` handles the flags and copies assets:

```sh
make wasm/live
```

## File layout

```
%LOCALAPPDATA%\OOF_RL\plugins\
  live.wasm        # plugin binary
  live\            # assets dir, named after the plugin ID (optional)
    view.html
    view.js
```

The host serves everything under `live\` at `/plugins/live/` where `live` is the plugin ID.

## Lifecycle

```
host                             plugin
────                             ──────
load .wasm
  _initialize              →     Go runtime + init()
  plugin_metadata()        →     returns PluginMeta JSON
  [wire routes, assets, event subscriptions from metadata]

InitPlugins  (sorted by Requires)
  plugin_init(cfgJSON)     →     one-time setup, returns 0 or error code

runtime
  plugin_on_event()        →     receive event from bus
  host_publish_event()     ←     push event onto bus
  plugin_handle_http()     →     handle a declared route

shutdown
  plugin_shutdown()        →     cleanup
```

`InitPlugins` runs a topological sort before the loop, so if plugin B declares `Requires: ["a"]`, plugin A is always initialized first. A cycle or unknown plugin ID is a startup error.

If an enabled plugin requires a disabled plugin, startup also fails with a dependency error.

Disabled plugins are runtime-inactive for init/routes/assets, but still appear in settings schema with `enabled=false`.

## ABI

All parameters are `uint32` — pointers or byte lengths in the module's linear memory. The host always allocates guest memory by calling the plugin's exported `malloc`; the plugin never allocates host memory.

### Exports (host → plugin)

| Export | Signature | Notes |
|---|---|---|
| `plugin_metadata` | `(outPtr, outMax u32) → n u32` | Write JSON-encoded `PluginMeta` into `outPtr`; return byte count |
| `plugin_init` | `(cfgPtr, cfgLen u32) → errCode u32` | Called once after metadata is read; return 0 for success |
| `plugin_on_event` | `(typePtr, typeLen, payloadPtr, payloadLen u32)` | JSON-marshalled `OOFEvent` delivered by the bus |
| `plugin_handle_http` | `(reqPtr, reqLen, outPtr, outMax u32) → n u32` | JSON `HTTPRequest` in, JSON `HTTPResponse` out |
| `plugin_shutdown` | `()` | Cleanup before unload |
| `malloc` | `(size u32) → ptr u32` | Host calls this to allocate guest memory |
| `free` | `(ptr, size u32)` | Host calls this to release guest memory |

`HTTPResponse` is intended for JSON/text payloads. For binary file delivery (for example images), write files under `/data/public/...` and let the host serve them from `/api/plugins/{pluginID}/data/{path...}`.

### Imports (plugin → host)

| Import | Signature | Notes |
|---|---|---|
| `env.host_log` | `(level, ptr, len u32)` | Write to the host's logger |
| `env.host_publish_event` | `(certainty, typePtr, typeLen, payloadPtr, payloadLen u32)` | Publish onto the event bus |

## Host mounts and sandbox paths

At load time the host mounts two directories into the plugin sandbox:

- `/replays` -> configured replay directory
- `/data` -> `<data_dir>/plugin_data/<pluginID>/`

Use normal file APIs in guest code against those mount points.

## Memory model

The host calls `malloc` to allocate a buffer in the plugin's linear memory, writes data into it, calls an exported function with the pointer, then calls `free`. The plugin's `malloc`/`free` shims keep a GC-protection map so the Go GC doesn't collect slices whose raw pointers have been handed out.

Inside exported functions, `sdk.ReadBytes(ptr, len)` returns a slice backed directly by linear memory. Don't retain it past the function call — the host frees the allocation immediately after return.

## Events

**Receiving:** the host marshals each `OOFEvent` to JSON and calls `plugin_on_event`. The plugin declares which event types it wants in `PluginMeta.Events`.

**Publishing:** call `sdk.PublishEvent(certainty, eventType, payloadJSON)`. The host wraps it as a `oofevents.RawEvent` on the bus. Any native or WASM plugin can subscribe to that event type string.

Use a namespaced type string (e.g. `live.state.changed`) to avoid colliding with native event types defined in `internal/oofevents/eventtypes.go`.

## Metadata guidance

`PluginMeta` fields are interpreted as follows:

- `id` (`PluginID`): canonical runtime identity; used for init, view loading, assets, and plugin data paths.
- `nav_tab.id` (`ViewID`): UI navigation slug only.
- `routes`: plugin-owned HTTP routes handled via `plugin_handle_http`, declared as `{ "path": "...", "method": "GET|POST|..." }`. `method` is optional when a route supports multiple methods.
- `requires`: dependency plugin IDs for startup ordering/validation.
- `declared_events`: optional event declarations for plugin-emitted event types; types must be non-empty/unique and certainty must be valid.
- `settings`: plugin settings metadata surfaced in settings schema (`key`, `label`, `type`, `default`, `options`, `placeholder`, `developer`, `description`).

Single-page plugin model: one `view.html` + one `view.js` per plugin.

`DBPrefix` exists as a compatibility surface in the broader plugin API and is not required for typical WASM plugin behavior.

## SDK (`plugins/sdk/`)

| File | Build target | Contents |
|---|---|---|
| `abi.go` | both | `PluginMeta`, `HTTPRequest`, `HTTPResponse`, `Certainty`, `DeclaredEvent` |
| `pdk.go` | wasip1 only | `Log`, `ReadBytes`, `WriteOutput`, `PublishEvent`, `Malloc`, `Free` |

The host imports `abi.go` types to drive the protocol. Plugin code imports the whole package for both.