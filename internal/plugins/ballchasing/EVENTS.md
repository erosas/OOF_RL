l# ballchasing — Events

## Subscribed events

| Event | Source | Purpose |
|---|---|---|
| `match.ended` | oofevents bus | Broadcast save-replay reminder to the browser while the user is still on the post-match screen |
| `match.destroyed` | oofevents bus | Trigger background auto-upload of the most recent replay file to Ballchasing |

## Published events (WebSocket hub broadcasts)

These are not OOF bus events — they are JSON messages broadcast directly to all connected browser clients via the WebSocket hub.

| Event key | Constant | Payload | When |
|---|---|---|---|
| `bc:save-replay-reminder` | `WSEventSaveReplayReminder` | `{"Event": "bc:save-replay-reminder"}` | On `match.ended` — prompts user to save the replay before leaving the post-match lobby |
| `bc:uploaded` | `WSEventUploaded` | `{"Event": "bc:uploaded", "Data": {"replays": [{"name": "...", "bc_id": "..."}]}}` | After successful auto-upload — notifies the UI that one or more replays were uploaded |