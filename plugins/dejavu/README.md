# Deja Vu

Deja Vu is a read-only current-roster history recall plugin for OOF RL.

It shows prior saved-match history for current roster players relative to the
player explicitly selected in Session. OOF RL does not currently have a
backend-authoritative account identity for this purpose, so Deja Vu uses the
existing Session-selected tracked player and does not guess by name, team,
frequency, or roster order.

## MVP Behavior

- The view reads `localStorage.oof_session_player`, the same Session selection
  key used by the Session tab.
- The view never writes, clears, or auto-populates that key.
- The plugin subscribes to live `state.updated` events and keeps only the
  current roster in memory.
- Historical matching uses exact stable `PrimaryId` values only.
- Players without a usable stable live ID remain visible, but their history
  metrics are unavailable.
- The Session-selected tracked player must be present in the current roster and
  assigned to team `0` or `1` before history queries run.
- Team `0` and team `1` are the only classified playing teams.
- The active match is excluded by `hist_matches.match_guid` matching the live
  match GUID.
- All playlists are included in the MVP.

## Metrics

For selected player `S`, target player `P`, and historical match `M`, a prior
encounter counts when:

- `S` and `P` each have a `hist_player_match_stats` row in `M`.
- Both rows have team numbers `0` or `1`.
- `S.primary_id != P.primary_id`.
- `M.match_guid` is not the current live match GUID.
- The match has not already been counted for that target.

Historical relationship is classified per historical match:

- `with`: selected and target historical team numbers are equal.
- `against`: selected and target historical team numbers differ.

W/L mirrors Session's current result rule:

- Result-eligible means `incomplete` is false and `winner_team_num >= 0`.
- A win means the selected player's historical team equals `winner_team_num`.
- Otherwise the result-eligible encounter is a loss.
- Encounters without a usable result still count as prior encounters and are
  exposed as no-result counts.

## Read-Only Boundary

Deja Vu uses `sdk.DBQuery` only. It performs no:

- SQLite writes
- Schema migrations
- Index creation
- Settings writes
- Sidecar storage
- Session, Live, History, overlay, saved-match, or replay-capture mutation

The plugin caches a single read result by current match GUID, selected tracked
player ID, and the sorted set of current stable target IDs. Score-only and stat-
only live updates reuse the existing cached history result.

## Deferred Work

- Notes
- Party detection
- Playlist filters
- Searchable all-player history
- Live tab integration
- Overlay or HUD rendering
- Identity heuristics or alias merging

## Inspiration

This feature is inspired by AdamK33n3r's BakkesMod Deja Vu plugin. The OOF RL
implementation is independent and uses OOF RL's event, SQLite, WASM plugin, and
vanilla JavaScript view conventions. No source code or substantial
implementation structure from the reference plugin is copied here.
