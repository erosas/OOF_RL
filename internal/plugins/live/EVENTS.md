# live — Events

## Subscribed events

| Event | Source | Purpose |
|---|---|---|
| `state.updated` | oofevents bus | Update in-memory player/team state; broadcast to browser via WebSocket hub |
| `match.destroyed` | oofevents bus | Clear in-memory state; broadcast reset to browser |

## Published events

This plugin does not publish events to the OOF event bus.

It broadcasts raw WebSocket messages directly to connected browser clients via the hub (not through the bus).