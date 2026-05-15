# Engine and Overlay Boundary

This document captures the architecture boundary introduced with Momentum Engine and Overlay HUD v1.

## Core Rule

Engines may run without overlays. Overlays may never be required for engines. Disabled overlays must do zero GUI rendering.

## Current PR Scope

PR #47 introduces Momentum Engine output and the Overlay HUD display surface together, but they should be treated as separate architectural responsibilities:

- Momentum Engine enriches event-derived pressure/control signals.
- Overlay HUD renders Momentum Engine output.
- Overlay HUD must not mutate Momentum Engine math, match data, saved history, session data, replay capture, or database rows.
- Disabling the Overlay HUD plugin disables the GUI/runtime surface after restart. It does not define a long-term engine enable/disable model.

## Reviewability Notes

PR #47 intentionally keeps the first Momentum Engine and Overlay HUD checkpoint behavior-preserving after review stabilization. Larger structure changes should be split into follow-up PRs instead of being folded into this runtime patch.

The current implementation contains several responsibilities in the same runtime files because the initial checkpoint proves the engine signal, overlay transport, native HUD lifecycle, and display widgets together. Future cleanup should separate those responsibilities without changing the display-only contract.

The Overlay HUD plugin should trend toward a thin forwarding/display boundary. Momentum interpretation, event-derived pressure/control state, confidence, volatility, and pulse decisions belong in engine-owned code. Overlay code should adapt engine output to widgets, expose read-only preview/debug surfaces, and report diagnostics.

## Follow-Up Refactor Targets

These items are candidates for immediate follow-up PRs after PR #47, not requirements for the first Overlay HUD checkpoint:

- Move normalized event flags closer to the core OOF event model. Examples include representing epic saves and own goals as core event fields/flags instead of Momentum-only normalization details.
- Split `internal/plugins/overlayhud/plugin.go` by responsibility. Candidate boundaries include route handlers, typed-event adaptation, event worker lifecycle, replay-state tracking, duplicate filtering, player reference cache, preferences, performance diagnostics, and native HUD visibility reporting.
- Split `internal/plugins/overlayhud/view.js` by frontend responsibility. Candidate boundaries include widget classes, Overlay Lab controls, preset import/export, performance probe/debug UI, native HUD visibility handling, and data fetch/adaptation.
- Keep low-level native window wrappers separate from app-specific Overlay HUD lifecycle. `internal/overlay/overlay.go` should trend toward platform/window primitives, while dormancy and HUD visibility policy should live in an app-specific overlay layer.
- Centralize plugin lifecycle policy so startup disable checks, route availability, and overlay boot behavior do not drift.
- Layer MomentumControlWheel styling/configuration into clearer base tokens, state styling, animation behavior, performance/reduced-motion overrides, and debug/probe styling.
- Keep future runtime path changes separate from design/reference artifacts where practical.

## Future Systems

The same boundary applies to planned enrichment systems:

- Momentum Engine must remain usable without Overlay HUD enabled.
- Boost Engine must remain usable without Boost Overlay enabled.
- Future Scoreboard Engine must remain usable without Scoreboard Overlay enabled.

Engine runtime and GUI rendering must stay separated.

When an overlay/plugin GUI is disabled:

- no overlay window should render
- no animation loop should run
- no polling/fetch loop should run
- no DOM/render updates should run
- no hidden background rendering should continue
- engine/event processing may continue if independently enabled

## Goal

Users should be able to keep enrichment engines running while disabling all GUI overlays for zero overlay-rendering overhead.

## Follow-Up

Define explicit settings and lifecycle semantics for separate `engine enabled` and `overlay enabled` controls before adding real Boost, Scoreboard, Journey, or Career integrations that depend on shared enrichment output.
