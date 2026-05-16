# MomentumControlWheel Layer Manifest

Logged: May 13, 2026, 12:24 AM ET

Scope: UI only. This manifest defines the SVG build target for the Momentum Control Wheel. Do not add match-state logic, do not mutate Live, History, Session, saved matches, replay capture, the database, or Momentum Engine data.

## Component

Name: `MomentumControlWheel`

ViewBox: `0 0 1024 1024`

Core props:

```ts
type MomentumControlWheelProps = {
  time: string;
  bluePercent: number;
  orangePercent: number;
  state:
    | "neutral"
    | "blue-pressure"
    | "orange-pressure"
    | "blue-control"
    | "orange-control"
    | "volatile"
    | "dominant-blue"
    | "dominant-orange";
  confidence: "low" | "medium" | "high" | "max";
  volatility: number;
  showOOFBadge: boolean;
};
```

## Implementation Rule

Do not redesign the component. Do not change layout, colors, typography, or layer order unless the manifest/tokens/states are updated first. The SVG layer manifest, token file, state file, and motion file are the source of truth.

## SVG Group Order

Render groups in this exact order:

```text
svg#momentum-control-wheel
  defs
  g#background
  g#outer-aura
  g#outer-energy-streaks
  g#outer-sparks
  g#outer-mechanical-frame
  g#segment-ring-underlay
  g#segment-ring-active
  g#segment-ring-bevels
  g#inner-tick-ring
  g#center-disc
  g#center-color-washes
  g#center-texture
  g#center-rim
  g#contested-front-line
  g#text-layer
  g#oof-badge
  g#debug-overlays
```

## 01 - Root SVG

| Layer ID | SVG Type | Purpose |
| --- | --- | --- |
| `momentum-wheel-root` | `<svg>` | Main component container |
| `defs` | `<defs>` | Gradients, blur filters, glow filters, masks |

## 02 - Background / Safe Plate

| Layer ID | SVG Type | Description |
| --- | --- | --- |
| `bg-transparent-hitbox` | `<rect>` | Invisible full-size bounding box |
| `bg-vignette` | `<circle>` / `<radialGradient>` | Soft dark falloff behind wheel |
| `bg-subtle-noise` | SVG filter | Very faint HUD texture |
| `bg-radial-shadow` | `<circle>` | Outer shadow grounding the wheel |

Rules:
- Background must be optional.
- In actual overlay mode, background opacity should be reduced or disabled.
- Do not bake a black square behind the component unless debug mode is enabled.

## 03 - Outer Aura / Energy Field

| Layer ID | SVG Type | Description |
| --- | --- | --- |
| `outer-aura-blue` | Arc path / stroked circle mask | Soft blue glow around left side |
| `outer-aura-orange` | Arc path / stroked circle mask | Soft orange glow around right side |
| `outer-aura-purple-contest` | Small blurred circle/path | Purple-white glow at contested boundary |
| `outer-energy-streaks-blue` | `<line>` / `<path>` group | Short blue motion streaks outside ring |
| `outer-energy-streaks-orange` | `<line>` / `<path>` group | Short orange motion streaks outside ring |
| `outer-sparks-blue` | Small circles/lines | Blue particles |
| `outer-sparks-orange` | Small circles/lines | Orange particles |

Rules:
- Keep these groups separate and toggleable.
- Accessibility/performance modes may lower or disable streaks and sparks.

## 04 - Outer Mechanical Frame

| Layer ID | SVG Type | Description |
| --- | --- | --- |
| `outer-frame-base` | `<circle>` stroke | Dark metal circular frame |
| `outer-frame-highlight` | `<circle>` stroke | Thin bright rim highlight |
| `outer-frame-shadow` | `<circle>` stroke | Inner dark separation line |
| `outer-frame-panels-top` | `<path>` | Small mechanical panel at top |
| `outer-frame-panels-bottom` | `<path>` | Small mechanical panel at bottom |
| `outer-frame-panels-left` | `<path>` | Small mechanical panel at left |
| `outer-frame-panels-right` | `<path>` | Small mechanical panel at right |
| `outer-frame-bolts` | `<circle>` group | Optional small anchor dots |

Visual target: this layer borrows from the premium Rocket League boost gauge look.

## 05 - Segmented Momentum Ring

| Layer ID | SVG Type | Description |
| --- | --- | --- |
| `segment-ring-underlay` | 96 rounded rects or paths | Dark inactive segment bed |
| `segment-ring-blue-active` | Rounded rect group | Blue-owned momentum segments |
| `segment-ring-orange-active` | Rounded rect group | Orange-owned momentum segments |
| `segment-ring-neutral-caps` | Rounded rect group | White/purple contested seam segments |
| `segment-ring-bevels` | Thin white overlays | Small highlight on each segment |
| `segment-ring-inner-shadow` | Masked stroke | Adds depth inside segment ring |
| `segment-ring-outer-highlight` | Masked stroke | Bright outer rim edge |

Segment construction:

```text
Segment count: 96
Segment width: 10px
Segment height: 46px
Segment radius: 999px
Ring radius: 365px
Center point: 512, 512
Rotation origin: 512, 512
angle = index * (360 / 96)
x = centerX
y = centerY - ringRadius
transform = rotate(angle, centerX, centerY)
```

Use individual segment primitives rather than `stroke-dasharray` so the ring matches the pill-like boost gauge art direction.

## 06 - Inner Tick Ring

| Layer ID | SVG Type | Description |
| --- | --- | --- |
| `inner-tick-ring-base` | 120 thin lines | Fine pressure/timing ticks |
| `inner-tick-ring-blue` | Line group | Blue-side tick tint |
| `inner-tick-ring-orange` | Line group | Orange-side tick tint |
| `inner-tick-ring-muted` | Line group | Low-opacity inactive ticks |
| `inner-crosshair-lines` | `<line>` group | Subtle vertical/horizontal alignment guides |

## 07 - Contested Front Line

| Layer ID | SVG Type | Description |
| --- | --- | --- |
| `contest-top-core` | `<circle>` | White-hot flare at contested seam |
| `contest-top-purple-glow` | `<circle>` with blur | Purple bloom around flare |
| `contest-top-vertical-beam` | `<line>` | Thin beam through center |
| `contest-top-electric-cracks` | `<path>` group | Small jagged purple energy paths |
| `contest-bottom-seam` | `<line>` / `<circle>` | Smaller seam at 6 o'clock |
| `contest-left-pointer` | `<circle>` + line | Optional moving pressure marker |

State behavior:

| State | Front Line Behavior |
| --- | --- |
| Neutral | Centered at top, balanced glow |
| Blue Pressure | Shifts slightly toward orange side |
| Orange Pressure | Shifts slightly toward blue side |
| Volatile | Jitter/flicker, purple intensifies |
| Dominant | Opposing seam becomes small and dim |

## 08 - Center Disc

| Layer ID | SVG Type | Description |
| --- | --- | --- |
| `center-disc-base` | `<circle>` | Dark glass center |
| `center-disc-inner-shadow` | `<circle>` stroke | Inner depth |
| `center-disc-blue-wash` | `<circle>` with radial gradient mask | Blue color fill |
| `center-disc-orange-wash` | `<circle>` with radial gradient mask | Orange color fill |
| `center-disc-purple-contest-wash` | `<path>` / radial gradient | Purple highlight near contested axis |
| `center-disc-honeycomb` | Pattern mask | Subtle hex texture |
| `center-disc-rim` | `<circle>` stroke | Thin inner rim glow |
| `center-disc-glass-highlight` | `<ellipse>` / gradient | Top glass reflection |

Center fill rules:

| State | Center Fill |
| --- | --- |
| Neutral | Subtle blue-left / orange-right gradient |
| Blue Pressure | Blue radial wash, 10-18% opacity |
| Orange Pressure | Orange radial wash, 10-18% opacity |
| Blue Control | Stronger blue wash, 18-28% opacity |
| Orange Control | Stronger orange wash, 18-28% opacity |
| Volatile | Purple pulse overlay, 12-22% opacity |
| Dominant Blue | Blue bloom, 28-40% opacity |
| Dominant Orange | Orange bloom, 28-40% opacity |

Rule: timer text must stay white and readable at all times.

## 09 - Text / Data Layer

| Layer ID | SVG Type | Description |
| --- | --- | --- |
| `text-time` | `<text>` | Main game clock |
| `text-state` | `<text>` | Current state label |
| `text-confidence-label` | `<text>` | `CONFIDENCE:` |
| `text-confidence-value` | `<text>` | `LOW`, `MEDIUM`, `HIGH`, or `MAX` |
| `text-percent-blue` | Optional `<text>` | Blue percentage |
| `text-percent-orange` | Optional `<text>` | Orange percentage |

Type rules:

| Text | Style |
| --- | --- |
| Time | Large, bold, white, center aligned |
| State | Uppercase, team-colored or cyan in neutral |
| Confidence label | Small, uppercase, muted white |
| Confidence value | Cyan / team color / purple depending on state |
| Percentages | Optional, only if layout needs them |

Font stack:

```css
font-family: "Inter", "Sora", "Space Grotesk", sans-serif;
```

## 10 - OOF Badge

| Layer ID | SVG Type | Description |
| --- | --- | --- |
| `oof-badge-shell` | Rounded rect | Small badge container |
| `oof-badge-fill` | Linear gradient | Dark glass fill |
| `oof-badge-border` | Rounded rect stroke | Thin border |
| `oof-badge-text` | `<text>` | OOF |

Rule: badge is branding only. It should never compete with the timer or state label.
