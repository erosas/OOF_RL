# Momentum Overlay Lifecycle Wrap-Up

Status: lifecycle and control baseline checkpoint

Scope: documentation only. This wrap-up records the current rebuilt Momentum
Overlay control surface, what is safe to leave as the baseline, and what remains
deferred before returning to Momentum Timeline work.

## Purpose

This document answers:

```text
Is the rebuilt Momentum Overlay controllable, non-invasive, and safe to leave as
a finished baseline?
```

The short answer is:

```text
Yes, for the current internal/manual baseline.
```

The overlay can be launched and controlled without making Momentum Engine depend
on overlay lifecycle, without exposing user-facing settings, and without adding
plugin-manager semantics.

## Current Baseline

The rebuilt Momentum Overlay currently has:

- app-owned Momentum runtime infrastructure,
- read-only `momentum.SnapshotProvider` access,
- Overlay HUD ViewModel/RenderModel/SVG display pipeline,
- request-driven `DisplayAdapter`,
- internal browser preview route,
- internal SVG preview route,
- internal manual native shell launch route,
- internal manual show/hide/dormant control route,
- shell state tracking for launched/visible/hidden/dormant,
- legacy app overlay hotkey behavior kept separate.

## Routes

| Route | Purpose | User-Facing? | Notes |
| --- | --- | --- | --- |
| `/internal/momentum-overlay-preview` | Minimal HTML wrapper around the Momentum SVG output. | No | Internal validation/display target. |
| `/internal/momentum-overlay-preview.svg` | Request-driven SVG output for refresh. | No | Used by the preview wrapper. |
| `/internal/momentum-overlay-launch` | Manual native shell launch/reuse. | No | Validation route, not startup behavior. |
| `/internal/momentum-overlay-control?action=status` | Report shell state. | No | Does not create a shell. |
| `/internal/momentum-overlay-control?action=show` | Show an already launched shell. | No | Does not create a shell. |
| `/internal/momentum-overlay-control?action=hide` | Hide an already launched shell. | No | Does not stop Momentum runtime. |
| `/internal/momentum-overlay-control?action=dormant` | Put the shell into dormant display state. | No | Current implementation hides the shell. |

These routes intentionally remain hidden from nav/settings.

## Hotkey Findings

The current codebase has two separate concepts:

1. The legacy overlay window path uses the configured `overlay_hotkey`, which
   defaults to `F9`.
2. The rebuilt Momentum Overlay path uses internal manual launch/control routes.

No dedicated committed `F8` Momentum Overlay hotkey binding was found in the
current code during this wrap-up inspection.

That means:

- `F9` remains the legacy overlay path for now.
- The rebuilt Momentum Overlay is controlled through internal validation routes
  for this baseline.
- F8/F9 unification and user-configurable overlay hotkeys remain future Overlay
  Manager work.

This is an intentional containment decision for this wrap-up. Adding a global
hotkey binding now would expand the PR from lifecycle documentation into
application-level input ownership.

## Shell State Model

| State | Meaning | Current Owner |
| --- | --- | --- |
| `not-launched` | No retained Momentum native shell exists. | Overlay HUD plugin state. |
| `visible` | A retained shell exists and was asked to show. | Manual launch/control routes. |
| `hidden` | A retained shell exists and was asked to hide. | Manual control route. |
| `dormant` | A retained shell exists and display work should be idle. | Manual control route. |

Repeated launch requests reuse the retained shell instead of creating unlimited
shells.

## Engine Independence

Momentum Engine remains independent from overlay visibility:

- hiding the overlay does not stop Momentum runtime,
- dormant overlay state does not reset Momentum state,
- control routes do not mutate Momentum Engine math,
- display routes consume snapshots through read-only access,
- no overlay route writes DB, Session, History, Live, saved matches, or replay
  data.

This preserves the intended future model:

```text
Engine ON + Overlay ON  = enrichment runs and HUD can render.
Engine ON + Overlay OFF = enrichment runs with no visible HUD.
Engine OFF + Overlay OFF = no enrichment and no HUD.
Engine OFF + Overlay ON = invalid/disabled or requires Momentum Engine.
```

## Display Workload Boundary

The rebuilt display pipeline is request-driven:

- `DisplayAdapter` renders only when `RenderHTML` or `RenderSVG` is called,
- the preview wrapper refreshes the SVG route while that page is loaded,
- hiding/dormant control does not itself create a polling loop,
- manual shell creation does not install a hotkey listener,
- manual shell creation does not add plugin lifecycle ownership.

The current dormant operation hides the shell. It does not yet observe external
window destruction or pause an already loaded WebView script from outside the
page. Those details belong to a later Overlay Manager/native shell hardening
slice if needed.

## What Is Safe To Leave

Safe baseline behavior:

- Momentum runtime can run without Overlay HUD.
- Overlay HUD can render from read-only Momentum snapshots.
- Internal preview and launch routes are available for validation.
- Manual show/hide/dormant routes are available for controlled testing.
- Legacy `F9` behavior remains untouched.
- No user-facing settings, presets, Overlay Lab, or plugin manager behavior were
  added.

## Deferred Items

The following are intentionally deferred:

- live match clock,
- goal/replay reset parity,
- confidence label/value polish,
- extra event effects,
- dedicated rebuilt Momentum hotkey binding,
- F8/F9 configurable hotkey unification,
- true transparent borderless overlay shell hardening,
- external shell-close detection,
- Overlay Manager / Plugin Manager,
- Overlay Lab,
- settings/config/presets,
- remaining visual polish.

## Acceptance Criteria For This Baseline

- Manual launch route creates or reuses one shell.
- Control route reports not-launched safely.
- Control route show/hide/dormant actions do not create a shell when none
  exists.
- Control route show/hide/dormant actions operate on the retained shell when one
  exists.
- Momentum runtime remains independent from shell visibility.
- Legacy overlay hotkey behavior remains untouched.
- Rebuilt Momentum hotkey unification is documented as deferred.
- Routes remain hidden/internal.
- No DB, Session, History, Live, saved match, or replay mutation is introduced.

## Recommended Next Work

Return to Momentum Timeline with the stabilized Momentum runtime/display
foundation.

Suggested branch:

```text
feature/momentum-timeline-reentry-plan
```

Goal:

```text
Resume Momentum Timeline using the now-stabilized Momentum runtime/display
foundation.
```

