# Momentum Overlay In-Game Validation

Status: pending PR #79 validation

Scope: visual validation only. This checklist verifies the rebuilt
MomentumControlWheel in real gameplay before any signal/display calibration
work begins.

## Purpose

PR #77 restored MomentumControlWheel geometry parity.
PR #78 restored styling and motion response parity.

PR #79 validates that the rebuilt overlay is readable, coherent, and stable in
Rocket League before tuning Momentum Engine output or display thresholds.

## Non-Goals

- No Momentum Engine math changes.
- No event weighting changes.
- No display-state threshold tuning.
- No signal/display calibration.
- No native shell, F8/F9, or dormancy lifecycle changes.
- No Overlay Lab, settings, config, or preset work.
- No plugin manager or lifecycle architecture.
- No DB, schema, Session, History, Live, saved match, or replay mutation.
- No frontend framework or dependency changes.

## Validation Build

Build:

```powershell
go build -o OOF_RL_momentum_ingame_validation_test.exe .
```

Expected app route:

```text
http://localhost:8080/internal/momentum-overlay-preview
```

Expected manual overlay launch route:

```text
http://localhost:8080/internal/momentum-overlay-launch
```

## Test Metadata

Record during validation:

| Field | Value |
| --- | --- |
| Branch | `feature/momentum-overlay-ingame-validation` |
| Base | PR #78 / `feature/momentum-control-wheel-styling-motion-parity` |
| EXE | `OOF_RL_momentum_ingame_validation_test.exe` |
| Date/time | TBD |
| Tester | TBD |
| Rocket League mode | TBD |
| App data directory | `%LOCALAPPDATA%\OOF_RL` |
| Commit SHA | TBD |

## Browser Preview Checklist

- [ ] Preview route returns 200.
- [ ] SVG uses `viewBox="0 0 1024 1024"`.
- [ ] 96 pill segments are visible.
- [ ] 120 inner ticks are visible.
- [ ] Timer text is readable.
- [ ] State label is readable.
- [ ] No-data/stale/inactive states are readable.
- [ ] Motion does not obscure text.

## Native Overlay Checklist

- [ ] Manual launch route responds.
- [ ] Native overlay opens.
- [ ] Native overlay loads the preview surface.
- [ ] Overlay can appear over Rocket League.
- [ ] Overlay does not freeze the app.
- [ ] Overlay does not create repeated duplicate shells during normal validation.
- [ ] F8/F9/dormancy behavior is unchanged by this PR.

## Gameplay Visual Checklist

- [ ] Geometry remains aligned with old MomentumControlWheel layout.
- [ ] Blue and orange ownership regions are easy to distinguish.
- [ ] Pressure states are visually distinct from neutral.
- [ ] Control states are stronger than pressure states.
- [ ] Volatile state is visually distinct without constant visual noise.
- [ ] Contested seam/front line starts at 12 o'clock in neutral.
- [ ] Contested seam/front line follows the active blue/orange split during pressure/control changes.
- [ ] Contested seam/front line remains readable.
- [ ] Center wash supports state readability without hiding text.
- [ ] Timer and state text remain readable during gameplay.
- [ ] No-data, stale, and inactive states remain understandable.
- [ ] Visual effects feel alive but not distracting.

## PR #47 Parity Audit Notes

Old PR #47 treated MomentumControlWheel docs and assets as the source of
truth for display parity:

- `MomentumControlWheel.tokens.json`
- `MomentumControlWheel.states.json`
- `MomentumControlWheel.layer-manifest.md`
- `MomentumControlWheel.motion.md`
- `view.js`
- `momentum-control-wheel.css`
- `momentum-control-wheel.test.js`

Current rebuilt behavior already matches the large structural contract:

- `0 0 1024 1024` viewBox
- `512,512` center
- `96` pill segments
- `120` inner ticks
- old major SVG group IDs/classes
- blue/orange ownership around the old wheel split model
- pressure/control/volatile state classes
- display-only rendering through `SnapshotProvider -> ViewModel -> RenderModel -> SVG`

Known visual parity gaps still to track after PR #79:

- Dominant visual states (`dominant-blue`, `dominant-orange`) are not restored
  yet.
- Confidence label/value behavior from old PR #47 is not restored yet.
- Contested-front detail is simplified and does not yet include old electric
  cracks/pointer behavior.
- CSS/token coverage is still lighter than old PR #47's full configurable
  visual system.
- Runtime response uses refreshed static SVG output rather than old in-place
  `view.js` DOM mutation, so motion feel may still need parity review.

PR #79 may harden visual validation issues such as seam placement, but it must
not tune Momentum Engine math or signal/display calibration.

## Performance Observations

Record:

- App freezes or hangs:
- Rocket League frame drops:
- Overlay stutter:
- Browser preview stutter:
- Native overlay refresh issues:
- CPU/GPU concerns:
- Reduced-motion behavior, if testable:

## Known Limitation

The overlay may still not feel fully aligned with old PR #47 signal behavior.
That is intentionally deferred. PR #79 should not tune engine weighting,
display thresholds, share normalization, or calibration logic.

Future signal/display calibration should happen in a separate PR after this
visual validation pass is complete.

## Acceptance Criteria

- In-game overlay launches and remains visible.
- Geometry still matches PR #77 parity target.
- Styling/motion still matches PR #78 parity target well enough for gameplay.
- Text remains readable.
- Pressure/control/volatile states are visually distinguishable when they occur.
- No obvious lifecycle, persistence, or runtime ownership regressions.
- Any follow-up signal/display calibration questions are recorded without being
  fixed in this PR.
