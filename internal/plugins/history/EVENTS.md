# history — Events

## Subscribed events

| Event | Source | Purpose |
|---|---|---|
| `match.started` | oofevents bus | Begin a new match record; capture arena, playlist, match GUID |
| `state.updated` | oofevents bus | Track current player snapshots (primary ID, shortcut, car touches) and team snapshots |
| `goal.scored` | oofevents bus | Record goal with scorer/assister/last-touch shortcuts and impact location |
| `ball.hit` | oofevents bus | Optionally persist ball-hit rows when `storage.ball_hit_events` is enabled |
| `clock.updated` | oofevents bus | Track current game time (seconds elapsed) for goal timestamps |
| `stat.feed` | oofevents bus | Persist stat-feed rows (saves, shots, demos, etc.) with shortcut-resolved player IDs |
| `match.ended` | oofevents bus | Mark match as ended in state; triggers final stat accumulation |
| `match.destroyed` | oofevents bus | Flush accumulated match data to the database |

## Published events

This plugin does not publish any events to the OOF event bus or to the browser.