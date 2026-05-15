# Momentum Timeline B-lite Blueprint

## Purpose

Momentum Timeline B-lite is the first post-match story layer for the Momentum system.

The Momentum Overlay answers what is happening now. It is a live HUD surface for current event-derived pressure, control, contest state, and momentum share.

The Momentum Timeline answers what happened across the match. It should turn Momentum Engine output and notable match events into a readable timeline that helps players review pressure windows, momentum swings, and event clusters after play.

## B-lite Scope

B-lite is intentionally smaller than Replay Studio or a full match-analysis product. It should prove the timeline shape, wording, and interaction model before deeper replay, badge, and history integrations.

Included in B-lite:

- Timeline bar.
- Pressure, control, contest, and neutral bands.
- Event markers for goals, assists, shots, saves, epic saves, and demos.
- Clickable timeline sections.
- Expandable selected moment drawer.
- Timestamps for replay review.
- Team pressure, control, and contest totals.
- Player pressure contribution summaries.

Excluded from B-lite:

- Full replay viewer.
- Replay Studio.
- Clip generation.
- Badge Engine implementation.
- Journey or Career integration.
- DB/schema changes.
- Saved match/session/history/replay mutation.
- Deep coaching inference.

## Safe Terminology

Use these terms:

- Event-derived pressure/control.
- Pressure contribution.
- Momentum influence.
- Contest involvement.
- Pressure sequence.
- Momentum swing.

Avoid these terms:

- True possession.
- Ball control.
- Rotation quality.
- Tactical advantage.
- Win odds.

Timeline copy should be careful and review-oriented. It can describe events and signal changes, but it should not infer player intent, tactical correctness, or hidden game state.

## Initial Placement

Start with an experimental Momentum Timeline preview route/page.

Do not place B-lite inside Overlay Lab. Overlay Lab is for live overlay tuning, visual customization, presets, and HUD preview behavior.

Do not integrate B-lite into History match detail yet. History integration should happen after the timeline data contract and interaction model are stable.

Recommended first placement:

- A temporary Momentum Timeline preview page or plugin view.
- Clearly labeled as experimental or preview.
- Hidden from default app navigation if needed, or exposed only as a developer/internal route until validated.

## Initial Data Source

Use fixture data or in-memory preview data first.

B-lite should not write to SQLite, saved matches, session history, replay data, or app data. It should not require a schema migration.

Recommended first data options:

- Static fixture JSON embedded in the preview page for layout validation.
- In-memory preview data generated from existing Momentum output shape.
- Optional local frontend-only sample buffer for preview interaction, reset on reload.

Do not treat fixture output as authoritative match data. The first implementation should prove rendering, interaction, wording, and data shape only.

## B-lite V1 Decisions

These decisions apply to the first fixture-backed preview page only. They are intended to keep the initial implementation isolated, reviewable, and separate from the Momentum Overlay HUD v1 work.

- Fixture data should live in a separate frontend-only JSON fixture when the host route can load it without app-data access. If route loading makes that awkward, embedded in-memory fixture data is acceptable for the first patch.
- The first route should be hidden or internal. It should not appear as a default production navigation item until the interaction model is validated.
- Elapsed time should be the default timestamp format for the first preview. Match-clock formatting can be added later after the product context is clearer.
- Dense event marker grouping should be a follow-up. V1 should render readable individual markers and can use simple stacking or spacing only when needed for legibility.
- Player contributions should be match-wide in V1. Segment-specific contribution highlighting can be added later if the data contract supports it cleanly.
- The first implementation should use a separate Momentum Timeline preview page or plugin boundary. It should not reuse or modify Overlay HUD internals unless a later task explicitly approves that integration.

## Minimal Data Contract

The B-lite contract should be serializable as JSON and independent from the live overlay rendering code.

```json
{
  "match": {
    "id": "preview-match-001",
    "source": "fixture",
    "playlist": "2v2",
    "durationSeconds": 300,
    "blueName": "Blue",
    "orangeName": "Orange"
  },
  "totals": {
    "bluePressureSeconds": 92,
    "blueControlSeconds": 64,
    "orangePressureSeconds": 84,
    "orangeControlSeconds": 58,
    "contestedSeconds": 51,
    "neutralSeconds": 13
  },
  "segments": [
    {
      "id": "seg-001",
      "startSecond": 0,
      "endSecond": 18,
      "state": "neutral",
      "team": "none",
      "confidence": 0.22,
      "pressureShareBlue": 0.5,
      "pressureShareOrange": 0.5,
      "summary": "Opening neutral pressure."
    },
    {
      "id": "seg-002",
      "startSecond": 18,
      "endSecond": 42,
      "state": "pressure",
      "team": "blue",
      "confidence": 0.68,
      "pressureShareBlue": 0.64,
      "pressureShareOrange": 0.36,
      "summary": "Blue pressure sequence before a shot."
    }
  ],
  "events": [
    {
      "id": "evt-001",
      "second": 37,
      "type": "shot",
      "team": "blue",
      "playerId": "player-blue-1",
      "playerName": "Blue Player",
      "segmentId": "seg-002",
      "label": "Shot",
      "summary": "Shot during blue pressure sequence."
    }
  ],
  "playerContributions": [
    {
      "playerId": "player-blue-1",
      "playerName": "Blue Player",
      "team": "blue",
      "pressureContribution": 0.34,
      "momentumInfluence": 0.28,
      "contestInvolvement": 0.12,
      "events": {
        "shots": 2,
        "saves": 1,
        "assists": 0,
        "goals": 1,
        "demos": 0
      }
    }
  ]
}
```

### Match Metadata

Match metadata should identify the preview source, match duration, teams, and future match/replay references.

Required B-lite fields:

- `id`
- `source`
- `durationSeconds`
- `blueName`
- `orangeName`

Optional future fields:

- `matchGuid`
- `sessionId`
- `replayPath`
- `playlist`
- `startedAt`

### Totals

Totals summarize the match-level story without requiring History integration.

Initial totals:

- Blue pressure seconds.
- Blue control seconds.
- Orange pressure seconds.
- Orange control seconds.
- Contested seconds.
- Neutral seconds.

### Segments

Segments are continuous timeline windows.

Initial segment fields:

- `id`
- `startSecond`
- `endSecond`
- `state`
- `team`
- `confidence`
- `pressureShareBlue`
- `pressureShareOrange`
- `summary`

Allowed states:

- `neutral`
- `pressure`
- `control`
- `contested`

### Events

Events are markers over the segment bands.

Initial event fields:

- `id`
- `second`
- `type`
- `team`
- `playerId`
- `playerName`
- `segmentId`
- `label`
- `summary`

Initial event types:

- `goal`
- `assist`
- `shot`
- `save`
- `epic_save`
- `demo`

### Player Contributions

Player contribution summaries are descriptive, not authoritative possession or tactical grading.

Initial contribution fields:

- `playerId`
- `playerName`
- `team`
- `pressureContribution`
- `momentumInfluence`
- `contestInvolvement`
- `events`

## UI Anatomy

### Summary Strip

The summary strip should sit above the timeline and show the match-level totals.

Suggested content:

- Blue pressure/control time.
- Orange pressure/control time.
- Contested time.
- Neutral time.
- Top pressure contributor.
- Biggest momentum swing placeholder.

### Timeline Bar

The timeline bar is the primary visual.

It should show:

- Blue pressure/control bands.
- Orange pressure/control bands.
- Contested bands.
- Neutral bands.
- Time scale from match start to match end.

Visual behavior:

- Blue and orange should remain distinct.
- Contested should use a separate mixed/neutral treatment, not imply either team owns the moment.
- Low-confidence or neutral windows should be visually calmer.

### Event Marker Row

Event markers sit above or below the timeline bar.

Marker types:

- Goal.
- Assist.
- Shot.
- Save.
- Epic save.
- Demo.

Markers should remain readable at match scale. If events cluster, B-lite can group markers or show a compact stack.

### Selected Moment Drawer

Clicking a segment or marker opens a drawer.

Drawer content:

- Timestamp range.
- Segment state.
- Team, if applicable.
- Summary text.
- Events inside the selected window.
- Player contributions related to the window.
- Future replay jump placeholder.

The drawer should not claim tactical causes. It should summarize event-derived pressure/control and notable events.

### Player Contribution Summary

The player contribution summary should show team-grouped players and their contribution values.

Initial display:

- Player name.
- Team.
- Pressure contribution.
- Momentum influence.
- Contest involvement.
- Relevant event counts.

## Interaction Behavior

### Click Segment

Clicking a segment selects the time window and opens the selected moment drawer.

Expected behavior:

- Highlight selected segment.
- Populate drawer with segment summary.
- List events within the segment.
- Show player contribution highlights for the segment if available.

### Click Event Marker

Clicking an event marker selects the event and opens the drawer at the event timestamp.

Expected behavior:

- Highlight marker.
- Highlight containing segment.
- Show event summary.
- Show nearby pressure/control context.

### Open Drawer

The drawer should be expandable/collapsible.

Required states:

- Empty state before selection.
- Segment-selected state.
- Event-selected state.

### Show Events and Timestamps

All visible events in the drawer should include timestamps.

Use match-clock or elapsed-time formatting consistently. The blueprint does not decide final formatting; implementation should choose one and document it.

### Future Replay Jump Placeholder

Add a disabled or non-functional placeholder for future replay jumps.

Copy should make the state clear:

- `Replay jump planned`
- `Replay Studio integration later`

Do not wire Replay Studio behavior in B-lite.

## Acceptance Criteria

B-lite blueprint acceptance:

- Defines purpose, scope, terminology, placement, data source, data contract, UI anatomy, interactions, and future integrations.
- Uses safe pressure/control wording.
- Excludes Replay Studio, clip generation, Badge Engine implementation, Journey/Career integration, DB/schema changes, and deep coaching inference.

Initial B-lite implementation acceptance:

- No DB/schema changes.
- No saved app data mutation.
- No Live/History/Session/saved match/replay mutation.
- Fixture or in-memory data only.
- Vanilla JS/CSS compatible.
- No React, TypeScript, JSX, Vite, Webpack, or new frontend build pipeline.
- Timeline bar renders pressure/control/contest/neutral segments from the minimal contract.
- Event markers render from the minimal contract.
- Clicking a segment opens the selected moment drawer.
- Clicking an event marker opens the drawer with event context.
- Player contribution summary renders from fixture/in-memory data.
- Copy uses safe terminology.
- UI leaves a clear future path to History match detail and Replay Studio.

## Future Integration Notes

### History Match Detail

After B-lite proves the interaction model, History match detail is the likely first product integration point.

History integration should show the timeline for a saved match only after the data source and storage contract are approved.

### Session Summaries

Session pages can later summarize pressure/control trends across several matches.

Possible future summaries:

- Most contested match.
- Highest team pressure match.
- Best pressure recovery sequence.
- Match with biggest momentum swing.

### Journey Trends

Journey can use Timeline-derived summaries to show long-term progression.

Possible future trend concepts:

- Pressure contribution over time.
- Contest involvement over time.
- Momentum swing frequency.
- Pressure recovery frequency.

### Highlights Page

Highlights can use Timeline windows and events to surface notable moments.

Possible future highlight candidates:

- Goal after pressure sequence.
- Epic save during pressure window.
- Demo before a goal.
- Contested sequence.
- Rapid shot cluster.

### Badge Engine

Badge Engine can later label Timeline windows.

Potential badge inputs:

- Pressure Swing.
- Contested Battle.
- Defensive Stand.
- Pressure Recovery.
- Pressure Collapse.

B-lite should not implement Badge Engine logic.

### Replay Studio

Replay Studio can later consume Timeline timestamps for review and clip workflows.

B-lite should only reserve placeholders for replay jumps.

### Coaching Pages

Coaching pages can later use Timeline windows to ask review-oriented questions.

Safe examples:

- What started this momentum swing?
- Which events occurred during this pressure sequence?
- How did pressure recover in this window?

Avoid exact tactical claims or intent inference.

## Open Questions

- After the hidden preview is validated, should the first visible product placement be History match detail, a standalone Momentum page, or another plugin route?
- When real match data is approved, should the timeline store derived segment summaries, rebuild them from saved match events, or support both?
- What threshold should define a momentum swing once the preview moves beyond fixture labels?
- Which timestamp mode should be user-facing by default after preview: elapsed time, match clock, or a toggle?
- What is the minimum acceptable dense-marker grouping behavior for real saved matches?

## Recommended Next Implementation Task

Create a fixture-backed Momentum Timeline B-lite preview page.

Implementation constraints:

- Fixture or in-memory data only.
- No DB/schema changes.
- No saved match/session/history/replay mutation.
- Vanilla HTML/CSS/JS only.
- No new dependencies.

First implementation should prove:

- Timeline bar rendering.
- Event marker rendering.
- Segment and marker selection.
- Selected moment drawer.
- Player contribution summary.
- Safe copy and review-oriented wording.
