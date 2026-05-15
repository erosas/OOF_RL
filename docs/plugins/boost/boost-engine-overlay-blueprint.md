# Boost Engine + Boost Overlay Blueprint

## Recommended Branch

`feature/boost-engine-overlay-blueprint`

This branch should remain docs-only. It defines the Boost Engine and Boost Overlay architecture before any implementation, storage, live integration, replay integration, or UI wiring work begins.

## Purpose

Boost is the next independent enrichment system after Momentum.

Momentum describes event-derived pressure/control and contest state. Boost should describe boost-related match context: available boost signal quality, recent boost usage pressure, low-boost warnings, and team-level boost posture when the available data supports it.

The first Boost work should separate the system into two responsibilities:

- Boost Engine enriches boost-related signals.
- Boost Overlay displays boost-related state.

The overlay must remain display-only. It should not compute or mutate match state, saved data, replay data, Momentum output, or session state.

## Boost Engine Responsibilities

Boost Engine should be responsible for deriving a small, reusable boost state signal from available live or fixture data.

Candidate responsibilities:

- Normalize boost-related inputs into a stable contract.
- Track current or most recent known player boost values when available.
- Mark unknown, stale, or fixture-only boost values clearly.
- Derive low-boost warning states.
- Derive team-level boost availability summaries when enough player data exists.
- Derive recent boost pressure indicators, such as sustained low boost or rapid boost drain, without tactical certainty claims.
- Emit output that can later be reused by Live, History, Journey, Career, overlays, and post-match review surfaces.

Boost Engine should not:

- Mutate Live, History, Session, saved match, replay, or database state.
- Change Momentum Engine math.
- Infer rotation certainty, tactical advantage, player intent, or win odds from boost alone.
- Depend on a specific overlay implementation.

## Boost Overlay Responsibilities

Boost Overlay should be responsible for rendering Boost Engine output in a readable live HUD surface.

Candidate responsibilities:

- Display current boost state using Boost Engine output only.
- Display low-boost and depleted-boost warnings.
- Display team boost panel summaries if enough data exists.
- Display stale, unknown, or fixture-only states without pretending the data is authoritative.
- Respect overlay settings, reduced-motion mode, and performance constraints.
- Stay toggleable and removable through the plugin system.

Boost Overlay should not:

- Compute canonical boost state itself.
- Write to SQLite or app data.
- Mutate live match/session/history data.
- Parse replays.
- Integrate with Overlay Lab unless explicitly approved in a later task.
- Depend on Momentum Overlay internals.

## Current Available Boost Data Assumptions

This blueprint does not assume full authoritative boost data is already available.

Initial implementation should start from fixture or mock Boost Engine output until actual data availability is confirmed.

Potential data sources to verify before implementation:

- Existing live match event payloads.
- Existing player state snapshots, if any.
- Existing overlay fixture/debug payloads.
- Replay-derived player boost values in a future post-match path.
- Manual fixture JSON for visual and contract validation.

Assumptions for planning:

- Some live sources may have no boost value.
- Some values may be delayed, partial, or player-specific only.
- Team-level summaries may be unavailable until all active players have recent values.
- Replay/post-match data may provide richer boost context later than live data.

## Live-Match Limitations

Live Boost should be conservative.

Known limitations to plan around:

- Live data may not expose every player's boost value consistently.
- Missing values must be shown as unknown, not zero.
- Stale values must be identified separately from fresh values.
- Low boost does not prove poor positioning, bad rotation, or tactical failure.
- Team boost summaries are only as reliable as the available player samples.
- Overlay rendering must not block or delay live match event handling.
- Display failures must fail closed without affecting match tracking.

## Replay And Post-Match Future Possibilities

Replay and post-match surfaces may eventually support richer boost analysis after the live overlay contract is stable.

Possible future uses:

- Timeline boost markers for low-boost sequences.
- Post-match boost conservation summaries.
- Team boost recovery windows.
- Boost starvation windows, labeled cautiously.
- Player boost usage trends in Journey or Career.
- Replay review anchors for boost-related moments.
- Combination with Momentum output after both contracts are stable.

Future replay/post-match work should not be treated as part of the first overlay implementation.

## Minimal BoostStateSignal Contract

Boost Engine output should be serializable and independent from overlay rendering code.

```json
{
  "matchId": "preview-match-001",
  "source": "fixture",
  "generatedAt": "2026-05-15T00:00:00Z",
  "freshnessMs": 120,
  "players": [
    {
      "playerId": "blue-1",
      "playerName": "Blue Player",
      "team": "blue",
      "boost": 28,
      "state": "low",
      "isKnown": true,
      "isStale": false,
      "lastUpdatedMs": 120
    }
  ],
  "teams": {
    "blue": {
      "knownPlayers": 2,
      "unknownPlayers": 0,
      "averageBoost": 34,
      "lowestBoost": 12,
      "lowBoostPlayers": 1,
      "state": "strained"
    },
    "orange": {
      "knownPlayers": 1,
      "unknownPlayers": 1,
      "averageBoost": null,
      "lowestBoost": 46,
      "lowBoostPlayers": 0,
      "state": "partial"
    }
  },
  "warnings": [
    {
      "id": "warn-blue-1-low",
      "type": "low_boost",
      "team": "blue",
      "playerId": "blue-1",
      "severity": "medium",
      "summary": "Blue Player is in a low boost state."
    }
  ]
}
```

Initial player fields:

- `playerId`
- `playerName`
- `team`
- `boost`
- `state`
- `isKnown`
- `isStale`
- `lastUpdatedMs`

Allowed player states:

- `unknown`
- `depleted`
- `low`
- `moderate`
- `healthy`

Initial team fields:

- `knownPlayers`
- `unknownPlayers`
- `averageBoost`
- `lowestBoost`
- `lowBoostPlayers`
- `state`

Allowed team states:

- `unknown`
- `partial`
- `strained`
- `stable`
- `healthy`

## Minimal BoostOverlaySignal Contract

Boost Overlay can consume a narrower display contract produced from `BoostStateSignal`.

```json
{
  "source": "fixture",
  "isPreview": true,
  "freshnessLabel": "Fixture preview",
  "players": [
    {
      "id": "blue-1",
      "name": "Blue Player",
      "team": "blue",
      "boostLabel": "28",
      "stateLabel": "Low boost",
      "severity": "medium",
      "isStale": false
    }
  ],
  "teamPanels": [
    {
      "team": "blue",
      "title": "Blue boost state",
      "summary": "One player is in a low boost state.",
      "state": "strained",
      "knownPlayers": 2,
      "unknownPlayers": 0
    }
  ],
  "warnings": [
    {
      "id": "warn-blue-1-low",
      "label": "Low boost",
      "team": "blue",
      "severity": "medium",
      "summary": "Blue Player is in a low boost state."
    }
  ]
}
```

Overlay signal rules:

- It should be display-ready.
- It should preserve unknown and stale states.
- It should not require overlay code to infer team state.
- It should include preview labels when using fixture/mock data.

## UI And Overlay Anatomy

Initial Boost Overlay anatomy:

- Compact player boost cards.
- Team boost panel.
- Warning strip for low or depleted boost states.
- Freshness/source label.
- Unknown/stale data affordance.

Optional later anatomy:

- Minimal boost trend sparkline.
- Recently recovered boost indicator.
- Team low-boost count chip.
- Disabled replay/post-match placeholder only in preview contexts.

Overlay layout should stay readable at game-overlay scale. It should favor clear state, labels, and contrast over dense decoration.

## Boost Warning Concepts

Warning concepts should be descriptive and conservative.

Initial warning types:

- `low_boost`
- `depleted_boost`
- `team_low_boost`
- `unknown_boost`
- `stale_boost`

Warning severity:

- `info`
- `low`
- `medium`
- `high`

Safe warning examples:

- "Blue Player is in a low boost state."
- "Orange team boost data is partial."
- "Boost signal is stale."
- "Two blue players are in low boost states."

Avoid warning examples:

- "Blue has lost rotation."
- "Orange has tactical advantage."
- "Blue will concede."
- "This player made the wrong decision."

## Team Boost Panel Concepts

Team panels should summarize available boost signal quality without overstating certainty.

Initial panel content:

- Team name.
- Known player count.
- Unknown player count.
- Average boost when all or enough values are fresh.
- Lowest known boost.
- Low-boost player count.
- State label.

State label examples:

- "Unknown boost signal"
- "Partial boost signal"
- "Strained boost state"
- "Stable boost state"
- "Healthy boost state"

Team panels should not claim team control, true possession, tactical superiority, or rotation quality.

## Safe Terminology

Use these terms:

- Boost state.
- Boost signal.
- Low boost state.
- Depleted boost state.
- Boost availability.
- Boost pressure.
- Boost recovery.
- Boost warning.
- Team boost state.
- Partial boost signal.
- Stale boost signal.
- Fixture/mock boost output.

Avoid these terms:

- Rotation certainty.
- Tactical certainty.
- Tactical advantage.
- True control.
- Possession.
- Ball ownership.
- Win odds.
- Player intent.
- Guaranteed outcome.

Boost copy can describe known boost values and cautious derived states. It should not infer hidden decision-making or tactical correctness.

## Non-Goals

- No code implementation in this branch.
- No DB/schema changes.
- No saved match/session/history/replay mutation.
- No Live, History, Session, Journey, or Career integration.
- No Momentum Engine integration.
- No Momentum Overlay refactor.
- No Overlay Lab integration.
- No replay parsing.
- No event-bus integration.
- No new frontend framework, TypeScript, JSX, Vite, Webpack, or build pipeline.
- No coaching-grade claims.
- No rotation or tactical certainty claims.

## Initial Implementation Phases

### Phase 0 - Data Availability Audit

- Inspect current live event and player state payloads.
- Identify whether boost values exist, are fresh, or are missing.
- Document unknowns before implementing engine behavior.

### Phase 1 - Fixture Contract Preview

- Create fixture/mock BoostStateSignal output.
- Create a hidden/internal Boost Overlay preview route or plugin view.
- Render player cards, team panels, warnings, and unknown/stale states.
- Keep the route clearly labeled as preview/mock.

### Phase 2 - Boost Engine Skeleton

- Add an isolated Boost Engine package only after the contract is reviewed.
- Normalize fixture/mock input first.
- Add unit tests for state thresholds, unknown values, stale values, and team summaries.
- Do not connect live data yet.

### Phase 3 - Display-Only Overlay Integration

- Add a toggleable Boost Overlay plugin surface.
- Consume Boost Engine output.
- Preserve reduced-motion and performance constraints.
- Keep overlay failures non-blocking.

### Phase 4 - Controlled Live Signal Integration

- Connect live boost data only after available fields are verified.
- Keep missing and stale values visible.
- Avoid DB/schema changes unless a separate approved storage plan exists.

### Phase 5 - Post-Match And Timeline Planning

- Define how boost sequences could appear in Timeline, History, Journey, or Career.
- Keep integration with Momentum deferred until both systems have stable contracts.

## Acceptance Criteria

Blueprint acceptance:

- Defines Boost Engine and Boost Overlay responsibilities separately.
- Defines current data assumptions and live limitations.
- Defines minimal BoostStateSignal and BoostOverlaySignal contracts.
- Defines UI anatomy, warning concepts, and team panel concepts.
- Defines safe terminology and non-goals.
- Keeps the work docs-only.
- Preserves plugin isolation and display-only overlay boundaries.

Future first implementation acceptance:

- Fixture/mock output only.
- Hidden/internal preview route or plugin view.
- Vanilla HTML/CSS/JS only.
- No new dependencies.
- No DB/schema changes.
- No saved app data mutation.
- No Live, History, Session, saved match, replay, Overlay Lab, Momentum, or event-bus integration.
- Boost Overlay consumes display-ready boost output.
- Unknown and stale states render safely.
- Low/depleted boost warnings render without tactical certainty claims.
- Existing app behavior remains unaffected.

## Risks And Guardrails

- Boost data may be unavailable or partial in live sources; unknown must not be treated as zero.
- Stale boost values can be misleading; freshness must be explicit.
- Boost alone must not imply rotation certainty, tactical certainty, player intent, or win odds.
- Overlay code must not become a second Boost Engine.
- Team panels must not hide incomplete player data.
- Fixture output must be clearly labeled as preview/mock.
- Do not store derived boost summaries until a separate storage contract is approved.
- Do not connect Boost to Momentum until both systems have stable independent contracts.
- Do not let overlay rendering block live match tracking.

## Open Questions

- Which live payloads currently expose boost values, if any?
- What freshness window should classify a boost value as stale?
- Should average team boost require all active players, a majority, or any known player values?
- What thresholds should define depleted, low, moderate, and healthy boost states?
- Should team state be derived from average boost, lowest known boost, low-player count, or a weighted combination?
- Should the first preview route be a dedicated Boost plugin page or an Overlay HUD preview variant?
- Should Boost Overlay ship as a separate toggle from Momentum Overlay?
- What should be the first product surface after preview: live overlay only, Timeline annotations, or History match detail?

## Recommended Next Implementation Task

Create a fixture-backed Boost Overlay preview page.

Recommended branch:

`feature/boost-overlay-fixture-preview`

Implementation constraints:

- Fixture/mock output only.
- Hidden/internal route only.
- Vanilla HTML/CSS/JS only.
- No new dependencies.
- No DB/schema changes.
- No saved app data mutation.
- No Live, History, Session, saved match, replay, Overlay Lab, Momentum, or event-bus integration.

First implementation should prove:

- Boost player cards.
- Team boost panels.
- Low/depleted boost warnings.
- Unknown and stale states.
- Preview/mock labeling.
- Display-only overlay boundaries.
