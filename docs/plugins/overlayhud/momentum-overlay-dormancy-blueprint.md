# Momentum Overlay F9/Dormancy Blueprint

## Purpose

This blueprint defines the intended F9, visibility, hidden, and dormancy behavior for the rebuilt Momentum Overlay HUD before lifecycle code is implemented.

The goal is to keep the Momentum Engine and Overlay HUD separate:

- Momentum Engine owns event-derived pressure/control logic.
- Overlay HUD displays read-only Momentum state.
- Overlay visibility must not decide whether Momentum runtime infrastructure exists.
- Hidden or disabled overlay display must avoid GUI/rendering overhead.

## Current Rebuilt Vertical Slice

The rebuilt stack currently proves:

```text
typed events
-> Momentum Engine
-> runtime service
-> event wiring
-> runtime registration
-> SnapshotProvider
-> Overlay HUD consumer
-> ViewModel
-> RenderModel
-> SVG renderer
-> display adapter
-> surface target
-> manual native shell
-> manual launch route
```

This is a manual validation slice. It does not yet define F9 behavior, startup launch behavior, settings, or full overlay lifecycle ownership.

## State Model

### Not Launched

No native Momentum overlay shell exists.

Expected behavior:

- Momentum Engine may still run.
- SnapshotProvider remains available to internal Go consumers.
- No Momentum overlay WebView exists.
- No Momentum overlay GUI rendering occurs.
- F9 behavior is not implied by this state until implemented by a later PR.

### Manually Launched

The hidden internal manual launch route has created a Momentum overlay shell for validation.

Expected behavior:

- The shell loads the Momentum overlay surface target.
- The shell may be visible immediately for manual verification.
- The shell is retained only narrowly enough to avoid immediate destruction and duplicate launches.
- This is not app startup behavior.
- This is not plugin lifecycle ownership.

### Visible

The native shell exists and is shown.

Expected behavior:

- The shell displays the current overlay route output.
- Rendering is allowed because the overlay is visible.
- Momentum state remains read-only from the overlay's perspective.
- The overlay must not mutate Momentum Engine state or saved app data.

### Hidden

The native shell exists but is not visible.

Expected behavior:

- Momentum Engine continues to receive typed events and update runtime-only state.
- Overlay GUI rendering should stop or be minimized as much as the implementation allows.
- No user-facing overlay pixels should be visible.
- Hidden state must not mutate DB, Session, History, Live, saved matches, replay data, or Momentum Engine math.

### Dormant

The overlay display layer is intentionally idle.

Expected behavior:

- No active GUI rendering loop.
- No overlay polling loop.
- No repeated SVG/HTML generation unless a request or explicit show action occurs.
- The native shell may be hidden or not created, depending on the implementation phase.
- Momentum Engine can remain active independently.

Dormant is stronger than hidden: hidden describes visibility, while dormant describes display workload.

### Closed/Destroyed

The native shell has been destroyed or is no longer available.

Expected behavior:

- Momentum Engine remains independent.
- Future manual launch or F9 behavior may create a fresh shell.
- Any retained shell reference should be cleared when a later lifecycle-aware implementation can observe destruction safely.
- Closing the shell must not stop Momentum runtime infrastructure.

## F9 Behavior Expectations

Future F9 behavior should be a display visibility control, not an engine lifecycle control.

Expected V1 behavior:

- If the shell is not launched, F9 may create and show it.
- If the shell is visible, F9 hides it.
- If the shell is hidden, F9 shows it.
- F9 must not start or stop Momentum Engine.
- F9 must not mutate Session, History, Live, saved matches, replay data, or SQLite rows.
- F9 must not become plugin manager load/unload behavior.

F9 hold/toggle behavior should follow existing app configuration only when a later PR explicitly scopes that integration.

## Manual Launch Behavior Expectations

Manual launch remains an internal validation path.

Expected behavior:

- `GET /internal/momentum-overlay-launch` creates or reuses one shell.
- Repeated launch requests should not create unlimited overlay shells.
- The route should remain hidden from nav and settings.
- The route should be safe to remove or replace after F9/manual controls mature.
- Manual launch is not startup behavior.

## Engine Behavior While Overlay Is Hidden

When the overlay is hidden or dormant, Momentum Engine should continue to:

- consume typed game action events,
- update runtime-only Momentum state,
- expose read-only snapshots through SnapshotProvider,
- remain independent from Overlay HUD lifecycle.

This supports the future model where Momentum enrichment can run without visible overlay rendering.

## Overlay Work That Must Stop While Hidden

When hidden or dormant, Overlay HUD display should avoid:

- active SVG regeneration loops,
- WebView animation loops,
- polling SnapshotProvider for display-only updates,
- repeated HTML rendering,
- unnecessary route requests,
- any GPU/compositor work caused by visible overlay effects.

The implementation may still keep a lightweight native shell reference if that is the safest way to preserve manual state, but it should not actively render display updates while hidden.

## Zero GUI/Rendering Overhead

For this rebuild, "zero GUI/rendering overhead" means:

- no visible overlay window,
- no display polling loop,
- no animation loop,
- no repeated SVG/HTML generation for hidden display,
- no route refresh for hidden display,
- no optional visual effects running in WebView,
- no extra frontend work beyond the app's normal background runtime.

It does not mean Momentum Engine is off. Engine runtime overhead is controlled separately from overlay display overhead.

## Future Engine/Overlay Enablement Model

The desired future behavior is:

```text
Engine ON + Overlay ON  = Momentum enrichment runs and HUD can render.
Engine ON + Overlay OFF = Momentum enrichment runs, no GUI overlay rendering.
Engine OFF + Overlay OFF = no Momentum enrichment and no overlay rendering.
Engine OFF + Overlay ON = invalid or disabled because Overlay HUD requires Momentum Engine.
```

This blueprint does not implement those settings. It defines the behavior that later settings and lifecycle work should preserve.

## Not Plugin Manager Behavior

This blueprint does not define a plugin manager.

Out of scope:

- plugin load/unload semantics,
- dependency graph ownership,
- route ownership framework,
- background worker orchestration,
- cleanup hook framework,
- restart-required vs live-toggle behavior,
- generalized settings UI for all plugins.

Momentum Overlay dormancy should stay narrowly focused on this overlay surface until a separate plugin manager blueprint exists.

## Not Overlay Lab Behavior

This blueprint does not define Overlay Lab.

Out of scope:

- visual controls,
- presets,
- theme editors,
- CSS token editing UI,
- performance tuning controls,
- user-facing preview panels,
- frontend control surfaces.

Overlay Lab can consume this lifecycle contract later, but it should not be required for F9/dormancy implementation.

## Not Settings/Config Behavior Yet

This blueprint does not add or change user settings.

Out of scope:

- new config fields,
- overlay enabled/disabled settings,
- engine enabled/disabled settings,
- hotkey rebinding,
- hold/toggle mode changes,
- persistence of overlay-specific lifecycle state.

Existing overlay config may be read by a future implementation only when that PR explicitly scopes the behavior.

## Startup/Autostart Non-Goals

Future implementation must not assume autostart unless a PR explicitly scopes it.

Non-goals:

- launching Momentum overlay at app startup,
- showing Momentum overlay automatically on app start,
- creating a shell before manual/F9 action,
- making manual launch route part of normal user workflow.

## Failure Handling Expectations

Future implementation should handle failures without breaking Momentum runtime:

- If WebView2 shell creation fails, return a clear error and leave Momentum Engine running.
- If the surface target URL cannot be built, do not create a shell.
- If a shell already exists, reuse or no-op rather than spawning duplicates.
- If the shell is closed externally, later lifecycle-aware code should clear its retained reference when feasible.
- Failure to show/hide the overlay must not affect event ingestion, Momentum state, Session, History, Live, replay capture, or DB state.

## Future Implementation Phases

### Phase 1: Launch Hardening

- Improve manual launch response clarity.
- Add launcher failure tests if missing.
- Add manual validation notes if useful.
- Keep hidden/internal route behavior.

### Phase 2: Controlled Dormancy

- Add explicit show/hide/dormant operations around the manual shell.
- Keep operations request-driven.
- Do not add F9 yet.
- Verify hidden state avoids display refresh work.

### Phase 3: F9 Integration

- Wire F9 to the controlled show/hide operations.
- Preserve Momentum Engine independence.
- Avoid plugin manager semantics.
- Avoid autostart unless separately scoped.

### Phase 4: Settings and User Controls

- Add overlay enablement only after lifecycle behavior is proven.
- Keep engine enablement separate from overlay enablement.
- Avoid Overlay Lab or preset systems unless explicitly scoped.

### Phase 5: Overlay Lab and Visual Controls

- Add visual controls after runtime lifecycle behavior is stable.
- Preserve reduced-motion and performance rules.
- Keep display-only systems from mutating Momentum Engine math or saved data.

## Acceptance Criteria for Later Implementation PRs

Later implementation PRs should prove:

- Momentum Engine keeps running while overlay is hidden.
- Overlay HUD can be hidden without visible rendering.
- Hidden/dormant overlay does not poll or regenerate display output.
- F9 show/hide does not mutate DB, Session, History, Live, saved matches, replay data, or Momentum Engine math.
- Repeated launch/show requests do not create unlimited shells.
- Failure to create or show a shell is contained and visible to the caller.
- Overlay HUD remains hidden from nav/settings until a PR explicitly scopes user-facing controls.
- Manual launch remains internal-only or is removed when superseded by controlled runtime behavior.

