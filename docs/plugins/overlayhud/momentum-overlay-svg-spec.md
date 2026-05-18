# Momentum Overlay SVG Display Spec

## Purpose

Define the display contract for the future Momentum Overlay HUD SVG renderer.

This document maps `overlayhud.ViewModel` into a visual overlay blueprint without
adding rendering code, routes, runtime registration, frontend controls, or
overlay window behavior.

The intended architecture chain is:

```text
Momentum Engine -> SnapshotProvider -> overlayhud.ViewModel -> SVG/display contract -> future renderer
```

## Non-Goals

- Implementing SVG, HTML, CSS, or JavaScript rendering.
- Registering Overlay HUD in app startup.
- Adding routes, assets, WebView behavior, F9 behavior, or Overlay Lab controls.
- Changing Momentum Engine math or `overlayhud.ViewModel`.
- Writing to SQLite or mutating Session, History, Live, saved match, replay, or capture data.
- Adding dependencies or a frontend build pipeline.
- Claiming possession, tactical certainty, rotations, win odds, or coaching-grade control.

## Component Geometry

The early rebuilt renderer used a simplified `0 0 320 320` arc wheel as
validation scaffolding while the Momentum runtime, preview route, and native
surface were rebuilt. That geometry is not the parity target.

The renderer should now target the old PR #47 `MomentumControlWheel` geometry so
layout and future CSS/motion parity have stable hooks.

- SVG `viewBox`: `0 0 1024 1024`
- Nominal authored size: `1024px x 1024px`
- Minimum readable rendered size: `220px x 220px`
- Center point: `512,512`
- Outer frame radius: `410`
- Segment ring radius: `365`
- Inner tick ring: `258..274`
- Center disc radius: `230`
- Inner rim radius: `238`
- Segment count: `96`
- Tick count: `120`
- Text baseline should be centered and must not overlap the segment ring.

The SVG root should remain square. Responsive scaling may change rendered pixel
size, but it must preserve the `1:1` aspect ratio and the `0 0 1024 1024`
coordinate system.

## Component Dimensions

| Element | Geometry |
| --- | --- |
| Outer mechanical frame | radius `410` |
| Segment pill ring | `96` rounded rects, `x=505`, `y=108`, `width=14`, `height=108`, rotated around `512,512` |
| Segment bevels | `96` thin rounded rects, `x=507.5`, `y=113`, `width=3`, `height=42`, rotated around `512,512` |
| Inner tick ring | `120` radial lines from radius `258` to `274` |
| Center disc | radius `230` |
| Timer row | centered near `y=506` |
| State label row | centered near `y=570` |
| Status overlay row | centered below state label when needed |

Timer text should remain the most readable element. Other labels must shrink,
hide, or simplify before the timer becomes cramped.

## Layer Order

Future SVG output should use this back-to-front order, matching the old
MomentumControlWheel layer manifest where practical:

1. `momentum-wheel-root`
2. `defs`
3. `background`
4. `outer-aura`
5. `outer-energy-streaks`
6. `outer-sparks`
7. `outer-mechanical-frame`
8. `segment-ring-underlay`
9. `segment-ring-active`
10. `segment-ring-bevels`
11. `inner-tick-ring`
12. `center-disc`
13. `center-color-washes`
14. `center-texture`
15. `center-rim`
16. `contested-front-line`
17. `text-layer`
18. `oof-badge`
19. `debug-overlays`
20. `hud-status-overlay`

The status overlay is only for inactive, stale, and no-data states. It should not
cover the timer when a timer is available.

## Editable SVG Group Names

Use stable group names so future tests, screenshots, and debug tools can target
specific layers:

- `svg#momentum-control-wheel`
- `g#momentum-wheel-root`
- `g#background`
- `g#outer-aura`
- `g#outer-energy-streaks`
- `g#outer-sparks`
- `g#outer-mechanical-frame`
- `g#segment-ring-underlay`
- `g#segment-ring-active`
- `g#segment-ring-blue-active`
- `g#segment-ring-orange-active`
- `g#segment-ring-neutral-caps`
- `g#segment-ring-bevels`
- `g#inner-tick-ring`
- `g#inner-tick-ring-base`
- `g#inner-tick-ring-blue`
- `g#inner-tick-ring-orange`
- `g#inner-tick-ring-muted`
- `g#inner-crosshair-lines`
- `g#center-disc`
- `g#center-color-washes`
- `g#center-texture`
- `g#center-rim`
- `g#contested-front-line`
- `g#text-layer`
- `g#oof-badge`
- `g#debug-overlays`
- `g#hud-status-overlay`

Dynamic child nodes should use `data-state`, `data-team`, or `data-segment`
attributes instead of changing group IDs.

## Arc And Ring Geometry

Momentum share is represented as individual pill segments, not a stroked arc.

- Segment count: `96`
- Segment angle step: `3.75deg`
- Segment rect: `x=505`, `y=108`, `width=14`, `height=108`, `rx=999`, `ry=999`
- Rotation origin: `512,512`
- Segment angle: `index * 3.75`
- Blue ownership begins at `180deg` and fills clockwise by `BlueShare * 360`.
- Orange owns the remaining visible territory.
- If shares do not sum to `1`, the renderer should normalize them for display.
- If both shares are unavailable, render the neutral `0.5 / 0.5` state.
- Do not inflate a true zero share into a visible team advantage.

Volatility affects seam/cap emphasis and later motion/styling, but this geometry
PR only needs stable segment/tick primitives. Full visual volatility styling is
deferred to PR #78.

## Segment Count Rules

The old MomentumControlWheel uses fixed-count geometry:

- Momentum pill segments: `96`
- Inner tick marks: `120`
- Tick angle step: `3deg`
- Tick line: radial line from radius `258` to `274`
- Crosshair lines:
  - vertical `512,268 -> 512,756`
  - horizontal `268,512 -> 756,512`

Segment and tick counts should remain constant across states to avoid layout
churn.

## Center Timer And State Label Rules

The center region has three text roles:

- Timer: primary, largest, always preferred when available.
- State label: secondary, safe descriptive label.
- Confidence caption: tertiary, optional compact confidence text.

Text rules:

- Timer maximum length target: `5` characters, such as `4:12` or `0:07`.
- Timer fallback: `--:--`.
- State labels must use safe terms only:
  - `NO DATA`
  - `NEUTRAL`
  - `BLUE PRESSURE`
  - `ORANGE PRESSURE`
  - `BLUE CONTROL`
  - `ORANGE CONTROL`
  - `VOLATILE`
  - `STALE`
  - `INACTIVE`
- Do not use possession, rotation, predicted win, tactical advantage, or ball
  control language.
- The renderer should prefer smaller text over overlap.
- If space is constrained, hide the confidence caption before hiding the state
  label.

## ViewModel To Visual State Mapping

| ViewModel field | Visual use |
| --- | --- |
| `MatchActive` | Enables active styling. False dims the component. |
| `HasData` | Controls no-data fallback styling. |
| `IsStale` | Applies stale overlay and reduces emphasis. |
| `BlueShare` | Blue momentum arc share. |
| `OrangeShare` | Orange momentum arc share. |
| `StateLabel` | Center state label text. |
| `Confidence` | Outer confidence ring intensity/progress. |
| `Volatility` | Inner volatility segment activation. |
| `LastUpdated` | Optional future debug or staleness display source. |

The renderer should treat `ViewModel` as immutable input for one render frame.
It must not call Momentum mutation methods or read raw engine internals.

## Confidence Visual Mapping

Confidence communicates how strongly the display should present the current
momentum signal. It is not a win probability.

| Confidence | Visual treatment |
| --- | --- |
| `0.00` | Hidden or minimal confidence ring. |
| `0.01 - 0.34` | Low opacity ring, no glow. |
| `0.35 - 0.66` | Standard opacity ring. |
| `0.67 - 1.00` | Strong opacity ring, optional subtle emphasis. |

Optional emphasis must be disabled in reduced-motion or performance modes.

## Volatility Visual Mapping

Volatility communicates how unstable or shifting the recent event-derived signal
is. It is not tactical certainty.

| Volatility | Visual treatment |
| --- | --- |
| `0.00 - 0.20` | Few or no active ticks. |
| `0.21 - 0.50` | Moderate active ticks. |
| `0.51 - 0.80` | Many active ticks, stronger contrast. |
| `0.81 - 1.00` | Near-full ticks, optional static warning accent. |

Volatility may affect contrast and tick count. It should not shake, pulse, or
move the timer in reduced-motion mode.

## Stale, No-Data, And Inactive States

### Stale

When `IsStale` is true:

- Reduce ring opacity.
- Show `STALE` only if there is room and it does not compete with the timer.
- Keep last known shares visible but subdued.
- Do not imply current pressure.

### No Data

When `HasData` is false:

- Use neutral `0.5 / 0.5` share display.
- Use `NO DATA` as the state label.
- Hide confidence emphasis.
- Keep timer readable if available.

### Inactive

When `MatchActive` is false:

- Dim the component.
- Stop optional motion.
- Preserve shape and spacing.
- Do not clear the display unless the future runtime explicitly sends an empty
  view model.

## CSS And Display Tokens

Future renderer CSS should use tokens rather than hard-coded repeated colors.

| Token | Intended use |
| --- | --- |
| `--overlayhud-blue` | Blue team arc. |
| `--overlayhud-orange` | Orange team arc. |
| `--overlayhud-track` | Neutral ring tracks. |
| `--overlayhud-panel` | Center panel fill. |
| `--overlayhud-text-primary` | Timer text. |
| `--overlayhud-text-secondary` | State and captions. |
| `--overlayhud-muted` | Stale and inactive elements. |
| `--overlayhud-warning` | High volatility accent. |
| `--overlayhud-shadow` | Optional depth/shadow. |

Initial token values should be chosen for contrast against both dark and bright
Rocket League scenes. The renderer should avoid a one-hue palette.

## Reduced-Motion Rules

Reduced-motion mode must preserve readability and meaning.

- Disable sweep animations.
- Disable pulsing, glow breathing, shake, and spinning.
- Keep ring state updates instantaneous or use short opacity transitions only.
- Do not animate timer position, size, or opacity.
- Volatility should use static segment counts.
- Confidence should use static ring progress and opacity.

Performance mode should follow the same rules and may also disable shadows,
filters, and expensive blur effects.

## Accessibility And Readability Constraints

- Timer readability is mandatory.
- Text must not overlap rings or adjacent text.
- State labels must remain safe and non-overclaiming.
- Color must not be the only indicator of team pressure; arc position and label
  should also communicate state.
- Use sufficient contrast for timer and state labels.
- Avoid tiny text below practical overlay readability.
- Preserve a stable square hitbox even when data is stale or inactive.
- Do not use flashing or rapid repeated transitions.

## Fixture ViewModel States

Future renderer tests should include these fixture states.

### Neutral Empty

```go
overlayhud.ViewModel{
    MatchActive: false,
    HasData: false,
    IsStale: true,
    BlueShare: 0.5,
    OrangeShare: 0.5,
    StateLabel: "NO DATA",
    Confidence: 0,
    Volatility: 0,
}
```

### Active Shifting

```go
overlayhud.ViewModel{
    MatchActive: true,
    HasData: true,
    IsStale: false,
    BlueShare: 0.52,
    OrangeShare: 0.48,
    DisplayState: "neutral",
    StateLabel: "NEUTRAL",
    Confidence: 0.42,
    Volatility: 0.38,
}
```

### Blue Pressure

```go
overlayhud.ViewModel{
    MatchActive: true,
    HasData: true,
    IsStale: false,
    BlueShare: 0.72,
    OrangeShare: 0.28,
    DisplayState: "blue-control",
    StateLabel: "BLUE CONTROL",
    Confidence: 0.76,
    Volatility: 0.24,
}
```

### Orange Pressure Volatile

```go
overlayhud.ViewModel{
    MatchActive: true,
    HasData: true,
    IsStale: false,
    BlueShare: 0.31,
    OrangeShare: 0.69,
    DisplayState: "volatile",
    StateLabel: "VOLATILE",
    Confidence: 0.67,
    Volatility: 0.86,
}
```

### Stale Last Known

```go
overlayhud.ViewModel{
    MatchActive: true,
    HasData: true,
    IsStale: true,
    BlueShare: 0.64,
    OrangeShare: 0.36,
    DisplayState: "stale",
    StateLabel: "STALE",
    Confidence: 0.51,
    Volatility: 0.17,
}
```

## Future Implementation Phases

1. Restore MomentumControlWheel geometry: `1024` viewBox, `96` pill segments,
   `120` ticks, and old group IDs/classes.
2. Restore CSS/token/motion parity: aura layers, seam effects, sparks,
   pressure/control/volatile/dominant styling, reduced-motion behavior.
3. Add in-game visual validation screenshots and performance notes.
4. Add Overlay Lab or visual controls in a separate frontend-only PR, if still
   needed.

## Acceptance Criteria

- Spec documents the MomentumControlWheel geometry target.
- Spec preserves the architecture chain from Momentum Engine to future renderer.
- Spec defines exact SVG geometry, dimensions, layer order, group names, segment
  rules, tick rules, display tokens, and fixture states.
- Spec uses safe Momentum terminology.
- Spec documents stale, no-data, inactive, confidence, and volatility display
  behavior.
- Spec does not add code, routes, assets, registration, dependencies, or
  persistence changes.
