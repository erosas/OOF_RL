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
