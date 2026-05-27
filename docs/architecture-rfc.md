# Architecture RFC: Plugin Identity and Runtime Ownership

Status: accepted (Phase 0)
Date: 2026-05-26
Owners: core/runtime

## Purpose

This RFC locks the foundational platform semantics used by host runtime, plugin SDK, and frontend plugin loading.

It exists to remove ambiguity before API cleanup and dedupe work.

## Decisions

### 1) Canonical identity

- `PluginID` is the canonical identity for runtime, API, data, and asset ownership.
- `ViewID` is a frontend navigation slug only.
- Route and data paths should be keyed by `PluginID`.
- Identity cutover is breaking: no backward-compatibility aliases for ViewID-based backend loading paths.

Examples:

- API: `/api/plugins/{pluginID}/...`
- Static plugin assets: `/plugins/{pluginID}/...`
- Plugin data root: `plugin_data/{pluginID}/...`

### 2) Disabled plugin semantics

Disabled plugins are runtime-inactive for plugin lifecycle and serving concerns:

- excluded from plugin initialization
- excluded from plugin route registration
- excluded from plugin asset registration

For Phase 0 and Phase 1, disabled plugins remain visible in settings so users can re-enable and configure them.

### 3) History ownership

`history` is host-core, not a normal plugin.

This means host runtime and storage own history behavior, while UI surface may remain plugin-like.

### 4) Binary/file serving

Response-side binary transfer through `sdk.HTTPResponse.BodyBytes` is deprecated.

Host serves plugin public files directly via:

- route: `/api/plugins/{pluginID}/data/{path...}`
- root: `plugin_data/{pluginID}/public/{path...}`

The host must enforce path normalization and traversal protection. Arbitrary files under `plugin_data/{pluginID}` are not exposed.

### 5) Dependency policy

- Keep plugin dependency metadata (`Requires`).
- Startup is strict: if an enabled plugin depends on a disabled plugin, plugin init fails fast with a clear dependency error.
- Disabled plugins remain listed in settings with `enabled=false` so users can resolve dependency issues by re-enabling.

## Rationale

- Separating `PluginID` and `ViewID` avoids route/data confusion and future API churn.
- Runtime-inactive disabled behavior aligns user intent with actual lifecycle behavior.
- Treating `history` as host-core matches current data/runtime ownership and reduces hybrid complexity.
- Host-served plugin public files remove expensive binary-through-WASM response plumbing.

## Implementation Scope

### Immediate (Phase 0/1)

- Add helpers in server lifecycle to filter active plugins.
- Apply disabled filtering to init/register paths.
- Keep settings visibility for disabled plugins.
- Start naming cleanup toward explicit `ViewID` terminology.

### Next (Phase 2)

- Add host public data route and tests.
- Migrate `debugassistant` screenshot flow.
- Keep request-side `BodyBytes` for outbound upload use cases.

## Non-goals (for this RFC)

- Full plugin capability sandbox model redesign.
- Full frontend refactor in the same change as backend lifecycle fixes.
- Immediate removal of response-side `BodyBytes` without migration window.

## Open follow-ups

- Decide exact deprecation timeline for `HTTPResponse.BodyBytes`.
- Normalize code naming from mixed `NavTab.ID` usage to explicit `ViewID` terms.
- Document route conflict and plugin ID conflict handling rules.

