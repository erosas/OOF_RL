# OOF RL Future Ideas

This file captures product and API ideas that are not implemented yet. Treat
these notes as planning references, not current app behavior.

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
