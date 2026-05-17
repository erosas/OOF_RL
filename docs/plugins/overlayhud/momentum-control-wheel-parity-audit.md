# MomentumControlWheel Parity Audit

Status: PR #80 candidate audit

Scope: documentation and classification only. This audit compares old PR #47
MomentumControlWheel behavior against the rebuilt Momentum Overlay path so the
next implementation PR can target the right layer without guessing.

## Purpose

This document answers:

```text
Did we fully rebuild old PR #47 MomentumControlWheel behavior from the ground
up, and if not, exactly what is missing, where does it belong, and what PR
should fix it?
```

The short answer is:

```text
The rebuilt architecture and core wheel geometry are strong.
Full visual, display-contract, and engine-output parity are not complete yet.
```

That is expected. Old PR #47 bundled engine math, event normalization, widget
rendering, native HUD lifecycle, Overlay Lab controls, configuration, and
performance tooling. The rebuild intentionally separated those responsibilities.

## Explicit Non-Goals

- Do not tune Momentum Engine math in this PR.
- Do not retune event weights or signal calibration in this PR.
- Do not redesign the wheel visually.
- Do not add Overlay Lab, settings, config, presets, or plugin manager work.
- Do not change native shell, F8/F9, or dormancy lifecycle behavior.
- Do not mutate DB/schema, Session, History, Live, saved match, or replay data.
- Do not add frontend frameworks, dependencies, React, TypeScript, Vite, or
  Webpack.
- Do not touch unrelated Timeline/assets files.

## Sources Reviewed

### Old PR #47 Sources

All requested old PR #47 sources were available on
`feature/momentum-overlay-hud-v1`:

| Source | Status | Notes |
| --- | --- | --- |
| `MomentumControlWheel.tokens.json` | Found | ViewBox, colors, opacity, radii, typography, segment counts. |
| `MomentumControlWheel.states.json` | Found | Neutral, pressure, control, volatile, dominant fixtures. |
| `MomentumControlWheel.layer-manifest.md` | Found | Group order, layer IDs, center/text rules. |
| `MomentumControlWheel.motion.md` | Found | Aura, shimmer, contested flicker, event reactions. |
| `engine-overlay-boundary.md` | Found | Engine/overlay separation goal and follow-up refactors. |
| `dev-sandbox/README.md` | Found | Old visual sandbox context. |
| `dev-sandbox/momentum-variant-b-preview.html` | Found | Old visual preview artifact. |
| `internal/plugins/overlayhud/view.js` | Found | Old widget, adapter, display state, config, runtime DOM updates. |
| `momentum-control-wheel.css` | Found | Old CSS/token/motion/performance behavior. |
| `momentum-control-wheel.test.js` | Found | Old protected behavior and config/test fixtures. |
| `internal/momentum/config.go` | Found | Old weights, thresholds, decay, pulse settings. |
| `internal/momentum/types.go` | Found | Old flow output, overlay fields, pulse/debug data. |
| `internal/momentum/engine.go` | Found | Old pressure/control engine behavior. |
| `internal/momentum/update_state_delta.go` | Found | Old UpdateState fallback event derivation. |
| `internal/momentum/normalizer.go` | Found | Old raw event normalizer. |
| `internal/momentum/*_test.go` | Found | Old engine/normalizer/update-state-delta tests. |

### Current Rebuilt Sources

| Source | Status | Notes |
| --- | --- | --- |
| `momentum-overlay-svg-spec.md` | Found | Rebuilt display spec, now updated toward old geometry. |
| `momentum-overlay-dormancy-blueprint.md` | Found | Dormancy/lifecycle blueprint. |
| `momentum-overlay-ingame-validation.md` | Found | PR #79 validation and seam parity notes. |
| `internal/momentum/types.go` | Found | Runtime-only typed-event MomentumState. |
| `internal/momentum/engine.go` | Found | Bounded typed-event engine. |
| `internal/momentum/runtime.go` | Found | Thread-safe service/status/reset. |
| `internal/momentum/wiring.go` | Found | Typed event bus wiring. |
| `internal/plugins/overlayhud/viewmodel.go` | Found | SnapshotProvider to display ViewModel mapping. |
| `internal/plugins/overlayhud/rendermodel.go` | Found | ViewModel to RenderModel geometry/state mapping. |
| `internal/plugins/overlayhud/svg_renderer.go` | Found | Static SVG renderer. |
| `internal/plugins/overlayhud/display_adapter.go` | Found | Request-driven SVG/HTML adapter and CSS wrapper. |
| `internal/plugins/overlayhud/preview_route.go` | Found | Internal preview route. |
| `internal/plugins/overlayhud/launch_route.go` | Found | Manual launch route. |
| `internal/plugins/overlayhud/control_route.go` | Missing on this branch | Prompt listed it, but current branch only has preview/SVG/launch routes. |
| `internal/plugins/overlayhud/*_test.go` | Found | Rebuilt Go tests. |

## Sources Unavailable Or Assumptions

- No old rendered screenshots were inspected in this audit. Visual conclusions
  are based on old docs, CSS, JS, tests, and recent manual validation notes.
- Current `control_route.go` is not present on this branch. The native shell has
  `Show`, `Hide`, and `Dormant` primitives, but route-level controlled dormancy
  is not auditable here.
- Old PR #47 frontend behavior was dynamic DOM mutation in `view.js`. The rebuilt
  path is static SVG refreshed by a route. Motion parity must account for that
  architectural difference.
- Old PR #47 included Overlay Lab/config/presets. Those are intentionally out of
  scope for the current rebuild line.

## Summary Verdict

| Area | Verdict |
| --- | --- |
| Architecture | Mostly matched and improved. Responsibilities are now better separated than old PR #47. |
| Data contract | Partial. Current runtime lacks several old display-facing output fields. |
| State parity | Partial. Core states exist, dominant and old confidence semantics are missing. |
| Geometry | Mostly matched after PR #77 and PR #79 seam fix. |
| Visual styling | Partial. Major look restored, detailed token/CSS parity missing. |
| Motion/response | Partial. Some motion restored, old response/config/event effects missing. |
| Engine-output behavior | Partial. Old pressure/control model was rebuilt in bounded form, but output semantics differ. |
| Tests | Partial. Current tests cover architecture/renderer basics; old config/motion/display tests are not fully represented. |

## Architecture Parity

| Old PR #47 Behavior | Current Rebuild | Status | Category | Severity | Recommended PR | Reason |
| --- | --- | --- | --- | --- | --- | --- |
| Engine and overlay were bundled in one large PR but docs required engines to run without overlays. | Momentum Engine, service, wiring, SnapshotProvider, ViewModel, renderer, and shell are separated. | Matched/improved | intentionally deferred | polish | Defer | Rebuild follows the intended boundary better than old PR #47. |
| Overlay HUD must not mutate Momentum Engine or match data. | Overlay consumes `momentum.SnapshotProvider` read-only. | Matched | intentionally deferred | polish | Defer | Correct display-only boundary. |
| Disabled overlay should produce zero GUI/rendering overhead. | Preview/display adapter renders on request; manual shell is explicit. Full lifecycle/toggle model is not complete on this branch. | Partial | lifecycle/control | important | Future lifecycle | Manual routes are controlled, but route-level control is absent here. |
| Old native HUD visibility and dormancy were mixed into `view.js`/native overlay behavior. | Manual shell primitives are separate from Overlay HUD display logic. | Partial/improved | lifecycle/control | important | Future lifecycle | Separation is correct, but full user-facing lifecycle is not audited here. |

## Data Contract Parity

| Contract Item | Old PR #47 | Current Rebuild | Status | Category | Severity | Recommended PR | Reason |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Engine output shape | `MomentumFlowOutput` with `state`, `blue`, `orange`, `confidence`, `volatility`, `dominantTeam`, `overlay`, `debug`. | `MomentumState` with team `Pressure`, `MomentumInfluence`, `ContestInvolvement`, `EventDerivedControl`, `Confidence`, `Volatility`, last event. | Partial | display-contract/mapping | blocker | PR #81 | Current output lacks old overlay/debug display envelope. |
| Blue/orange share source | Old wheel preferred `overlay.momentumBarBluePercent`; fallback to `controlShare`; flow bar used pressure/share. | Current ViewModel uses `MomentumInfluence` only. | Partial | display-contract/mapping | blocker | PR #81 | This may explain "responds but not like old #47" without requiring engine math first. |
| `pressureShare` vs `controlShare` | Old output exposed both. Wheel adapter preferred control share when bar percent was absent. | Current state exposes `Pressure` and `EventDerivedControl`, but ViewModel ignores both for share. | Partial | display-contract/mapping | blocker | PR #81 | Need explicit display mapping decision before calibration. |
| Confidence numeric | Old engine output numeric confidence. | Current team confidence averaged into ViewModel. | Partial | display-contract/mapping | important | PR #81 | Averaging team confidence may not match old global confidence behavior. |
| Confidence buckets | Old wheel used `low`, `medium`, `high`, `max` with thresholds `0.36`, `0.62`, `0.82`. | Current RenderModel uses `is-low`, `is-medium`, `is-high`; no `max` bucket. | Partial | display-contract/mapping | important | PR #81 or visual polish | Bucket semantics affect labels and dominant state. |
| Volatility | Old global `volatility`, scaled by response config. | Current team volatility averaged into ViewModel. | Partial | engine-calibration | important | PR #81 | May change volatile feel/state activation. |
| Recent event pulse | Old `OverlayOutput.Pulse`, `PulseTeam`, debug `LastStrongEvent`, `recentEventEnergy`, `recentEventTeam`, `recentEventType`. | Current state stores only last event fields; no pulse energy/decay fields in ViewModel/RenderModel. | Missing | display-contract/mapping | important | PR #81 | Affects sparks, afterglow, goal/demo/shot response. |
| Match clock/timer | Old adapter derived timer from `display.matchClock`, debug events, or match clock fallback. | Current center text remains `--:--`. | Missing | display-contract/mapping | important | PR #81 | Timer readability exists, but live timer parity is missing. |
| Stale/no-data/inactive | Old hidden/dormant/native visibility plus data modes in JS. | Current ViewModel has `IsStale`, `HasData`, `MatchActive`. | Partial | display-contract/mapping | important | PR #81 | Current semantics are simpler but useful. |

## State Parity

| State | Old Behavior | Current Behavior | Status | Category | Severity | Recommended PR | Reason |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `neutral` | 50/50, top seam, balanced glow, low contest pulse, label `NEUTRAL`. | 50/50 fallback, top seam after PR #79, label `NEUTRAL`. | Matched | intentionally deferred | polish | Defer | Good enough for current rebuild. |
| `blue-pressure` | Blue 62-ish fixture, stronger blue aura, blue wash, orange readable. | Supported by ViewModel/RenderModel/CSS. | Partial | visual-only | important | Future visual polish | Main state exists; old token values and response config are incomplete. |
| `orange-pressure` | Orange pressure mirror. | Supported. | Partial | visual-only | important | Future visual polish | Same as blue pressure. |
| `blue-control` | Blue 72-ish fixture, stronger aura/streaks, center control opacity. | Supported. | Partial | visual-only | important | Future visual polish | Main state exists; old dominant/control effects not complete. |
| `orange-control` | Orange control mirror. | Supported. | Partial | visual-only | important | Future visual polish | Same as blue control. |
| `volatile` | Old display label `CONTESTED`, purple seam flicker, alternating sparks, shockwave. | Current label is `VOLATILE`; purple seam/flicker exists but simplified. | Partial | display-contract/mapping | important | PR #81 or visual polish | Label/response semantics differ; visual effects are simplified. |
| `dominant-blue` | Near-full blue, orange sliver, max confidence, strong blue bloom, dominant sparks. | Missing as explicit display state. | Missing | display-contract/mapping | blocker | PR #81 | Old state requires mapping plus visual classes. |
| `dominant-orange` | Near-full orange mirror. | Missing as explicit display state. | Missing | display-contract/mapping | blocker | PR #81 | Same as dominant-blue. |
| `stale` | Old native/data behavior was mixed with JS runtime state. | Current explicit stale state and overlay text. | Partial | display-contract/mapping | important | PR #81 | Needs behavior decision: show last-known vs dim vs freeze. |
| `no-data` | Old startup/default/no signal fallback. | Current explicit no-data state. | Matched/partial | display-contract/mapping | polish | PR #81 | Adequate, but old label/visual treatment differs. |
| `inactive` | Old dormant/hidden logic used native visibility/perf flags. | Current MatchActive false maps inactive. | Partial | lifecycle/control | important | Future lifecycle | Lifecycle semantics not fully represented on this branch. |

## Geometry Parity

| Geometry Item | Old PR #47 | Current Rebuild | Status | Category | Severity | Recommended PR | Reason |
| --- | --- | --- | --- | --- | --- | --- | --- |
| ViewBox | `0 0 1024 1024` | `0 0 1024 1024` | Matched | intentionally deferred | polish | Defer | Restored in PR #77. |
| Center | `512,512` | `512,512` | Matched | intentionally deferred | polish | Defer | Restored. |
| Segments | 96 pill rects; old JS used `x=505`, `y=108`, `width=14`, `height=108`. | 96 pill rects with same dimensions. | Matched | intentionally deferred | polish | Defer | Current implementation follows old JS/test source. |
| Ticks | 120 inner ticks; old tests expected inner tick group. | 120 ticks rendered. | Matched | intentionally deferred | polish | Defer | Current tests cover count. |
| Seam convention | Old wheel angle `0deg` was 12 o'clock via `polar(angle - 90)`. | PR #79 fixed seam/aura/front-line orientation. | Matched | intentionally deferred | polish | Defer | Manual validation confirmed. |
| Ownership origin | Old blue starts at 180deg and fills clockwise by blue percent. | Current `segmentOwner` matches that rule. | Matched | intentionally deferred | polish | Defer | Covered by RenderModel/renderer tests. |
| Group IDs/layer order | Old manifest had detailed group order. | Major groups present; some detailed child groups empty/simplified. | Partial | visual-only | important | Future visual polish | Stable hooks exist, but detailed layers are incomplete. |
| Center layout | Old time y=448, state y=554, confidence y=616 in JS; later tests validated confidence. | Current time y=506, state y=570, no confidence value. | Partial | visual-only | important | Future visual polish | Current layout keeps readability but differs from old. |
| Confidence label/value | Old `text-confidence-label`, `text-confidence-value`. | `hud-confidence-label` empty; no value text. | Missing | visual-only | important | Future visual polish | Explicit old behavior missing. |
| OOF badge | Old badge shell/fill/border/text. | Empty `oof-badge` group. | Missing | visual-only | polish | Future visual polish | Branding only, not blocker. |

## Visual Styling Parity

| Styling Item | Old PR #47 | Current Rebuild | Status | Category | Severity | Recommended PR | Reason |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Color tokens | Rich `--mcw-*` variables and token JSON. | Smaller CSS token subset in HTML wrapper. | Partial | visual-only | important | Future visual polish | Enough for current look, not full parity. |
| Segment gradients | Old blue/orange gradient fills with stroke/filter tuning. | Solid fills and simplified shadows. | Partial | visual-only | important | Future visual polish | Visual depth differs. |
| Inactive segments | Old opacity/brightness configurable. | Static simplified opacity. | Partial | visual-only | polish | Future visual polish | Not a blocker. |
| Frame styling | Old frame panels/bolts/highlights. | Simplified circles only. | Partial | visual-only | polish | Future visual polish | Looks less premium than old PR #47. |
| Aura styling | Old base/peak per state and pulse timing. | Simplified aura opacity variables. | Partial | visual-only | important | Future visual polish | Affects feel. |
| Center washes | Old base/reactive/blue/orange/purple/honeycomb. | Blue/orange/purple circles, no honeycomb/reactive layer. | Partial | visual-only | important | Future visual polish | Center response differs. |
| Purple volatile treatment | Old volatile flicker, contest glow, sparks. | Basic flicker and purple glow. | Partial | visual-only | important | Future visual polish | Volatile state likely feels under-modeled. |
| Seam cap styling | Old cap glow with seam flare/flicker variables. | Basic seam cap with volatility-dependent glow. | Partial | visual-only | important | Future visual polish | PR #79 fixed placement, not all styling. |
| Confidence styling | Old label/value and confidence text brightness. | Ring/class only. | Missing/partial | visual-only | important | Future visual polish | Missing center confidence text behavior. |
| Text styling | Old Inter/Sora/Space Grotesk stack and strict text roles. | Segoe/Arial, timer/state only. | Partial | visual-only | polish | Future visual polish | Readable but not exact. |
| Badge styling | Old OOF badge visual layer. | Empty group. | Missing | visual-only | polish | Future visual polish | Branding only. |
| No-data/stale/inactive | Old visibility/perf/native states plus widget config. | Explicit simple classes and dimming. | Partial | display-contract/mapping | important | PR #81 | Needs clear semantics. |
| Dominant styling | Old dominant classes, bloom, pulse, sparks. | Missing explicit dominant state/classes. | Missing | display-contract/mapping | blocker | PR #81 | Requires state mapping first. |

## Motion And Response Parity

| Motion Item | Old PR #47 | Current Rebuild | Status | Category | Severity | Recommended PR | Reason |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Aura breathing | Old `mcw-aura-*` animations and state pulse ms. | Simplified CSS transitions/limited animation. | Partial | visual-only | important | Future visual polish | Feel differs. |
| Segment shimmer | Old segment filters/transitions and active side shimmer. | Basic transitions. | Partial | visual-only | polish | Future visual polish | Not calibration. |
| Contested flare pulse | Old flicker/pulse tied to volatility/config. | Basic flicker on volatile. | Partial | visual-only | important | Future visual polish | State response less rich. |
| Volatility flicker | Old volatile flicker and purple/white seam sparks. | Simplified volatile animation. | Partial | visual-only | important | Future visual polish | Affects perceived live response. |
| Sparks | Old deterministic spark fixtures with roles: neutral, pressure, control, dominant, volatile. | Only a few static spark nodes/roles. | Partial | visual-only | important | Future visual polish | Old tests expected many roles/counts. |
| Streaks | Old pressure/control/dominant streak behavior. | Static simple blue/orange streaks. | Partial | visual-only | polish | Future visual polish | Visual-only. |
| Shockwave/recent events | Old goal/shot/save/demo pulse and afterglow response. | Missing pulse/event energy model in render path. | Missing | display-contract/mapping | important | PR #81 | Needs recent-event data contract. |
| Goal/major event pulse | Old `PulseGoalBurst` and full-ring pulse behavior. | Missing in ViewModel/RenderModel. | Missing | display-contract/mapping | important | PR #81 | Data not currently exposed. |
| Confidence change response | Old confidence changed rim/text brightness. | Ring class only; no text. | Partial | visual-only | polish | Future visual polish | Depends on bucket mapping. |
| Reduced-motion | Old reduced-motion suppressed sparks/aura/event response without corrupting stored prefs. | CSS media query disables animations/filter partially. | Partial | visual-only | important | Future visual polish | No stored/effective config split. |
| Performance mode | Old performance-safe config disabled sparks, filters, aura, center reactive, etc. | No full performance mode contract in current renderer. | Missing/partial | visual-only | important | Future visual polish | Useful later, not PR #81 blocker. |
| Runtime update model | Old in-place DOM updates preserved element identity and transitions. | Static SVG refreshed every 250ms in HTML wrapper. | Partial | display-contract/mapping | important | PR #81 | May affect motion feel and should be tested before calibration. |

## Engine-Output And Display Behavior Parity

| Behavior | Old PR #47 | Current Rebuild | Status | Category | Severity | Recommended PR | Reason |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Event source model | Old normalized raw events plus UpdateState deltas. | Typed `oofevents.GameActionEvent` from PR #51. | Intentionally different | intentionally deferred | polish | Defer | This was a deliberate rebuild decision. |
| UpdateState fallback | Old `UpdateStateDeltaNormalizer` derived shots/saves/demos/assists/goals. | Not present in Momentum package; PR #51 typed events handle StatFeed/BallHit foundation. | Partial | engine-calibration | important | PR #81 | Need decide whether typed events fully replace fallback needs. |
| Ball-hit chains | Old chain window and increasing control/pressure bonuses. | Current chain window and bounded bonuses exist. | Partial | engine-calibration | important | PR #81 | Numeric scales differ significantly. |
| Shot/save/goal/demo weights | Old unbounded-ish control/pressure values with decay. | Current bounded 0..1 signal deltas. | Partial | engine-calibration | blocker | PR #81 | Expected old feel may require calibration, not visual fixes. |
| Decay | Old per-second decays per control/pressure/volatility/confidence. | Current fixed per-event decay. | Partial | engine-calibration | blocker | PR #81 | Major response-feel difference. |
| Classification thresholds | Old engine thresholds plus display adapter thresholds/holds. | Current ViewModel share thresholds: 58 pressure, 70 control, volatility with low confidence. | Partial | engine-calibration | blocker | PR #81 | Could drive incorrect pressure/control frequency. |
| Display state hold/transition | Old display state min/max hold and contested flip behavior. | Not represented. | Missing | display-contract/mapping | blocker | PR #81 | Likely important to "feels like old #47." |
| Recent event energy | Old pulse/debug decay into event energy. | Not represented. | Missing | display-contract/mapping | important | PR #81 | Needed for sparks/afterglow. |
| Confidence bucket logic | Old numeric to `low/medium/high/max`. | Current numeric class only; no max. | Partial | display-contract/mapping | important | PR #81 | Needed before dominant states. |
| Volatility scaling | Old response config scaled volatility. | Current averaged team volatility and thresholds. | Partial | engine-calibration | important | PR #81 | Affects volatile state. |
| Share normalization | Old normalized percent and fallback logic. | Current shared `shares()` normalizes MomentumInfluence. | Partial | display-contract/mapping | blocker | PR #81 | Need determine whether control or pressure should drive wheel. |

## Test Parity

| Test Area | Old Protection | Current Protection | Status | Category | Severity | Recommended PR | Reason |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Geometry counts | Old JS tests covered 96 inactive/blue/orange/cap segment refs and 120 ticks. | Current tests cover 96 inactive rendered segments and 120 ticks. | Partial | visual-only | polish | Future visual polish | Current does not mirror old parallel blue/orange hidden refs exactly. |
| Seam/orientation | Old tests checked seam movement and contest aura not fixed at top. | PR #79 tests neutral/blue/orange seam orientation. | Matched | intentionally deferred | polish | Defer | Good. |
| Config sanitization | Old tests covered extensive visual/response config clamping. | Current has no equivalent config system. | Missing | intentionally deferred | polish | Future visual polish | Config intentionally deferred. |
| Reduced/performance | Old tests covered stored prefs vs effective render suppression. | Current tests mostly check HTML contains media query/classes. | Partial | visual-only | important | Future visual polish | Not PR #81 unless data affects signal. |
| Dominant states | Old tests checked dominant blue aura and dominant spark roles. | Current tests do not cover dominant because state is missing. | Missing | display-contract/mapping | blocker | PR #81 | Needs display mapping. |
| Recent event response | Old tests covered recent event energy and spark roles. | Missing. | Missing | display-contract/mapping | important | PR #81 | Needs data contract. |
| Engine weighting | Old engine tests covered old normalized events/weights/state output. | Current engine tests cover bounded typed-event behavior, epic saves, own goals, lifecycle. | Partial | engine-calibration | blocker | PR #81 | Calibration needs fixture parity tests. |
| UpdateState fallback | Old tests protected stat delta normalization. | Not represented. | Missing/intentional | engine-calibration | important | PR #81 | Need decide if typed event foundation removes need. |

## Gap Classification

| Gap | Primary Category | Severity | Recommended PR | Reason |
| --- | --- | --- | --- | --- |
| Dominant blue/orange state mapping | display-contract/mapping | blocker | PR #81 | Requires confidence/share bucket semantics, not just CSS. |
| Wheel share source: MomentumInfluence vs control/pressure shares | display-contract/mapping | blocker | PR #81 | Likely core cause of response mismatch. |
| Old display hold/transition behavior | display-contract/mapping | blocker | PR #81 | Prevents flickery or wrong state transitions. |
| Old per-second decay vs current per-event decay | engine-calibration | blocker | PR #81 | Major response-feel difference. |
| Old weight scale vs bounded current deltas | engine-calibration | blocker | PR #81 | Need deliberate calibration tests. |
| Recent event pulse/energy fields | display-contract/mapping | important | PR #81 | Needed for event response visuals. |
| Match clock/timer | display-contract/mapping | important | PR #81 | Old widget showed live match clock where available. |
| Confidence `low/medium/high/max` buckets | display-contract/mapping | important | PR #81 | Needed for labels and dominant state. |
| Volatile label old `CONTESTED` vs current `VOLATILE` | display-contract/mapping | important | PR #81 | Safe label choice needs decision. |
| Detailed spark/streak role parity | visual-only | important | Future visual polish | Visual fidelity, not engine math. |
| Electric cracks/pointer/center reactive layer | visual-only | important | Future visual polish | Missing old detailed layers. |
| Token/config/performance system | visual-only | important | Future visual polish | Old config surface intentionally deferred. |
| OOF badge | visual-only | polish | Future visual polish | Branding-only. |
| Control route absent on this branch | lifecycle/control | important | Future lifecycle | Not a MomentumControlWheel parity blocker. |
| Overlay Lab/presets | intentionally deferred | polish | Defer | Explicitly out of rebuild scope. |

## Matched Areas

- Engine and overlay boundaries are now cleaner than old PR #47.
- Overlay HUD consumes read-only Momentum state.
- Rebuilt geometry matches the old wheel viewBox, center, 96 segments, 120
  ticks, ownership origin, and seam convention.
- Manual preview/launch path can render the wheel in-game.
- Pressure/control/volatile classes and major visual hooks exist.
- No DB/session/history/live/replay mutation was introduced by the rebuild.

## Partial Areas

- Display contract preserves basic shares/confidence/volatility but not old
  overlay/debug/pulse fields.
- State model supports core states but not dominant states or old display holds.
- Styling restores the broad look but not old token/config/detail parity.
- Motion restores some life but not old recent-event, spark, shimmer, and
  performance-safe behavior.
- Engine rebuilt old weighting concepts into bounded typed-event fields, but
  response scale and decay are different.

## Missing Areas

- `dominant-blue` and `dominant-orange` display states.
- Confidence label/value text and `max` bucket.
- Recent event energy/team/type data into ViewModel/RenderModel.
- Match clock/timer data.
- Old display hold/transition/flip behavior.
- Old spark/streak role counts and deterministic fixture coverage.
- Full performance/reduced-motion render-cost controls.
- Old UpdateState delta fallback behavior, if still needed after PR #51 typed
  event foundation.

## Intentionally Deferred Areas

- Overlay Lab.
- Settings/config/preset UI.
- Plugin manager/lifecycle architecture.
- Full user-facing overlay manager and configurable hotkeys.
- Large frontend split or framework migration.
- Persisting Momentum state.

## Recommended PR #81 Scope

Title:

```text
Momentum Signal/Display Calibration
```

Branch:

```text
feature/momentum-signal-display-calibration
```

PR #81 should implement the minimum calibration/data-contract work needed to
make the rebuilt overlay respond like old PR #47:

1. Add/restore display-contract fields needed by the wheel:
   - share source decision: control, pressure, or blended momentum
   - confidence bucket label: low/medium/high/max
   - dominant display state eligibility
   - recent event energy/team/type
   - match clock if currently available from typed events/status
2. Add state transition/hold behavior if it belongs in the display layer.
3. Calibrate engine output only after display-contract parity tests exist.
4. Add old-behavior fixture tests before changing weights.
5. Keep all changes runtime-only and non-persistent.

PR #81 should not add Overlay Lab, settings/config, plugin manager, lifecycle
architecture, DB/schema changes, or session/history/live/replay mutation.

## Recommended Future Visual Polish Scope

After PR #81 clarifies the data contract and calibration:

- Restore dominant-state visual styling.
- Restore confidence label/value text.
- Restore old spark/streak roles where performance-safe.
- Add center reactive layer, honeycomb texture, and optional badge.
- Expand reduced-motion/performance-safe CSS.
- Consider whether static SVG refresh needs stable in-place updates for better
  motion parity.

## Unresolved Questions

1. Should the wheel display share be driven by `MomentumInfluence`,
   `EventDerivedControl`, `Pressure`, or a blended output matching old
   `momentumBarBluePercent`?
2. Should the state label use old `CONTESTED` for volatile, or keep the safe
   rebuilt label `VOLATILE`?
3. Are typed PR #51 events sufficient, or does UpdateState delta fallback still
   need a modern equivalent?
4. Should display hold/transition behavior live in `overlayhud.ViewModel`
   mapping, `RenderModel`, or Momentum Service status?
5. Should match clock be added to Momentum snapshots, or should Overlay HUD read
   it from a separate live status provider later?
