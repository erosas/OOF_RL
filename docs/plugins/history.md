# History Plugin

The History plugin records Rocket League match data into the local SQLite database and exposes that data through the History tab and local HTTP API.

It is the durable match record for OOF RL. Live tracking receives real-time events, but History is where completed matches, player stats, goal events, and statfeed events are persisted for later review.

---

## Purpose

The plugin exists to:

- Create one persistent match row per Rocket League match.
- Store players seen in completed matches.
- Store per-player match stats such as score, goals, assists, saves, shots, touches, and demos.
- Store goal events, including scorer, assister, goal speed, goal time, and impact location.
- Store statfeed events such as shots, saves, demos, and other feed entries.
- Serve match list and match detail data to the History UI.
- Provide match records that other app features can read, such as session summaries.

The plugin does not own the Rocket League socket connection. It consumes normalized events dispatched by the core app.

---

## Runtime Flow

The History plugin implements the standard `plugin.Plugin` interface in `internal/plugins/history/plugin.go`.

At startup:

1. `New()` runs the embedded history schema migration.
2. Missing compatibility columns are added with `AddColumnIfNotExists`.
3. The plugin registers API routes for players and matches.
4. The plugin waits for Rocket League events through `HandleEvent`.

During a match:

1. `MatchCreated` or `MatchInitialized` switches to the current match GUID.
2. `UpdateState` switches match context if a new GUID appears.
3. `UpdateState` creates or updates the active `hist_matches` row.
4. `UpdateState` also keeps the latest in-memory player, team, playlist, overtime, and clock state.
5. `GoalScored` stores goal events in `hist_goal_events` when the event belongs to the active match.
6. `BallHit` optionally stores ball hit records when enabled in config and the event belongs to the active match.
7. `StatfeedEvent` stores feed events in `hist_statfeed_events` when the event belongs to the active match.
8. `MatchEnded` flushes the final match state as a completed match when the event belongs to the active match.
9. `MatchDestroyed` flushes the match as incomplete if no `MatchEnded` event was received.

After the match is flushed, the plugin resets its in-memory match state and waits for the next match.

---

## Event Handling

| Event | History behavior |
|-------|------------------|
| `MatchCreated` | Switches to the incoming match GUID. If another match was active, that previous match is flushed as incomplete before the new GUID is stored. |
| `MatchInitialized` | Same as `MatchCreated`; switches to the incoming match GUID. |
| `UpdateState` | Switches match context when needed, creates the match row, updates latest players, teams, playlist, overtime, and game clock state. |
| `GoalScored` | Inserts a goal event with scorer, assister, last-touch player, speed, goal time, impact position, and timestamp when the event GUID matches the active match. |
| `BallHit` | Inserts ball hit data only when `cfg.Storage.BallHitEvents` is enabled and the event GUID matches the active match. |
| `StatfeedEvent` | Inserts a feed event for the actor, optional target, event type, team, and timestamp when the event GUID matches the active match. |
| `MatchEnded` | Marks the active match complete, records winner, overtime, forfeit state, final team scores, and player stats when the event GUID matches the active match. |
| `MatchDestroyed` | Marks the active match incomplete when the match is torn down without a prior `MatchEnded`. |

`GoalScored` events are filtered because Rocket League can emit a second replay-end goal packet with an empty scorer name. The plugin ignores that empty-scorer duplicate.

Events that include a stale match GUID are ignored for the active match. This prevents delayed `GoalScored`, `BallHit`, `StatfeedEvent`, or `MatchEnded` packets from mutating the wrong match after a match transition.

---

## In-Memory State

The plugin keeps a small amount of active-match state before writing final stats:

| Field | Purpose |
|-------|---------|
| `matchID` | Local SQLite match ID for the active match. |
| `matchGuid` | Rocket League match GUID for the active match. |
| `overtime` | Latest overtime state. |
| `playlistType` | Playlist ID from the match payload, when available. |
| `lastPlayers` | Latest known player map used when flushing final player stats. |
| `lastTeams` | Latest known team score state. |
| `lastTimeSeconds` | Latest clock value, used to infer forfeits. |

This state is intentionally short-lived. It should be reset at match boundaries so players or scores from one match do not leak into the next match.

---

## Match Boundary And Roster Handling

The plugin uses `switchMatch(matchGuid)` to protect match boundaries. When a new non-empty match GUID appears and it differs from the active GUID:

1. The previous active match is flushed as incomplete if it had already created a local match row.
2. All in-memory match state is reset.
3. The new match GUID becomes the active match.

Event handlers use `isActiveMatch(matchGuid)` before writing event data. Empty GUIDs are allowed because some payloads may not include a GUID, but non-empty stale GUIDs are ignored.

Roster handling uses a latest-snapshot strategy:

- Normal live `UpdateState` payloads replace `lastPlayers` with the current roster snapshot.
- This removes players that appeared in older snapshots but are no longer in the active match.
- Replay/post-match payloads marked with `BReplay` are treated more carefully.
- A replay snapshot can replace the roster only when it has at least as many players as the current roster.
- Smaller replay snapshots are ignored so a full 2v2 or 3v3 roster is not accidentally shrunk to a partial team after the match ends.

This behavior is covered by boundary tests in `internal/plugins/history/plugin_boundary_test.go`.

---

## Database Tables

All History plugin tables use the `hist_` prefix.

| Table | Purpose |
|-------|---------|
| `hist_players` | One row per player seen by the app. |
| `hist_matches` | One row per match, keyed by local ID and optionally by Rocket League match GUID. |
| `hist_player_match_stats` | Per-player stats for a specific match. |
| `hist_goal_events` | Goal log for a match, including scorer, assister, speed, and game time. |
| `hist_ball_hit_events` | Optional high-volume ball touch telemetry. |
| `hist_tick_snapshots` | Reserved for raw tick snapshot storage. |
| `hist_statfeed_events` | Statfeed/accolade events for match detail views. |

The match list endpoint derives `player_count` from `hist_player_match_stats`, and derives team goals from saved team scores when available. If final team scores were not stored, it falls back to summing player goals by team.

---

## API Routes

### `GET /api/players`

Returns all players seen by History.

Example:

```bash
curl http://localhost:8080/api/players
```

### `GET /api/matches`

Returns the latest matches, ordered by start time descending.

Example:

```bash
curl http://localhost:8080/api/matches
```

### `GET /api/matches?player={primaryID}`

Returns only matches where the given player appears in `hist_player_match_stats`.

Example:

```bash
curl "http://localhost:8080/api/matches?player=steam%7C76561198000000000"
```

### `GET /api/matches/{id}`

Returns one match detail payload:

- `match`
- `players`
- `goals`
- `events`

Example:

```bash
curl http://localhost:8080/api/matches/123
```

---

## UI Integration

The History tab is implemented by `internal/plugins/history/view.html` and `internal/plugins/history/view.js`.

The current UI flow is:

1. Load all players from `/api/players`.
2. Populate the player filter.
3. Load match rows from `/api/matches`.
4. Filter invalid placeholder rows with missing arena names.
5. Expand a selected match inline.
6. Load match detail from `/api/matches/{id}`.
7. Render player stats, goal events, and statfeed events.

The History page may also trigger other read-side behavior indirectly. For example, when match detail renders player rows, rank/MMR UI code may look up ranks for the players shown in that match.

---

## Impact On Live, Session, And History

### Live

The History plugin listens to the same event stream as Live, but it should not block or delay Live UI updates. History writes are local SQLite operations and should remain lightweight.

### Session

Session summaries rely on completed match records being accurate. If History flushes a match as incomplete or stores the wrong player roster, session totals can be missing or incorrect.

### History

History is the main read model for stored match data. Wrong match boundaries, stale players, missing goal events, or incorrect team scores will show up directly in the History tab.

Because of this, changes to History should be tested with real match lifecycles, not only static API calls.

---

## Known Limitations And Current Issues

- The plugin is stateful, so future lifecycle changes should preserve the explicit match GUID and roster snapshot protections.
- Late replay/post-match payloads can contain partial roster snapshots. Current code protects against shrinking the stored roster from smaller replay snapshots, but new Rocket League API payload shapes should still be playtested.
- `MatchDestroyed` is treated as incomplete when no `MatchEnded` event arrives. This is useful for disconnects and private-match edge cases, but it can also create expected incomplete rows during unusual test flows.
- Forfeit detection is inferred from `MatchEnded` firing while clock time remains and the match is not in overtime. This is practical, but it depends on reliable final clock state.
- Playlist data can be missing from Rocket League payloads. When that happens, UI code may infer the match type from player count.
- `BallHit` persistence is disabled unless configured because it can create a large amount of data.
- Many event write errors are intentionally ignored to avoid interrupting live match tracking. This keeps the app resilient, but it can hide persistence failures unless logs or tests catch them.

---

## Technology Decisions

- History is implemented as a plugin so match storage can remain isolated from core Rocket League connection code.
- SQLite is used as the durable local store because the app is a local desktop tracker.
- Tables are prefixed with `hist_` to avoid collisions with other plugins.
- `match_guid` is unique when present so repeated updates for the same Rocket League match resolve to one local match row.
- Active match GUID checks prevent delayed events from writing to the wrong match after a transition.
- Roster snapshots replace the active player map instead of accumulating indefinitely, with safeguards for smaller replay snapshots.
- The plugin stores final per-player stats at match flush instead of writing every tick.
- Goal and statfeed events are stored as event rows because they are useful for match detail timelines.
- High-volume ball hit records are optional to keep storage size under control.

---

## Technical Debt

- History lifecycle handling has regression coverage for stale GUIDs, roster replacement, and late partial replay snapshots. It still needs broader coverage around forfeits, private matches, abandoned matches, overtime, and mode changes.
- The plugin currently stores active state in memory without an explicit active-match object. A small internal match-state type could make future lifecycle changes easier to reason about.
- Error handling for persistence writes is minimal. A future pass could log selected failures without risking Live or Session performance.
- `hist_tick_snapshots` exists in the schema, but tick snapshot persistence is not part of the normal runtime path.
- The schema is embedded in plugin code. As the app matures, explicit migrations may be easier to review than large schema strings.
- Some data, such as match type, may be inferred by UI from player count when playlist data is missing. That can be misleading if stale players exist.

---

## Testing

Useful automated checks:

```bash
go test ./internal/plugins/history
go test ./...
go vet ./...
```

Useful manual test plan:

1. Start OOF RL with a clean log.
2. Play a normal completed match.
3. Confirm History creates one completed match row.
4. Expand the match and confirm all players, teams, stats, goal events, assists, goal speeds, and goal times are present.
5. Play a second match in a different playlist size.
6. Confirm the second match does not contain players from the first match.
7. Test an overtime match.
8. Test a forfeit.
9. Test a private or bot match that exits through an unusual path.
10. Compare `oof_rl.log`, raw capture metadata, and `hist_player_match_stats` if any player count looks wrong.

---

## Troubleshooting

When History data looks wrong, inspect these sources:

- `oof_rl.log` for match lifecycle events and plugin errors.
- Raw capture metadata, if available, for the roster Rocket League actually reported.
- `hist_matches` for match GUID, arena, winner, incomplete state, overtime, forfeit, playlist, and team scores.
- `hist_player_match_stats` for the final roster and per-player stats.
- `hist_goal_events` for goal speed, goal time, scorer, and assister values.
- `hist_statfeed_events` for feed/timeline events.

Common symptoms:

- A 2v2 shows as 3v3 or 4v4: check for stale players in `hist_player_match_stats`.
- Session stats are missing: check whether the match was marked incomplete.
- Match detail triggers rank lookups for unexpected players: the stored match roster likely contains stale players.
- Goal event time looks wrong: compare `hist_goal_events.goal_time` with the UI rendering logic before changing storage code.
