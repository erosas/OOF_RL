# Momentum Timeline UX Hardening Plan

## Recommended Branch

`feature/momentum-timeline-ux-hardening`

This branch should stay stacked on the Momentum Timeline B-lite preview work until PR #48 is merged. The work in this plan assumes the hidden fixture route exists at `/internal/momentum-timeline-preview`.

## Problem Statement

Momentum Timeline B-lite proves that a fixture-backed post-match story layer can exist independently from the live Momentum Overlay. The next risk is not data integration; it is readability under real match density.

The preview needs a clearer interaction model for crowded event markers, long match timelines, selected moment hierarchy, keyboard navigation, and fallback states before any saved match, History, Session, replay, live parsing, or event-bus integration is considered.

## Phase 1 Implementation Decisions

- Same-second events should stack vertically first.
- Same-second events can use slight horizontal offsets when needed for readability.
- Marker layering priority is goals, saves/epic saves, demos, assists, shots, then momentum swings.
- Zoom remains a concept only for this phase.
- Keyboard navigation should move through all moments in chronological order.
- Fixture variants can be separate files if simple; one expanded fixture is acceptable for this phase.
- Drawer-level contribution highlights can remain for now.
- Screenshots are required for visual/UI implementation patches, not docs-only patches.

## UX Hardening Goals

- Keep the preview timeline-first and review-oriented.
- Improve marker legibility when fixture events are close together.
- Make selected segment and selected event states easier to compare.
- Make the drawer easier to scan without adding coaching-grade claims.
- Define keyboard and mouse expectations for basic review workflows.
- Explore zoom and scaling behavior with fixture data only.
- Improve empty and partial-fixture states so missing fields do not look like broken production analytics.
- Preserve safe terminology around event-derived pressure/control, pressure contribution, momentum influence, contest involvement, pressure sequence, and momentum swing.

## Explicit Non-goals

- No real match data integration.
- No History, Session, saved match, replay, live parsing, or event-bus integration.
- No DB/schema changes.
- No saved app data mutation.
- No Overlay Lab placement or Overlay HUD refactor.
- No Momentum Engine math changes.
- No new frontend framework, TypeScript, JSX, Vite, Webpack, or build pipeline.
- No dense marker data aggregation or inferred tactical summary generation.
- No Badge Engine, Journey, Career, or Replay Studio implementation.

## Marker Spacing And Readability Strategy

Treat dense marker handling as a UX/readability problem only, not a data aggregation problem yet.

Phase 1 should use fixture-side density scenarios to harden layout behavior without changing event meaning:

- Preserve one marker per source fixture event.
- Avoid combining multiple events into one derived aggregate marker in this phase.
- Add deterministic marker lanes for close timestamps.
- Add minimum visual spacing so markers do not fully overlap.
- Keep marker labels compact but inspectable through hover/focus/title text.
- Highlight the containing segment when a marker is selected.
- Prefer readable stacking over clever clustering until real event density is better understood.

Docs-only before implementation:

- Define the density thresholds that trigger lanes or spacing.
- Define whether same-second events should stack vertically, offset horizontally, or both.
- Define marker priority order for visual layering, such as goal, momentum swing, save, shot, assist, demo.

## Timeline Scaling And Zoom Strategy

The first preview uses a full-match horizontal timeline. UX hardening should explore scaling without implying replay synchronization is available.

Recommended preview-only concepts:

- Full-match overview remains the default.
- Add a local zoom control only if fixture density proves the full-match view is too cramped.
- If zoom is added, keep it frontend-only and reset on reload.
- Use elapsed time as the default display mode.
- Keep match-clock formatting as a later product decision.
- Do not add replay jump behavior beyond disabled placeholder copy.

Docs-only before implementation:

- Decide whether zoom is required for Phase 1 or should remain a written interaction concept.
- Define supported zoom levels if implemented, such as full match, 90 seconds, 45 seconds.
- Define how selection behaves when a selected event falls outside the current zoom window.

## Segment And Event Selection Improvements

The current preview supports segment and event selection. Phase 1 should make selected state more explicit and easier to navigate.

Recommended improvements:

- Keep segment click selection.
- Keep event marker click selection.
- Visually link selected event to its containing segment.
- Preserve clear selection behavior.
- Add previous and next controls for selected segments/events if the drawer becomes the primary review surface.
- Make selected segment state distinct from hover state.
- Show whether the selected drawer is segment-selected or event-selected with a stable label.

Docs-only before implementation:

- Decide whether previous/next navigates all timeline moments or only the current type, such as event-to-event or segment-to-segment.
- Decide whether marker selection should scroll or pan zoomed views in future phases.

## Drawer Hierarchy Improvements

The drawer should help the user understand the selected moment without overstating tactical meaning.

Recommended hierarchy:

1. Selection type and elapsed timestamp.
2. Short safe narrative summary.
3. Segment state and confidence.
4. Events in or near the window.
5. Match-wide player contribution references.
6. Disabled future replay placeholder.

Drawer wording should continue to use review-safe language:

- "Event-derived pressure/control increased during this pressure sequence."
- "Contest involvement increased following the save event."
- "Momentum influence shifted during this window."

Avoid:

- tactical advantage
- ball ownership
- true control
- win odds
- rotation certainty
- coaching-grade player judgment

Docs-only before implementation:

- Decide whether player contribution references in the drawer should remain minimal until segment-specific contribution data exists.
- Define drawer labels for missing fixture fields.

## Keyboard And Mouse Interaction Expectations

The preview should support basic keyboard review without turning into a full replay tool.

Recommended expectations:

- Timeline segments and event markers should be focusable.
- Enter or Space should select the focused segment or marker.
- Escape should clear selection or collapse the drawer.
- Left and Right should move between focusable moments only after a clear focus model is defined.
- Tab order should move through summary, timeline controls, markers, segments, drawer controls, and player summaries predictably.
- Hover can provide labels, but hover must not be required for understanding.

Docs-only before implementation:

- Define whether Left/Right navigation follows chronological order across segments and events together, or keeps separate segment and marker tracks.
- Define focus ring styling before adding keyboard navigation.

## Empty And Fallback State Improvements

The preview should make missing fixture fields look intentional and non-production.

Recommended fallback behavior:

- Missing match metadata should show "Fixture metadata unavailable."
- Missing segments should show a full-width empty timeline state.
- Missing events should show "No fixture event markers available."
- Missing player contribution values should show "Fixture missing" instead of zero.
- Missing segment summaries should show "Fixture segment summary unavailable."
- Missing event summaries should show "Fixture event summary unavailable."
- Missing involved players should not imply the system failed to identify real players.

Docs-only before implementation:

- Define the exact fallback copy set before changing UI code.
- Decide whether fallback fixtures should live in the same JSON fixture or a separate sparse fixture.

## Accessibility And Readability Notes

- Preserve strong contrast for blue, orange, contested, and neutral states.
- Do not rely on color alone; use labels, marker text, and drawer summaries.
- Keep event marker labels short but distinguishable.
- Ensure the selected moment drawer is announced through `aria-live` only when updates are not disruptive.
- Keep button text visible and clear.
- Maintain readable typography at small app window sizes.
- Keep reduced-motion requirements in mind if future marker or segment transitions add animation.

## Acceptance Checklist

Planning acceptance:

- Defines UX hardening goals and non-goals.
- Treats dense markers as readability only.
- Keeps all future work fixture-only until explicitly approved.
- Preserves safe terminology boundaries.
- Separates docs-only decisions from implementation tasks.

Preview implementation acceptance for a later task:

- No DB/schema changes.
- No saved app data mutation.
- No History, Session, saved match, replay, live parsing, Overlay Lab, or event-bus integration.
- Route remains hidden/internal.
- Timeline still renders from fixture data.
- Dense marker fixture remains readable.
- Segment and event selection states are distinct.
- Drawer hierarchy is easier to scan.
- Keyboard basics work if included in the approved implementation scope.
- Empty/fallback states remain clearly fixture-only.

## Suggested Implementation Phases

### Phase 1A - Docs-only Decisions

- Finalize marker density thresholds.
- Finalize same-second event behavior.
- Finalize drawer information hierarchy.
- Finalize fallback copy.
- Decide whether zoom is implemented now or held as a design concept.
- Decide keyboard navigation scope.

### Phase 1B - Fixture Expansion

- Add dense marker fixture scenarios.
- Add sparse/missing field fixture scenarios.
- Add long-duration or overtime-style fixture scenarios if useful.
- Keep all fixtures frontend-only.

### Phase 1C - Preview UX Patch

- Improve marker lanes/spacing.
- Improve selected state styling.
- Improve drawer hierarchy and fallback copy.
- Add approved keyboard interactions.
- Add zoom/scaling only if Phase 1A approves it.

### Phase 1D - Verification

- Run targeted JS checks.
- Run Go route tests.
- Run `go test ./...`.
- Run `go build`.
- Run Playwright verification for route load, marker density, selection, drawer, keyboard behavior, and Settings omission.

## Risks And Guardrails

- Dense marker handling must not become event aggregation in this phase.
- Fixture narratives must not imply true possession, ball ownership, rotation quality, tactical advantage, win odds, or hidden player intent.
- Segment-specific player contribution should remain light until the data contract supports it cleanly.
- Zoom controls can create replay-sync expectations; keep copy and controls clearly preview-only.
- Keyboard navigation can become complex quickly; implement only the smallest useful focus model.
- Do not add routes to default navigation yet.
- Do not introduce persistence to remember preview state.
- Do not reuse Overlay HUD internals unless a later task explicitly approves that integration.

## Questions Before Implementation

- Should future dense-marker behavior move beyond lane stacking once real saved-match density is known?
- Should zoom become an implementation task after fixture density and real saved-match density are compared?
- Should future fallback testing move from one expanded fixture to separate fixture variants?
- Should drawer-level contribution highlights remain once segment-specific contribution data exists?
