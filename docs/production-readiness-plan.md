# Production Readiness Plan

This document is the working source of truth for making `OOF_RL` production-ready.

It is designed to be updated as work is completed. Every phase includes:
- goals
- concrete tasks
- acceptance criteria
- current status
- known risks

---

## Status Legend

- [ ] Not started
- [~] In progress
- [x] Done
- [!] Blocked / decision required

---

## Current Snapshot (verified on 2026-05-27)

### Verified code health
- [x] Host tests pass with `go test ./...`
- [x] Plugin tests pass for all plugins via `make test-plugins`
- [x] `make test-all` passes end to end
- [x] `plugins/ballchasing` has tests covering `normalizeGUID`, `applySettings`, `matchReplayFiles`
- [x] `plugins/history` reclassified as host-core nav-tab shell; functionality tested via `internal/histstore`

### All original review findings resolved
- [x] Disabled plugins are runtime-inactive for init/routes/assets
- [x] Plugin identity is canonical: `PluginID` for runtime/API/assets, `ViewID` for frontend nav
- [x] Host/plugin ownership is explicit and documented
- [x] `HTTPResponse.BodyBytes` removed from response path; host serves plugin public files directly
- [x] WASM plugin boilerplate reduced via SDK helpers (`WriteMetadata`, `HandleHTTPExport`, `JSONError`, `JSONResponse`, `ParseBool`, `ParseTime`, `QueryParam`)
- [x] Docs updated to match current architecture
- [x] Local server bound to `127.0.0.1`, timeouts set, origin checks enforced, pprof gated

---

## North Star

The target state is:

1. **Clear ownership boundaries**
   - host-owned features are explicitly host-owned
   - plugin-owned features are explicitly plugin-owned
   - disable semantics are consistent across runtime, routes, UI, assets, and settings

2. **Clean plugin API**
   - canonical plugin identity is stable
   - route definitions are explicit
   - binary/public file delivery is host-managed where appropriate
   - SDK helpers eliminate repetitive plugin boilerplate

3. **Production-grade runtime**
   - local server is hardened
   - plugin runtime behavior is predictable
   - diagnostics are dev-gated
   - tests cover critical behavior and regressions

4. **Operational clarity**
   - docs match code
   - build/test flows are consistent
   - releases and plugin packaging are repeatable

---

## Guiding Decisions We Need to Lock Early

### Decision 1: Canonical plugin identity
- [x] Decide whether `Plugin.ID()` is the only canonical runtime identifier
- [x] Demote current `NavTab.ID` into a view slug / tab slug / frontend-only ID if needed
- [~] Ensure APIs, asset paths, plugin data paths, settings, and logs all use the same canonical ID

**Recommended direction**
- Canonical runtime identifier: `PluginID`
- Optional UI identifier: `ViewID` or `TabID`

**Phase 0 decision (locked)**
- Use `PluginID` as canonical runtime/data/API identity.
- Keep `ViewID` as a separate frontend navigation slug.

### Decision 2: What “disabled plugin” means
- [x] Decide whether disabled means **hidden in UI only** or **inactive at runtime**
- [x] Document exact effects on:
  - plugin initialization
  - routes
  - assets
  - event subscriptions
  - settings schema
  - frontend injection

**Recommended direction**
- Disabled should mean **inactive at runtime** unless the feature is explicitly host-core.

**Phase 0 decision (locked)**
- Disabled plugins are runtime-inactive for init/routes/assets.
- Disabled plugins remain visible in settings so they can be re-enabled/configured.

### Decision 3: History ownership model
- [x] Decide whether `history` remains a host-owned feature with plugin-like UI assets
- [x] Or convert `history` into a truly plugin-owned feature

**Recommended direction**
- Keep `history` as a host-owned foundational feature and stop pretending it is fully plugin-owned.

**Phase 0 decision (locked)**
- `history` is treated as host-core and documented as such.

### Decision 4: Response-side binary/file serving
- [x] Decide whether to deprecate `sdk.HTTPResponse.BodyBytes`
- [x] Decide the public file-serving model for plugins

**Recommended direction**
- Keep request-side `BodyBytes` for outbound uploads
- Deprecate response-side `BodyBytes`
- Add a host-owned public data route serving only plugin public files, not arbitrary plugin data

**Phase 0 decision (locked)**
- Implement host route `/api/plugins/{pluginID}/data/{path...}` mapped to `plugin_data/{pluginID}/public/...`.
- Do not expose arbitrary plugin data paths.

---

## Proposed API Direction

### Recommended public file route

**Problem today**
- `sdk.HTTPResponse.BodyBytes` forces binary payloads through WASM memory and JSON encoding/decoding
- that is awkward, wasteful, and hard to scale cleanly

**Recommended design**
- host route: `/api/plugins/{pluginID}/data/{path...}`
- filesystem backing: `<data_dir>/plugin_data/{pluginID}/public/{path...}`
- host performs path validation and traversal protection
- plugins write files under `/data/public/...`
- host serves them directly

**Important constraint**
- Do **not** expose the full `plugin_data/<pluginID>/...` tree
- Only expose a declared public subtree or allowlisted paths

### Migration path
- [x] Add host public data route
- [x] Migrate `debugassistant` screenshot serving first
- [x] Keep `HTTPResponse.BodyBytes` temporarily for compatibility (removed — no compatibility window needed)
- [x] Remove/deprecate response-side `BodyBytes` after migration (removed from SDK and host response writer)

---

## Phase 0 — Architecture RFC and Inventory

**Status:** [x] Done

### Goals
- freeze the platform model before broad refactors
- make the current architecture explicit
- document where host/plugin ownership is mixed

### Tasks
- [x] Write architecture RFC for plugin identity, disable semantics, and host/plugin boundaries
- [x] Inventory all APIs under `internal/core/server.go` (covered by `docs/api.md`)
- [x] Inventory all WASM plugin exports/imports under `internal/wasmhost/host.go` and `plugins/sdk/*` (covered by `docs/wasm-plugins.md` and `docs/plugin-ownership.md`)
- [x] Inventory current plugin routes, assets, settings, and event subscriptions (covered by ownership matrix)
- [x] Mark each feature as one of:
  - host-owned
  - plugin-owned
  - hybrid (must be cleaned up)
- [x] Inventory all places where `Plugin.ID()` and `NavTab.ID` are used interchangeably (single-pass cutover completed)
- [x] Inventory all code and docs that refer to plugins using ambiguous identity terms like `id` (done; `PluginID`/`ViewID` terminology enforced)

### Deliverables
- [x] Architecture RFC document (`docs/architecture-rfc.md`)
- [x] Ownership matrix (`docs/plugin-ownership.md`)
- [x] Plugin capability matrix (`docs/plugin-ownership.md` — SDK capability surface section)
- [x] API inventory (`docs/api.md`)

### Acceptance criteria
- [x] Team can answer “what is a plugin?” in one paragraph (see `docs/plugins.md`)
- [x] Team can answer “what does disabled mean?” without ambiguity (see architecture RFC §2 and `docs/plugins.md`)
- [x] Team can answer whether `history` is host-owned or plugin-owned (see `docs/plugin-ownership.md` and architecture RFC §3)

---

## Phase 1 — Fix Runtime Semantics and Plugin Identity

**Status:** [x] Done

### Goals
- make runtime behavior match the product model
- stop ID confusion from spreading

### Tasks

#### 1.1 Canonicalize identity
- [x] Define canonical `PluginID`
- [x] Rename or conceptually separate nav/view slug from plugin runtime ID
- [x] Update backend routes to consistently use canonical plugin ID
- [x] Update frontend loading in `web/app.js` to use canonical plugin ID for plugin view/script/data loading
- [x] Review and update asset route prefixes under `internal/core/server.go`

#### 1.2 Make disabled plugins truly inactive
- [x] Create one filtered active-plugin set in the server lifecycle
- [x] Keep lifecycle helpers server-private in `internal/core/server.go` for Phase 1, then evaluate extraction to shared helpers after semantics stabilize
- [x] Enforce dependency policy: enabled plugins that require disabled plugins fail startup with a clear error
- [ ] Ensure inactive plugins are excluded from:
  - [x] `InitPlugins()`
  - [x] route registration
  - [x] asset registration
  - [x] nav output
  - [x] settings schema (intentionally deferred; disabled plugins remain listed with `enabled=false` — this is the desired behavior)
  - [x] frontend view injection
- [x] Add tests for disabled plugin behavior

#### 1.3 Resolve `history` ownership
- [x] Decide and document whether `history` is host-core
- [x] If host-core, stop modeling it like a normal WASM plugin
- [x] Update routes/settings/nav behavior to reflect the decision
- [x] Update docs accordingly

### Target files
- `internal/core/server.go`
- `internal/plugin/plugin.go`
- `internal/histstore/*`
- `plugins/history/main.go`
- `web/app.js`
- `docs/plugins.md`
- `docs/api.md`

### Acceptance criteria
- [x] Disabling a plugin prevents it from initializing and serving routes/assets
- [x] Plugin identity is unambiguous in code and docs
- [x] `history` no longer behaves like a hybrid by accident

---

## Phase 2 — Simplify Plugin API and Remove Awkward Boundaries

**Status:** [x] Done

### Goals
- reduce API confusion
- improve performance and maintainability
- remove brittle ABI patterns

### Tasks

#### 2.1 Public file route
- [x] Add host-level plugin public data route
- [x] Restrict serving to public subtrees / allowlisted files
- [x] Add traversal/path normalization protections
- [x] Add tests for valid, invalid, and traversal requests

#### 2.2 Deprecate response-side binary transport
- [x] Migrate `debugassistant` screenshots to public route serving
- [x] Audit all `BodyBytes` response usage
- [x] Leave request-side `HTTPFetchRequest.BodyBytes` intact for outbound uploads such as `ballchasing`
- [x] Deprecate `HTTPResponse.BodyBytes` in SDK and docs

#### 2.3 Evolve route metadata
- [x] Replace `PluginMeta.Routes []string` with richer route definitions
- [x] Include at least method information in route metadata
- [x] Review whether content type / route kind should also be explicit (decided: method explicit via `RouteMeta`; content type is set by plugin response headers)

#### 2.4 Unify settings schema model
- [x] Align WASM plugin settings metadata with native `plugin.Setting`
- [x] Support consistent labels, defaults, placeholders, and field types
- [x] Remove drift between host and WASM plugin settings capabilities

#### 2.5 Review/remove weak API surface
- [x] Review whether `DBPrefix()` should be removed
- [x] Review whether `DeclaredEvents()` should be removed or made first-class
- [x] Review whether plugin route declarations should be validated more strictly on load

### Target files
- `internal/wasmhost/host.go`
- `plugins/sdk/abi.go`
- `plugins/sdk/helpers.go`
- `plugins/debugassistant/logic.go`
- `plugins/ballchasing/logic.go`
- `docs/wasm-plugins.md`
- `docs/api.md`

### Acceptance criteria
- [x] Public plugin files are served without binary-through-WASM response transport
- [x] SDK route/settings metadata is clearer and less error-prone
- [x] Plugins no longer need response-side binary transport for normal file serving

---

## Phase 3 — Remove Duplicate Logic and Boilerplate

**Status:** [x] Done

### Goals
- reduce repetition across plugins and host code
- make plugin implementation style consistent

### Tasks

#### 3.1 Consolidate WASM plugin export boilerplate
- [x] Inventory duplicate `main.go` export patterns across plugins
- [ ] Create shared helpers/patterns in `plugins/sdk` for:
  - [x] metadata write
  - [x] HTTP request decode / response encode
  - [x] common error responses (`sdk.JSONError`, `sdk.JSONResponse` in `plugins/sdk/helpers.go`)
  - [x] event dispatch wrappers (decided: not abstracted — thin WASM ABI makes per-plugin switch statements idiomatic)
  - [x] init/apply settings wrappers where feasible (decided: init/settings logic varies too much per plugin to abstract usefully)
- [x] Reduce repeated `malloc`/`free`/`plugin_handle_http` patterns across plugins

#### 3.2 Consolidate plugin helper duplication
- [x] Replace local `jsonOK` / `jsonError` duplicates with `plugins/sdk/helpers.go` (`dashboard`, `session`, `live`)
- [x] Create shared query-param helper (`sdk.QueryParam`) and migrate `session`
- [x] Create shared bool parsing helper (`sdk.ParseBool`) and migrate `ballchasing`
- [x] Create shared time parsing helper (`sdk.ParseTime`) and migrate `session` and `ballchasing`

#### 3.3 Consolidate event payload handling
- [x] Identify repeated event payload DTOs like `state.updated`
- [x] Decide whether to add lightweight shared SDK event DTOs for common payloads (decided: intentionally local — each plugin defines its own projection of the event payload; shapes differ meaningfully across plugins)
- [x] Remove duplicated payload-shape structs where practical (no removal needed; local projections are intentional)

#### 3.4 Consolidate host HTTP utilities
- [x] Standardize JSON error responses in host handlers
- [x] Replace mixed `http.Error`/JSON behavior where APIs are intended to be JSON (all `http.Error` calls in `server.go` now use `httputil.JSONError`)
- [x] Review shared handler helpers under `internal/httputil`

### Known duplication hotspots already identified
- [x] WASM plugin `main.go` exports (addressed by `sdk.WriteMetadata`, `sdk.HandleHTTPExport` helpers)
- [x] plugin JSON response helpers
- [x] ad-hoc `parseTime` helpers
- [x] query parsing helpers
- [x] raw string-based method/path dispatch patterns (handled by `sdk.HandleHTTPExport` routing)
- [x] repeated boolean parsing (`true` / `1` / `on`)

### Target files
- `plugins/*/main.go`
- `plugins/*/logic.go`
- `plugins/sdk/*`
- `internal/httputil/httputil.go`
- `internal/core/server.go`

### Acceptance criteria
- [x] New plugins can be created with substantially less boilerplate
- [x] Common helpers live in one place
- [x] Duplicate logic is intentionally shared or intentionally kept for a documented reason

---

## Phase 4 — Security, Reliability, and Runtime Hardening

**Status:** [x] Done

### Goals
- make the local server safer and more predictable
- reduce operational risk from plugins and sockets

### Tasks

#### 4.1 Harden local server defaults
- [x] Bind explicitly to `127.0.0.1` by default unless configuration says otherwise
- [x] Add `http.Server` timeouts:
  - [x] `ReadHeaderTimeout` (10s)
  - [ ] `ReadTimeout` (intentionally skipped — large replay uploads via ballchasing would be clipped)
  - [x] `WriteTimeout` (30s)
  - [x] `IdleTimeout` (120s)
- [x] Gate pprof/statsviz behind `DevMode`
- [x] Review whether app port fallback/binding behavior should be more explicit (decided: current behavior is explicit — binds to `127.0.0.1:port` or logs an error)

#### 4.2 Harden WebSocket handling
- [x] Replace `CheckOrigin: return true` with explicit localhost/app-origin checks
- [x] Rework `internal/hub/hub.go` so one slow client cannot block all clients
- [x] Add tests for unregistering dead clients and blocking client behavior

#### 4.3 Harden WASM/plugin host boundary
- [x] Validate duplicate plugin IDs on load (`LoadPlugins` returns error; `LoadWASMPlugins` logs and skips)
- [x] Validate route conflicts on load (pre-check via `RoutePaths()` before any mux registration; conflicting plugin routes are skipped with a clear log)
- [x] Improve diagnostics for oversized host/plugin message buffers (decided: current logging around plugin failures is sufficient; buffer overflows are rare and logged clearly)
- [x] Review whether outbound HTTP, DB access, config access, and WS broadcast should be capability-scoped (decided: not in current model — see `docs/plugin-ownership.md` trust model)
- [x] Add clear logging around plugin init/apply-settings failures and route conflicts

#### 4.4 Review trust model
- [x] Decide whether external WASM plugins are trusted extensions or semi-trusted sandboxed code (decided: trusted extensions — see `docs/plugin-ownership.md`)
- [x] Align SDK surface with that trust model (documented; capability scoping is a non-goal for the current model)

### Target files
- `main.go`
- `internal/core/server.go`
- `internal/hub/hub.go`
- `internal/wasmhost/host.go`
- `plugins/sdk/pdk.go`

### Acceptance criteria
- [x] Local HTTP/WS endpoints are limited to intended local use by default
- [x] One slow or broken socket client does not degrade all clients
- [x] Plugin load/runtime failures are diagnosable and bounded

---

## Phase 5 — Tests, CI, Packaging, and Release Quality

**Status:** [x] Done

### Goals
- make quality visible and repeatable
- ensure every plugin is testable and packaged consistently

### Tasks

#### 5.1 Fix current test workflow issues
- [x] Make `plugins/debugassistant` testable in the same way as other plugins
- [x] Make `make test-plugins` pass end to end
- [x] Decide whether every plugin must support native test builds via `stub_main.go` + module wiring (yes — all shipped plugins now have `stub_main.go` and a `go.mod`)

#### 5.2 Raise plugin test coverage
- [x] Add tests for `plugins/ballchasing` (`normalizeGUID`, `matchReplayFiles`, `applySettings`)
- [x] `plugins/history` reclassified: plugin is a nav-tab shell only; all history functionality is tested via `internal/histstore`
- [x] Add tests for plugin public file route behavior (serves file, disabled plugin 404, unknown plugin 404, missing file 404, traversal 400)
- [x] Add tests for disabled plugin behavior (init skip, route/asset skip, nav exclusion, view 404)
- [x] Add tests for route conflict detection and plugin ID conflict detection

#### 5.3 Review workspace/module consistency
- [x] Normalize plugin module strategy (each plugin has its own `go.mod` under `plugins/<name>/`)
- [x] Ensure `go.work`, plugin `go.mod` files, and `Makefile` stay in sync (all 7 plugins in `go.work`; `debugassistant` `go.mod` added)
- [x] Decide whether all distributed plugins must be in `go.work` (yes — all plugins in `go.work`)

#### 5.4 Review release/build hygiene
- [x] Verify `Makefile` targets reflect current plugins and testability (`PLUGINS` list matches all 7 plugins; `test-all` covers host + SDK + plugins)
- [x] Ensure assets are copied consistently for all distributed WASM plugins (`make wasm/%` copies `assets/` for any plugin that has one)
- [x] Review whether generated binaries in the repo should be removed from versioned workspace state (no `.wasm` files committed; `.gitignore` excludes compiled artifacts)
- [x] Review `.gitignore` for release/build artifact completeness (`.exe`, `.db`, `coverage.*`, `config.toml`, `*.log`, `captures/` all excluded)

### Acceptance criteria
- [x] `make test-all` passes consistently
- [x] Every shipped plugin has a clear test strategy
- [x] Packaging and workspace layout are predictable

---

## Phase 6 — Documentation and Developer Experience

**Status:** [x] Done

### Goals
- make the architecture understandable to future contributors
- remove code/doc mismatches

### Tasks
- [x] Rewrite `docs/plugins.md` to match the current event bus + plugin model
- [x] Update `docs/wasm-plugins.md` to match actual ABI and host imports
- [x] Update `docs/api.md` to reflect canonical plugin IDs and public data routes
- [x] Update `README.md` prerequisites and plugin/system descriptions (Go 1.26+, WASM plugin architecture section added)
- [x] Add a short “how to build/test a plugin” guide that matches current tooling (added Testing section to `docs/wasm-plugins.md`)
- [x] Add a “host-owned vs plugin-owned features” reference (`docs/plugin-ownership.md`)

### Acceptance criteria
- [x] A new contributor can understand the plugin model without reading half the codebase
- [x] README/docs version/tooling requirements match the actual repo
- [x] Docs explain the chosen public file/data route clearly

---

## Workstream Backlog by Area

### Host runtime
- [ ] `main.go` hardening
- [ ] `internal/core/server.go` plugin lifecycle cleanup
- [ ] `internal/hub/hub.go` socket broadcast redesign
- [ ] `internal/httputil/httputil.go` JSON response standardization
- [ ] `internal/wasmhost/host.go` ABI simplification and route support

### SDK
- [ ] `plugins/sdk/abi.go` API cleanup
- [ ] `plugins/sdk/helpers.go` helper expansion
- [ ] `plugins/sdk/pdk.go` host-call ergonomics and capability review

### Plugins
- [ ] `plugins/live` boilerplate cleanup
- [ ] `plugins/ranks` boilerplate cleanup
- [ ] `plugins/session` helper dedupe and route cleanup
- [ ] `plugins/ballchasing` tests + helper dedupe
- [ ] `plugins/dashboard` helper dedupe
- [ ] `plugins/debugassistant` public file route migration + native testability
- [ ] `plugins/history` ownership cleanup

### Docs / DX
- [ ] `docs/plugins.md`
- [ ] `docs/wasm-plugins.md`
- [ ] `docs/api.md`
- [ ] `README.md`

---

## Known Risks / Watch List

- [ ] Breaking plugin compatibility while evolving `PluginMeta`
- [ ] Accidentally exposing private plugin data when introducing a public file route
- [ ] Keeping `history` in a hybrid state too long
- [ ] Refactoring identity semantics without updating frontend consumers
- [ ] Fixing duplicate logic before freezing architecture, causing churn
- [ ] Over-hardening local-only behavior in ways that make development painful

---

## Proposed Order of Execution

### Wave 1 — Freeze the model
- [ ] Architecture RFC
- [ ] Plugin identity decision
- [ ] Disabled-plugin semantics decision
- [ ] `history` ownership decision
- [ ] Public file-route decision

### Wave 2 — Fix correctness first
- [ ] Implement true active/inactive plugin filtering
- [ ] Clean up plugin identity usage
- [ ] Fix `debugassistant` test/build path
- [ ] Add missing runtime tests around plugin lifecycle

### Wave 3 — Simplify APIs
- [ ] Add host public file route
- [ ] Migrate `debugassistant`
- [ ] Introduce richer route metadata
- [ ] Unify settings metadata

### Wave 4 — Dedupe and harden
- [ ] SDK helper consolidation
- [ ] plugin boilerplate reduction
- [ ] server/socket hardening
- [ ] plugin boundary hardening

### Wave 5 — Final production polish
- [ ] docs sync
- [ ] release/build cleanup
- [ ] final regression sweep

---

## Completion Gates

Before calling the platform production-ready, all of the following should be true:

### Architecture
- [x] Plugin identity is canonical and documented
- [x] Disabled plugin semantics are enforced consistently
- [x] Host/plugin ownership is explicit

### API
- [x] Public plugin file serving is host-owned and safe
- [x] Response-side `BodyBytes` is removed or clearly deprecated
- [x] Plugin route and settings metadata are explicit and validated

### Reliability / Security
- [x] Local server is hardened by default
- [x] WebSocket broadcasting is resilient
- [x] Dev-only diagnostics are not exposed in normal mode

### Quality
- [x] `make test-all` passes
- [x] Shipped plugins have tests or documented justification
- [x] Docs match runtime behavior

---

## Progress Log

### 2026-05-25
- [x] Deep code review completed across host/runtime/plugins/docs
- [x] Initial production-readiness findings documented
- [x] This progress-tracking plan created
- [x] Verified that host tests pass
- [x] Verified that `make test-plugins` currently fails on `debugassistant`

### 2026-05-26 (continued, fourth pass)
- [x] Added `sdk.ParseBool` helper; migrated `ballchasing` off inline string compare; added `TestParseBool`, `TestQueryParam`, `TestParseTime` to `plugins/sdk/abi_test.go`
- [x] Added tests for route conflict detection (`TestRegisterSkipsPluginWithConflictingCoreRoute`, `TestRegisterSkipsPluginWithConflictingPluginRoute`) and duplicate plugin ID (`TestLoadPluginsDuplicateIDReturnsError`)
- [x] Added `RoutePaths() []string` override on `testPlugin` to enable declarative route conflict testing
- [x] Split oversized source files: `internal/core/server.go` (773→392 lines) + `handlers.go` (333), `internal/wasmhost/host.go` (835→530 lines) + `imports.go` (237), `web/app.js` (896→764 lines) + `arena-names.js` (133)
- [x] Split oversized test files: `internal/core/server_test.go` (878→458) + `handlers_test.go` (330) + `plugin_handlers_test.go` (239), `internal/wasmhost/worker_test.go` (718→480) + `imports_test.go` (249), `internal/rlevents/translator_test.go` (610→420) + `translator_gameactions_test.go` (198)

### 2026-05-27 (continued, third pass)
- [x] Added `sdk.ParseTime` and `sdk.QueryParam` to `plugins/sdk/helpers.go`
- [x] Removed local `jsonOK`/`jsonError` duplicates from `dashboard`, `session`, `live`; replaced with `sdk.JSONResponse`/`sdk.JSONError`
- [x] Removed local `parseTime` from `session` and `ballchasing`; replaced with `sdk.ParseTime`
- [x] Removed local `queryParam` from `session`; replaced with `sdk.QueryParam`
- [x] Updated `session` tests to use SDK functions
- [x] Added `plugins/ballchasing/logic_test.go` covering `normalizeGUID`, `applySettings`, and `matchReplayFiles` (5 sub-cases)
- [x] Reclassified `plugins/history` test item: plugin is a nav-tab shell; histstore tests cover the actual functionality

### 2026-05-27 (continued, second pass)
- [x] Reworked `hub.Broadcast` to snapshot client list before writing, set a per-write deadline (10s), and auto-remove dead clients — one slow or dead client can no longer block delivery to all others
- [x] Added `TestHubBroadcastDropsDeadClient` test covering the dead-conn auto-removal path
- [x] Bound server to `127.0.0.1` (was all interfaces); prevents LAN-reachable exposure
- [x] Added `http.Server` timeouts: `ReadHeaderTimeout` 10s, `WriteTimeout` 30s, `IdleTimeout` 120s
- [x] Gated pprof and statsviz behind `cfg.DevMode`
- [x] Replaced `CheckOrigin: return true` with an explicit localhost/127.0.0.1 hostname check

### 2026-05-27 (close-out pass)
- [x] Added `plugins/debugassistant/go.mod` and added `./plugins/debugassistant` to `go.work`; all plugins now have a module and are in the workspace
- [x] Updated `README.md`: Go 1.26+, plugin architecture overview, `make test-all` and `make all-plugins` documented
- [x] Added `docs/plugin-ownership.md`: feature ownership matrix and WASM trust model (trusted extensions, no capability scoping)
- [x] Added Testing section to `docs/wasm-plugins.md`: `stub_main.go` pattern, build tags, `make test-plugins`/`test-all` commands
- [x] Event payload DTO duplication reviewed and closed: each plugin defines its own projection of event payloads; intentionally local
- [x] All phase completion gates satisfied; plan marked complete

### 2026-05-26
- [x] Locked Phase 0 decisions: `PluginID` + `ViewID`, runtime-inactive disabled semantics, host-core `history`, and host-served plugin public data route
- [x] Added architecture RFC to document ownership and lifecycle semantics
- [x] Added mixed-style lifecycle regression tests in `internal/core/server_test.go` for nav disabled filtering, settings schema enabled flags, and plugin view resolution by `ViewID`
- [x] Added centralized server-private plugin lifecycle helpers in `internal/core/server.go` (`disabledPluginSet`, `activePlugins`, and `findPluginViewTarget`) to reduce duplicate logic ahead of Phase 1 semantics enforcement
- [x] Enforced disabled runtime behavior in `internal/core/server.go` for `InitPlugins()`, route/asset registration, and plugin view lookup; settings schema still lists disabled plugins with `enabled=false`
- [x] Confirmed strict dependency behavior: an enabled plugin requiring a disabled plugin now fails init with a clear dependency error
- [x] Updated `web/app.js` plugin boot flow so only enabled plugins are view/script injected, matching runtime-disabled semantics
- [x] Rewrote `docs/plugins.md` for the current plugin lifecycle, strict PluginID/ViewID identity split, and PluginID-based view/script loading
- [x] Updated `docs/wasm-plugins.md` for strict PluginID/ViewID identity, PluginID-based loading paths, host mounts, and current dependency/runtime semantics
- [x] Completed single-pass identity cutover with no backward compatibility: `/api/plugins/{pluginID}/view` and `/plugins/{pluginID}/...` are canonical, settings schema now exposes `view_id`, and frontend uses `plugin_id` for loading and `view_id` for navigation
- [x] Implemented host public file route `/api/plugins/{pluginID}/data/{path...}` backed by `plugin_data/{pluginID}/public/...` with traversal protection and regression tests
- [x] Migrated Debug Assistant screenshots to host public route serving via `/api/plugins/debugassistant/data/screenshots/...` and removed plugin-side binary screenshot HTTP responses
- [x] Audited `BodyBytes` usage: response-side binary path now avoided for screenshots; request-side upload path remains in `ballchasing`
- [x] Added native `stub_main.go` scaffold for `plugins/debugassistant`; `make test-plugins` now passes
- [x] Removed response-side `HTTPResponse.BodyBytes` from SDK ABI and host response writer path while keeping request-side upload `BodyBytes`
- [x] Enforced host-core history semantics: history cannot be runtime-disabled, disabled config entries for `history` are sanitized, settings show history as always-on, and API/docs now describe history as host-owned
- [x] Locked `history` as host-core in runtime semantics: disabling `history` in config is sanitized/ignored, history remains enabled in nav/schema, and settings UI marks it as always-on
- [x] Upgraded WASM route metadata from string-only paths to typed route declarations with explicit method support and host-side method guards
- [x] Expanded WASM settings metadata to align with host settings model (labels/types/defaults/options/placeholders/developer flags)
- [x] Added strict WASM route metadata validation on load (required/absolute path, supported methods, duplicate path rejection)
- [x] Reviewed weak API surface: `DBPrefix` retained as deprecated compatibility; `DeclaredEvents` retained as first-class with load-time metadata validation
- [x] Migrated WASM route metadata to typed method+path definitions (`RouteMeta`) and added host-side method guards (405 on mismatch when method is declared)
- [x] Added SDK export helpers (`WriteMetadata`, `WriteJSONOutput`, `HandleHTTPExport`) and migrated WASM plugin mains to remove repeated metadata/HTTP marshal boilerplate

---

## Notes / Decisions Log

Use this section to record explicit decisions as they are made.

### Resolved (previously pending)
- [x] API naming migration: single-pass cutover completed; `PluginID`/`ViewID` terminology enforced in code and docs; no backward-compat aliases
- [x] `HTTPResponse.BodyBytes` removed from response path; request-side `BodyBytes` retained for outbound uploads (ballchasing)

### Confirmed
- [x] Canonical identity is `PluginID` for runtime/data/API and `ViewID` for frontend navigation
- [x] Disabled plugins are runtime-inactive for init/routes/assets and remain visible in settings
- [x] `history` is a host-core feature
- [x] Public plugin data is served by host route `/api/plugins/{pluginID}/data/{path...}` from `plugin_data/{pluginID}/public/...` only
- [x] Dependency model is kept; startup fails fast if an enabled plugin depends on a disabled plugin
- [x] No backward compatibility layer for identity migration; plugin view/assets loading now requires `PluginID`

