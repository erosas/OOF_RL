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

The renderer should use a fixed SVG coordinate system so layout and animation do
not shift between states.

- SVG `viewBox`: `0 0 320 320`
- Nominal rendered size: `320px x 320px`
- Minimum readable rendered size: `220px x 220px`
- Center point: `160,160`
- Outer visual radius: `150`
- Safe content radius: `132`
- Center timer region radius: `64`
- Ring stroke cap: `round`
- Ring stroke joins: `round`
- Text baseline should be centered and must not overlap the outer rings.

The SVG root should remain square. Responsive scaling may change rendered pixel
size, but it must preserve the `1:1` aspect ratio and the `0 0 320 320`
coordinate system.

## Component Dimensions

| Element | Geometry |
| --- | --- |
| Outer confidence ring | radius `146`, stroke `8` |
| Momentum share ring | radius `132`, stroke `18` |
| Volatility tick ring | radius `108`, stroke `6` |
| Center timer region | radius `64` |
| State label row | centered at `y=188` |
| Timer row | centered at `y=154` |
| Confidence caption row | centered at `y=214` |

Timer text should remain the most readable element. Other labels must shrink,
hide, or simplify before the timer becomes cramped.

## Layer Order

Future SVG output should use this back-to-front order:

1. `hud-root`
2. `hud-background`
3. `hud-grid-guides` hidden in production
4. `hud-confidence-track`
5. `hud-confidence-ring`
6. `hud-momentum-track`
7. `hud-momentum-blue`
8. `hud-momentum-orange`
9. `hud-volatility-track`
10. `hud-volatility-segments`
11. `hud-center-panel`
12. `hud-timer-text`
13. `hud-state-label`
14. `hud-confidence-label`
15. `hud-status-overlay`

The status overlay is only for inactive, stale, and no-data states. It should not
cover the timer when a timer is available.

## Editable SVG Group Names

Use stable group names so future tests, screenshots, and debug tools can target
specific layers:

- `g#hud-root`
- `g#hud-background`
- `g#hud-confidence-track`
- `g#hud-confidence-ring`
- `g#hud-momentum-track`
- `g#hud-momentum-blue`
- `g#hud-momentum-orange`
- `g#hud-volatility-track`
- `g#hud-volatility-segments`
- `g#hud-center-panel`
- `g#hud-timer-text`
- `g#hud-state-label`
- `g#hud-confidence-label`
- `g#hud-status-overlay`

Dynamic child nodes should use `data-state`, `data-team`, or `data-segment`
attributes instead of changing group IDs.

## Arc And Ring Geometry

Momentum share is represented as a two-team circular ring.

- Blue starts at the top center, angle `-90deg`.
- Orange follows blue clockwise.
- `BlueShare` and `OrangeShare` should be clamped to `0..1`.
- If shares do not sum to `1`, the renderer should normalize them for display.
- If both shares are unavailable, render the neutral `0.5 / 0.5` state.
- Use a minimum visible arc of `4deg` only for non-zero values that would
  otherwise disappear.
- Do not inflate a true zero share into a visible team advantage.

The confidence ring is a single progress ring around the outside. It should use
`ViewModel.Confidence`, clamped to `0..1`.

The volatility ring is a segmented inner ring. It should use
`ViewModel.Volatility`, clamped to `0..1`.

## Segment Count Rules

Volatility should use discrete segments so high volatility is visible without
requiring heavy animation.

- Segment count: `24`
- Segment gap: `4deg`
- Segment active threshold: `ceil(Volatility * 24)`
- Active segments should fill clockwise from `-90deg`.
- Low volatility may show zero active segments.
- Reduced-motion mode must not animate segment activation.

Segment count should remain constant across states to avoid layout churn.

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
  - `BLUE PRESSURE`
  - `ORANGE PRESSURE`
  - `SHIFTING`
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
    StateLabel: "SHIFTING",
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
    StateLabel: "BLUE PRESSURE",
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
    StateLabel: "ORANGE PRESSURE",
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
    StateLabel: "BLUE PRESSURE",
    Confidence: 0.51,
    Volatility: 0.17,
}
```

## Future Implementation Phases

1. Add package-level SVG rendering helpers that accept `overlayhud.ViewModel`
   and return static display markup or a render model.
2. Add fixture-based renderer tests using the states above.
3. Add minimal Overlay HUD runtime registration only when visible behavior
   exists.
4. Add a narrow snapshot delivery path if the renderer needs one.
5. Add reduced-motion and performance-mode integration.
6. Add Overlay Lab or visual controls in a separate frontend-only PR, if still
   needed.

## Acceptance Criteria

- Spec is documentation only.
- Spec preserves the architecture chain from Momentum Engine to future renderer.
- Spec defines exact SVG geometry, dimensions, layer order, group names, arc
  rules, segment rules, display tokens, and fixture states.
- Spec uses safe Momentum terminology.
- Spec documents stale, no-data, inactive, confidence, and volatility display
  behavior.
- Spec does not add code, routes, assets, registration, dependencies, or
  persistence changes.
