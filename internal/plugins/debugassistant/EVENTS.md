# debugassistant — Events

## Subscribed events

Subscriptions are only established when the plugin is enabled (`enabled()` returns true).

| Event | Source | Purpose |
|---|---|---|
| `match.started` | oofevents bus | Log match start with arena and playlist |
| `state.updated` | oofevents bus | Log a summary of player count and team scores |
| `goal.scored` | oofevents bus | Log scorer, assister, last-touch player, and impact location |
| `stat.feed` | oofevents bus | Log stat-feed type, main target, and secondary target |
| `clock.updated` | oofevents bus | Log game time and overtime flag |
| `match.ended` | oofevents bus | Log match end signal |
| `match.destroyed` | oofevents bus | Log match teardown signal |

## Published events

This plugin does not publish any events to the OOF event bus or to the browser.