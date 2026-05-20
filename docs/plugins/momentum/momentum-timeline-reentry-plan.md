# Momentum Timeline Reentry Plan

Status: docs-only reentry plan

Scope: planning only. This document defines how to resume Momentum Timeline
work after the Momentum Overlay rebuild, without implementing Timeline UI,
storage, routes, or runtime code in this PR.

## Purpose

Resume Momentum Timeline from the stabilized Momentum runtime foundation.

The goal is to create a clean path toward a player-useful Timeline feature
without depending on Overlay HUD display internals or rushing persistence before
the event model is validated.

## Non-Goals

- Do not implement Timeline UI.
- Do not add frontend frameworks, React, TypeScript, Vite, Webpack, or new
  dependencies.
- Do not add DB/schema changes.
- Do not mutate Session, History, Live, saved match, or replay data.
- Do not change Momentum Engine math.
- Do not touch Overlay HUD behavior, routes, shell lifecycle, or visual parity.
- Do not add The Lab, Overlay Manager, Plugin Manager, or Boost work.
- Do not adopt or move design assets into production in this PR.

## Current Momentum Foundation

Momentum now has a layered runtime foundation:

```text
typed GameActionEvent
-> Momentum Engine
-> runtime Service
-> event wiring
-> runtime registration
-> read-only SnapshotProvider
```

Current runtime inputs available to Timeline:

- typed `oofevents.GameActionEvent` values,
- match lifecycle events,
- read-only `momentum.SnapshotProvider` snapshots,
- runtime-only `MomentumState` values.

Timeline must not consume Overlay HUD-specific objects:

- no `overlayhud.ViewModel`,
- no `overlayhud.RenderModel`,
- no SVG renderer output,
- no Overlay HUD route/display adapter output.

The overlay display pipeline is presentation-owned. Timeline should be
Momentum-owned or Timeline-owned runtime data.

## Artifacts Reviewed

### Tracked Status

`git ls-files` found no tracked files under `docs/plugins/momentum` and no
tracked root `momentum-timeline*.png` files at the time of this plan.

### Untracked Design Assets

The following untracked artifacts were found:

| Artifact | Current Use | Disposition |
| --- | --- | --- |
| `docs/plugins/momentum/assets/timeline-icons/goal.svg` | Design/reference icon | Keep as reference; consider production adoption in a later asset PR. |
| `docs/plugins/momentum/assets/timeline-icons/save.svg` | Design/reference icon | Keep as reference; consider production adoption in a later asset PR. |
| `docs/plugins/momentum/assets/timeline-icons/epic_save.svg` | Design/reference icon | Keep as reference; consider production adoption in a later asset PR. |
| `docs/plugins/momentum/assets/timeline-icons/assist.svg` | Design/reference icon | Keep as reference; consider production adoption in a later asset PR. |
| `docs/plugins/momentum/assets/timeline-icons/shot.svg` | Design/reference icon | Keep as reference; consider production adoption in a later asset PR. |
| `docs/plugins/momentum/assets/timeline-icons/demo.svg` | Design/reference icon | Keep as reference; consider production adoption in a later asset PR. |
| `docs/plugins/momentum/assets/timeline-icons/timeline-icon-design-notes.md` | Icon design notes | Reference only for future visual/UI work. |
| `docs/plugins/momentum/assets/timeline-icons/timeline-icon-preview.html` | Static preview sandbox | Reference only; do not wire into production. |
| `docs/plugins/momentum/assets/timeline-icons/timeline-icon-preview.png` | Preview screenshot | Reference only. |
| `momentum-timeline-preview.png` | Timeline concept screenshot | Reference only. |
| `momentum-timeline-svg-preview.png` | Timeline concept screenshot | Reference only. |
| `momentum-timeline-ux-hardening.png` | Timeline concept screenshot | Reference only. |
| `momentum-timeline-visual-tuning.png` | Timeline concept screenshot | Reference only. |

The icon design notes explicitly describe these assets as a standalone design
pass that is not wired into production UI and does not touch Momentum Engine
math, live/event-bus data, DB/schema, History, Session, saved matches, replay,
or Overlay HUD.

## Existing Timeline Code

No production Momentum Timeline code was found during this reentry inspection.

Existing references are design/reference artifacts or unrelated uses of the
word "timeline" in History/debug documentation and test wording.

## Data Source Options

| Source | Usefulness | V1 Decision |
| --- | --- | --- |
| `oofevents.GameActionEvent` | Best source for discrete Timeline markers such as goals, shots, saves, assists, demos, own goals, and epic saves. | Use. |
| Momentum runtime snapshots | Best source for pressure/control/confidence/volatility context around each marker. | Use through a read-only provider. |
| Match lifecycle events | Needed for start/restart/end/destroy boundaries. | Use for runtime collector resets/freeze behavior. |
| Overlay HUD ViewModel/RenderModel/SVG | Presentation-specific display contract. | Do not use. |
| Session/History/Live/replay storage | Useful later for persistence and review. | Defer until runtime model is validated. |
| Design icon SVGs/screenshots | Useful later for visual UI work. | Defer production adoption. |

## Recommended V1 Architecture

V1 should be runtime-only first.

Add a small Timeline runtime collector that consumes typed events and samples
Momentum snapshots without writing to persistence.

Recommended ownership:

```text
typed event bus
-> Momentum runtime service
-> Momentum Timeline runtime collector
-> read-only Timeline snapshot/accessor
-> future Timeline UI/API
```

Recommended package shape for the first implementation slice:

```text
internal/momentum/timeline
```

or another Momentum-owned internal package if the existing codebase suggests a
clearer location during implementation.

The collector should be independent from Overlay HUD. It should not import
`internal/plugins/overlayhud`.

## Runtime Collector Model

The first collector should keep a bounded in-memory per-match buffer.

Possible V1 entry shape:

```text
TimelineEntry
- sequence/index
- match GUID
- occurred at
- action kind
- actor team
- impact team
- player ID/name
- victim ID for demos
- own-goal flag
- epic-save flag
- sampled blue/orange pressure
- sampled blue/orange control influence
- sampled confidence
- sampled volatility
- display-safe reason/category
```

The exact Go type should be chosen during implementation after inspecting local
Momentum patterns again.

Lifecycle behavior:

- reset on match started,
- reset or segment on match restarted,
- freeze or mark ended on match ended,
- clear on match destroyed,
- avoid writing any match/session/history rows.

## Persistence Decision

Defer persistence.

Reason:

Timeline is likely to become useful in History and match review, but storing it
too early risks locking in the wrong data model and touching sensitive
Session/History/saved-match paths before runtime behavior is proven.

V1 should validate:

- event ordering,
- entry shape,
- snapshot sampling,
- match lifecycle boundaries,
- buffer size,
- performance overhead,
- player-readable categories.

Only after that should a separate persistence design decide whether Timeline
entries belong in a dedicated table, saved match payload, replay-derived data,
or a derived view built from existing event records.

## Risks To Core Data

The main risks are avoided by keeping V1 runtime-only:

- no SQLite schema changes,
- no DB writes,
- no Session mutation,
- no History mutation,
- no Live view mutation,
- no saved match mutation,
- no replay data mutation.

Implementation risks still to manage:

- event ordering across typed events and snapshots,
- match lifecycle reset timing,
- memory growth if the buffer is not bounded,
- sampling stale Momentum snapshots after match end/destroy,
- overclaiming labels such as possession or tactical control.

Use safe labels:

- pressure,
- control influence,
- contest,
- confidence,
- volatility,
- event impact.

Avoid labels:

- possession,
- rotation control,
- win prediction,
- ball control,
- tactical advantage.

## First Implementation Slice

Branch:

```text
feature/momentum-timeline-runtime-collector
```

Scope:

- runtime collector package only,
- no UI,
- no routes,
- no persistence,
- no Overlay HUD dependency,
- no app startup registration unless clearly required and explicitly scoped.

Suggested behavior:

- create collector type with bounded buffer,
- accept typed `GameActionEvent`,
- sample read-only Momentum snapshot/provider,
- expose read-only Timeline snapshot,
- handle lifecycle reset/end/destroy methods,
- add focused unit tests with fake events and fake Momentum snapshots.

Acceptance criteria:

- collector records supported action events,
- collector stores safe event metadata and sampled Momentum values,
- collector resets/ends safely through lifecycle methods,
- snapshots are copied/read-only,
- buffer is bounded,
- no DB/session/history/live/replay code is touched,
- no Overlay HUD package is imported.

## Recommended PR Stack

1. `feature/momentum-timeline-runtime-collector`
   - Add runtime-only collector and unit tests.

2. `feature/momentum-timeline-runtime-wiring`
   - Wire collector to typed event bus and Momentum snapshot provider if the
     first slice intentionally avoids registration.

3. `feature/momentum-timeline-access-contract`
   - Add a narrow read-only Timeline provider/accessor for future consumers.

4. `feature/momentum-timeline-internal-preview`
   - Add hidden/internal read-only preview route or debug endpoint if needed
     for validation. No production UI yet.

5. `feature/momentum-timeline-ui-plan`
   - Plan the vanilla HTML/CSS/JS Timeline surface and decide whether the
     existing design assets should be adopted.

6. `feature/momentum-timeline-ui-v1`
   - Implement the first user-facing Timeline UI after runtime behavior is
     proven.

7. Future persistence PR
   - Only after runtime and UI shape are validated.

## Acceptance Criteria For This Plan

- Existing Timeline artifacts are classified.
- The V1 architecture is runtime-only.
- Persistence is explicitly deferred.
- Timeline is independent from Overlay HUD display contracts.
- Risks to Session/History/Live/replay data are documented.
- The next implementation PR has a clear, narrow scope.

## Open Questions

- Should Timeline live under `internal/momentum/timeline` or a sibling
  `internal/timeline` package?
- What buffer size is appropriate for one match without over-retaining data?
- Should match restarts create a new segment or clear the current match buffer?
- Should Timeline entries store raw sampled numeric values only, or also
  precomputed safe labels for UI?
- Which typed event kinds should be included in V1 beyond goal, shot, save,
  assist, demo, and ball hit?
