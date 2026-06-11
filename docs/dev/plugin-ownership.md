# Plugin Ownership and Trust Model

This document captures the host/plugin ownership matrix and the trust model for WASM plugins.

---

## Feature Ownership Matrix

| Feature | Owner | Notes |
|---------|-------|-------|
| Match history storage and queries | **Host** (`internal/histstore`) | `plugins/history` is a nav-tab shell only |
| Event bus fan-out and translation | **Host** (`internal/rlevents`, `internal/hub`) | Plugins subscribe; they do not produce events |
| RL WebSocket client (auto-reconnect) | **Host** (`internal/rl`) | |
| Config read/write | **Host** (`internal/config`, `internal/core`) | Plugins receive settings via `apply_settings` call |
| SQLite schema and migrations | **Host** (`internal/db`) | Plugins run SQL via `host_db_query`/`host_db_exec` but own no schema or migrations |
| HTTP server, routing, middleware | **Host** (`internal/core`) | Plugin routes registered by host at startup |
| WebSocket broadcast to browser | **Host** (`internal/hub`) | |
| Static web shell (`web/`) | **Host** | Plugin views injected dynamically |
| Public file serving for plugins | **Host** (`/api/plugins/{id}/data/...`) | Backed by `plugin_data/{id}/public/` |
| Overlay window and hotkeys | **Host** (`overlay_windows.go`) | |
| Live scoreboard display | **Plugin** (`plugins/live`) | |
| Player rank lookups | **Plugin** (`plugins/ranks`) | |
| Session statistics | **Plugin** (`plugins/session`) | |
| Ballchasing.com upload | **Plugin** (`plugins/ballchasing`) | |
| Summary dashboard | **Plugin** (`plugins/dashboard`) | |

### Host-core features (cannot be disabled)

- `history` — data storage and history tab are always enabled regardless of config
- HTTP server, event bus, RL client — foundational; not plugin-controlled

---

## WASM Plugin Trust Model

**Current model: trusted extensions.**

All WASM plugins shipped with OOF RL (`plugins/`) are authored by the same team that maintains the host runtime. They are not sandboxed from each other or from host resources in the way an untrusted-plugin model would require.

### What this means in practice

- Plugins are loaded from `%LOCALAPPDATA%\OOF_RL\plugins` at startup.
- The WASM sandbox enforces memory isolation between plugin and host; neither can read the other's memory directly without going through the host ABI.
- However, plugins are *not* restricted by capability policy: any loaded plugin can call all available host imports (log, config-read, DB-read, DB-write, HTTP fetch, WS broadcast, etc.).
- No plugin signature verification is performed at load time.

### Implications

- Untrusted third-party WASM plugins loaded from `plugins/` would have full access to the local database, config, and network via host imports.
- If third-party plugin distribution becomes a goal, the trust model must be revisited before exposing install-from-URL flows.

### Non-goals for the current model

- Capability scoping (restricting which host imports a given plugin may call)
- Plugin signatures or publisher verification
- Sandboxed network or filesystem access beyond the WASM linear-memory boundary

---

## SDK Capability Surface

The following host imports are available to all plugins via `sdk/pdk.go` (`github.com/erosas/oof-plugin-sdk`):

| Function | Description |
|--------|-------------|
| `sdk.Log` | Emit a log line to the host log |
| `sdk.GetConfig` | Read a config value by key |
| `sdk.DBQuery` | Execute a read-only SQL query |
| `sdk.DBExec` | Execute a write SQL statement |
| `sdk.HTTPFetch` | Make an outbound HTTP request |
| `sdk.BroadcastWS` | Send a WebSocket message to all browser clients |
| `sdk.PublishEvent` | Publish an event onto the host event bus |
| `sdk.UploadFile` | Stream a WASI-mounted file to a URL via multipart POST (host reads from disk) |

File access uses normal Go file I/O against WASI-mounted paths (`/replays`, `/data`). All plugins have access to all imports. Capability scoping is not yet implemented.