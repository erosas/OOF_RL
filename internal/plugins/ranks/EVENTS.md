# ranks — Events

## Subscribed events

| Event | Source | Purpose |
|---|---|---|
| `state.updated` | oofevents bus | Detect new players and trigger MMR lookups via the tracker provider |
| `match.destroyed` | oofevents bus | Clear tracked player set so lookups fire again next match |

## Published events

This plugin does not publish any events to the OOF event bus or to the browser.