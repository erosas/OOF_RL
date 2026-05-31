# OOF RL Future Ideas

This file captures product and API ideas that are not implemented yet. Treat
these notes as planning references, not current app behavior.

## Ranks And MMR Follow-Up Ideas

These ideas are planned follow-up candidates after the UI reskin foundation.
They are not current behavior and should not be implemented inside the reskin
PR unless explicitly re-scoped.

### Rank Card Formatting And Focused Previews

Candidate behavior:

- Reformat Ranks into larger playlist/rank preview cards that match the visual
  language established during the reskin pass.
- Add a focused rank preview mode so the page can show the most relevant rank
  card prominently while keeping other playlists available in a shelf/grid.
- Add visible-mode controls later so users can decide which playlists appear on
  the Ranks page.
- Use confirmed match type, once available, to focus Live and Ranks on the
  relevant playlist instead of showing every known provider rank by default.

Current behavior to preserve until that feature lands:

- Ranks and Live may show all known provider ranks for a player.
- Provider rank data is fetched and rendered through existing MMR provider and
  tracker cache behavior.

### Rank Icons And Assets

Candidate behavior:

- Prefer provider-supplied rank icon URLs when they are stable and safe to load.
- If provider icons are unreliable or slow, consider storing small local rank
  icon assets and mapping them by tier/division.
- Keep icon use display-only. Icons must not become the source of rank truth.

Research needed:

- Confirm which providers expose stable icon URLs.
- Decide whether local icon assets are worth the maintenance cost.
- Confirm licensing/usage expectations before bundling provider-derived icons.

### Per-Mode MMR Thresholds And Prediction

Candidate behavior:

- Pull or maintain per-playlist MMR threshold rules so Ranks and Session can
  show promotion/demotion guidance, such as "roughly X MMR to next division."
- Scope thresholds by playlist/mode because rank ranges differ by mode.
- Label predictions clearly as estimates unless the source is authoritative.
- Keep calculations read-only and avoid mutating saved match/session data.

Research needed:

- Identify reliable MMR threshold sources per season and playlist.
- Decide how threshold data is updated when seasons change.
- Define fallback behavior when threshold data is missing, stale, or conflicts
  across sources.

### Optional MMR Recording

Candidate behavior:

- Add a user-controlled setting to record observed MMR snapshots with matches.
- Use recorded MMR to enrich History, Session summaries, and progression views.
- Make the toggle explicit because MMR history increases stored user data.

Safety notes:

- MMR recording is a data-model feature and should be a separate PR.
- Any implementation must document what is read, what is written, retention
  behavior, and how existing match/history rows are handled.
- If added to History, avoid backfilling or inferring old MMR unless explicitly
  requested and clearly labeled.

## Match Badge Logic Ideas

These badges are proposed future match-history and analysis signals. Some
depend on Rocket League Stats API fields that may not currently be available.

| Badge | Proposed Trigger Logic |
| --- | --- |
| Forfeit | Trigger if `match_end_reason == "forfeit"`. |
| Full-Time | Trigger if `match_duration >= regulation_time` and `match_end_reason == "time_expired"`. |
| Demo | Trigger if `total_demos >= demo_threshold`, such as 5 or more demos. |
| Overtime | Trigger if `match_end_time > regulation_time` or `overtime_status == true`. |
| Consecutive Goal | Track goal streaks per team. Trigger if one team scores X goals in a row without the other team scoring. |
| Hat Trick | Trigger per player if `player_goals >= 3`. |
| Perfect Save | Trigger if a player records saves while their team concedes 0 goals. |
| Comeback | Trigger if a team was trailing by 2 or more goals and later wins or ties. |
| Shutout | Trigger if a team allows 0 goals against by match end. |

## API Limitation Notes

Some badge logic may not be reliable or possible through the currently
available Rocket League Stats API data. Avoid presenting heuristic badges as
facts unless the required source data is available.

Missing or desired API fields:

- `match_duration`
- `time_expired`
- `match_end_reason`
- `overtime_status`
- explicit forfeit team
- richer replay/player state events

Examples of richer replay/player state events that would improve analytics:

- demos
- powerslide state
- boost state
- wall state
- supersonic state

## API Feature Request Draft

Subject: Feature Suggestion for Expanded Match Data in Rocket League Stats API

Dear Developer Team,

I am writing to suggest a few additional data points that would greatly enhance
match analytics. Specifically, having access to match duration, time expired,
and an explicit match end reason such as time expired, forfeit, or other
match-ending states would allow developers to categorize match outcomes more
precisely.

Additional useful fields would include overtime status, which team forfeited,
and richer replay/player state events such as demos, powerslide, boost state,
wall state, and supersonic state.

These additions would help developers build better analytics tools and give the
Rocket League community deeper insight into match flow, player behavior, and
match outcomes.

Best regards,

[Name]
