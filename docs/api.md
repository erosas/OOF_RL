# HTTP API

OOF RL exposes a local HTTP server (default `http://localhost:8080`). All JSON responses use `Content-Type: application/json`.

---

## WebSocket

### `GET /ws`

Upgrades to a WebSocket connection. The server pushes every RL event and internal status message as it arrives.

**Message format:**

```json
{ "Event": "EventName", "Data": { ... } }
```

**Events pushed by OOF RL:**

| Event | Description |
|-------|-------------|
| `_Status` | `{ "connected": true/false }` — RL TCP connection state |
| `MatchCreated` | New match GUID assigned |
| `MatchInitialized` | Match fully ready |
| `UpdateState` | Full game tick — players, ball, scores, boost |
| `GoalScored` | Goal scored |
| `BallHit` | Ball touched by a player |
| `MatchEnded` | Winner decided |
| `MatchDestroyed` | Post-game screen dismissed |
| `bc:uploaded` | `{ "replays": [{ "name", "bc_id", "bc_url" }] }` — auto-upload completed |

---

## App Config

### `GET /api/config`
Returns the current `config.toml` as JSON.

### `POST /api/config`
Accepts a partial or full config JSON body, saves it, and triggers a reconnect to the RL socket.

### `GET /api/config/ini`
Returns the current `DefaultStatsAPI.ini` values: `{ "PacketSendRate": 60, "Port": 49123 }`.  
Returns `{ "error": true, "note": "..." }` if the file is not found.

### `POST /api/config/ini`
Writes `DefaultStatsAPI.ini` with the given `PacketSendRate` and `Port` values and triggers a reconnect.

---

## Navigation & Plugins

### `GET /api/nav`
Returns the ordered list of enabled plugin tabs:
```json
[{ "id": "live", "label": "Live", "order": 10 }, ...]
```

### `GET /api/plugins/{id}/view`
Returns the raw `view.html` fragment for plugin `{id}`. Used by the frontend to inject plugin views on demand.

### `GET /api/settings/schema`
Returns all plugin settings definitions, used to render the Settings page:
```json
[{
  "plugin_id": "history",
  "nav_tab_id": "history",
  "title": "History",
  "enabled": true,
  "requires": [],
  "settings": [{ "key": "storage.ball_hit_events", "label": "Ball hit events", "type": "checkbox", ... }]
}]
```

### `POST /api/settings`
Accepts a flat `{ "key": "value" }` map and dispatches it to every plugin's `ApplySettings`. Saves `config.toml` and triggers a reconnect.

---

## History

### `GET /api/players`
All players ever seen: `[{ "PrimaryID": "steam|...", "Name": "...", "LastSeen": "..." }]`

### `GET /api/matches?player={primaryID}`
Match list, optionally filtered to matches where `player` appeared.
```json
[{
  "id": 42,
  "match_guid": "...",
  "arena": "DFH Stadium",
  "started_at": "...",
  "ended_at": "...",
  "winner_team_num": 0,
  "overtime": false,
  "playlist_type": 13,
  "team0_goals": 3,
  "team1_goals": 1,
  "player_count": 6
}]
```

### `GET /api/matches/{id}`
Match detail: per-player stats and goal log.
```json
{
  "players": [{ "primary_id": "...", "name": "...", "team_num": 0, "score": 450, "goals": 2, ... }],
  "goals":   [{ "scorer_id": "...", "scorer_name": "...", "goal_speed": 87.3, "goal_time": 62.1, ... }]
}
```

---

## Session

### `GET /api/session/stats?since={RFC3339}&player={primaryID}`
Stats for the given player since the given timestamp.
```json
{
  "summary": { "games": 5, "wins": 3, "losses": 2, "goals": 8, "assists": 4, "saves": 6, "shots": 14, "demos": 2 },
  "matches": [{ "match_id": 42, "arena": "DFH Stadium", "started_at": "...", "winner_team_num": 0, "player_team": 0, "goals": 2, ... }]
}
```

---

## Ranks

### `GET /api/ranks/players`
Players in the current match with their cached team assignment:
```json
[{ "primary_id": "steam|...", "name": "...", "team_num": 0 }]
```
Returns an empty array when no match is active.

---

## Tracker

### `GET /api/tracker/profile?id={primaryID}&name={displayName}`
Fetches tracker.gg ranks for the player. Results are cached in the database for `tracker_cache_ttl_minutes` minutes.

`id` format: `platform|accountID` (e.g. `steam|76561198000000001`, `epicgames|GamerTag`).  
`name` is required for non-Steam platforms to resolve the tracker.gg profile URL.

```json
{
  "cached": false,
  "fetched_at": "2025-01-01T12:00:00Z",
  "source": "tracker.gg",
  "ranks": [{
    "PlaylistID": 11,
    "PlaylistName": "Ranked Doubles 2v2",
    "MMR": 1042.5,
    "Tier": 12,
    "TierName": "Diamond I",
    "Division": 2,
    "IconURL": "https://..."
  }]
}
```

---

## Ballchasing

### `GET /api/ballchasing/ping`
Validates the API key by making a test request to ballchasing.com. Returns uploader info on success.

### `GET /api/ballchasing/replays`
Proxies `GET /api/replays` from ballchasing.com. Pass any supported query parameters.

### `GET /api/ballchasing/groups`
Proxies `GET /api/groups` from ballchasing.com.

### `POST /api/ballchasing/upload`
Uploads a local replay file to ballchasing.com.
```json
{ "replay_name": "2024-01-01.replay", "visibility": "public-team" }
```

### `GET /api/ballchasing/uploads`
Local upload history: `{ "2024-01-01.replay": { "ballchasing_id": "...", "bc_url": "...", "uploaded_at": "..." } }`

---

## Utilities

### `GET /api/data-dir`
Returns `{ "path": "C:\\Users\\...\\OOF_RL" }` — the data directory path.

### `GET /api/db/open-folder`
Opens the data directory in Windows Explorer.
