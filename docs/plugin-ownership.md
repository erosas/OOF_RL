# Plugin Ownership and Trust Model

This document captures the host/plugin ownership matrix and the trust model for WASM plugins. It complements the [Architecture RFC](architecture-rfc.md) with a quick reference for contributors.

---

## Feature Ownership Matrix

| Feature | Owner | Notes |
|---------|-------|-------|
| Match history storage and queries | **Host** (`internal/histstore`) | `plugins/history` is a nav-tab shell only |
| Event bus fan-out and translation | **Host** (`internal/rlevents`, `internal/hub`) | Plugins subscribe; they do not produce events |
| RL WebSocket client (auto-reconnect) | **Host** (`internal/rl`) | |
| Config read/write | **Host** (`internal/config`, `internal/core`) | Plugins receive settings via `apply_settings` call |
| SQLite schema and migrations | **Host** (`internal/db`) | Plugins have no direct DB access |
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
| Debug event log and screenshots | **Plugin** (`plugins/debugassistant`) | |

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

The following host imports are available to all plugins via `plugins/sdk/pdk.go`:

| Import | Description |
|--------|-------------|
| `sdk.Log` | Emit a log line to the host log |
| `sdk.ReadConfig` | Read the current app config as JSON |
| `sdk.DBQuery` | Execute a read-only SQL query |
| `sdk.DBExec` | Execute a write SQL statement |
| `sdk.HTTPFetch` | Make an outbound HTTP request |
| `sdk.BroadcastEvent` | Push a WebSocket event to all browser clients |
| `sdk.ReadDataFile` | Read a file from `plugin_data/{id}/` |
| `sdk.WriteDataFile` | Write a file to `plugin_data/{id}/` |

All plugins have access to all of these. Capability scoping (restricting individual plugins to a subset) is not yet implemented.