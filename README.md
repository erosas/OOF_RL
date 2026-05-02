# OOF RL

A local Rocket League stats collector with a web dashboard. Connects to the official Rocket League Stats API, stores match data in an embedded SQLite database, and serves a live dashboard in your browser — all from a single binary.

---

## Download

Grab the latest `oof_rl.exe` from the [Releases](../../releases) page. No installer required — just double-click.

---

## How It Works

The Rocket League client exposes a local TCP server that broadcasts game events in real time. OOF RL connects to that socket, fans all events out to any open browser tabs (for live display), and optionally persists them to SQLite based on your storage preferences.

> **Protocol note:** The RL Stats API is raw TCP with concatenated JSON objects, not a WebSocket. Each message is `{"Event":"EventName","Data":"<JSON-encoded string>"}` — the `Data` field is itself a JSON-encoded string that wraps the actual payload.

---

## Setup

### 1. Enable the RL Stats API

Create (or edit) the file below **before launching Rocket League**:

```
<UserDocuments>\My Games\Rocket League\TAGame\Config\DefaultStatsAPI.ini
```

(`<UserDocuments>` is typically `C:\Users\<you>\OneDrive\Documents` or `C:\Users\<you>\Documents`)

```ini
[TAGame.MatchStatsExporter_TA]
PacketSendRate=60
Port=49123
```

- The section name `[TAGame.MatchStatsExporter_TA]` is required — other section names are silently ignored by RL.
- `PacketSendRate` must be > 0 to open the socket (max 120; 60 is recommended).
- `Port` defaults to 49123.

You can also do this from the **Settings** tab in the dashboard — it writes the file for you.

> **Note:** Changes to the INI require a Rocket League restart.

### 2. Run the App

Double-click `oof_rl.exe` (or `go run .` during development). The app will:

- Open `http://localhost:8080` in your default browser automatically
- Try to connect to the RL TCP socket immediately and retry every 5 seconds
- Create `config.toml` and `oof_rl.db` next to the binary on first run

---

## Configuration

`config.toml` is created automatically with defaults. All settings are also editable from the **Settings** tab.

```toml
app_port = 8080
rl_install_path = ""   # auto-detected for Steam/Epic on first run
db_path = "oof_rl.db"
open_in_browser = false  # true = use system browser with DevTools; false = embedded WebView2 window

[storage]
match_metadata    = true   # match start/end, arena, winner
player_match_stats = true  # per-player goals/shots/saves/assists/demos — written once at match end
goal_events       = true   # goal scorer, speed, location, assister
ball_hit_events   = false  # every ball hit (high volume — up to 7,200 rows/min at 120 Hz)
tick_snapshots    = false  # raw UpdateState JSON (very high volume — ~35 MB per 5-min match)
tick_snapshot_rate = 1.0   # snapshots per second when enabled (0 = all)
other_events      = true   # crossbar hits, pause/unpause, etc.
raw_packets       = false  # keep all RL envelopes in memory, flush at match end
raw_packets_dir   = "captures" # each match gets packets_normalized_XXX.ndjson + packets_wire_XXX.ndjson + capture_meta.json + capture_index.json
```

Saving config or writing the INI from the Settings tab immediately triggers a reconnect attempt.

---

## Dashboard

| Tab | Description |
|-----|-------------|
| **Live** | Real-time scoreboard, boost bars, clock, supersonic/demolished states. Streams directly from the TCP socket — always on, regardless of storage settings. |
| **History** | List of stored matches. Filter by player. Click a match to see per-player stats and goal log. |
| **Players** | All seen players with aggregated stats (goals, assists, saves, shots, demos across all stored matches). Click a player to drill into their match history. |
| **Settings** | Edit app config (storage toggles, RL path, port) and write `DefaultStatsAPI.ini` directly. |

The status dot in the top-right corner shows RL connection state (green = connected, red = waiting).

---

## Platform Notes

### Steam
Tracker.gg profiles are looked up by the numeric Steam64 ID embedded in the player's `PrimaryId`. No display name is needed.

### Epic, PlayStation, Xbox
Tracker.gg profiles are looked up by the player's display name. The `name` parameter is required for accurate results.

### Nintendo Switch
Switch players whose privacy settings are enabled will have their identity masked by Rocket League (both the ID and display name appear as `****`). OOF RL detects this and skips tracker.gg lookups for masked players. Non-masked Switch players are supported normally.

---

## Architecture

```
oof_rl.exe
├── internal/config    config.toml read/write + DefaultStatsAPI.ini read/write
├── internal/db        SQLite schema, migrations, all queries (modernc.org/sqlite — no CGo)
├── internal/events    Go structs for all RL event/tick types
├── internal/hub       Fan-out hub: RL events → all connected browser WebSocket clients
├── internal/rl        TCP client (auto-connect, reconnect, event routing, storage)
└── internal/server    HTTP server, browser WebSocket upgrade, REST API
web/                   Embedded HTML/CSS/JS dashboard (no build step)
```

### API Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/ws` | Browser WebSocket — live event stream |
| GET | `/api/players` | All known players |
| GET | `/api/players/{id}` | Player aggregate stats + match list |
| GET | `/api/matches?player=` | Match list, optional player filter |
| GET | `/api/matches/{id}` | Match detail: player stats + goals |
| GET/POST | `/api/config` | Read/write app config |
| GET/POST | `/api/config/ini` | Read/write DefaultStatsAPI.ini |
| GET | `/api/replays` | Local Rocket League `.replay` files from Documents Demos folder |
| GET | `/api/captures` | List local capture directories |
| GET | `/api/captures/{id}/meta` | Read `capture_meta.json` |
| GET | `/api/captures/{id}/index` | Read `capture_index.json` |
| GET | `/api/captures/{id}/events` | Concatenated normalized NDJSON stream |
| GET | `/api/tracker/profile?id=&name=` | Fetch tracker.gg profile (cached) |
| GET | `/api/ballchasing/replays` | List replays from ballchasing.com |
| POST | `/api/ballchasing/upload` | Upload a local replay to ballchasing.com |
| GET | `/api/ballchasing/uploads` | Local upload history |
| GET | `/api/ballchasing/ping` | Validate ballchasing API key |

### Database Schema

```
players             (primary_id PK, name, last_seen)
matches             (id, match_guid, arena, started_at, ended_at, winner_team_num, overtime, playlist_type)
player_match_stats  (match_id, primary_id, team_num, score, goals, shots, assists, saves, touches, car_touches, demos)
goal_events         (match_id, scorer_id, scorer_name, assister_id, assister_name, ball_last_touch_id, goal_speed, goal_time, impact_x/y/z, scored_at)
ball_hit_events     (match_id, player_id, pre/post_hit_speed, loc_x/y/z, hit_at)
tick_snapshots      (match_id, captured_at, raw_json)
tracker_cache       (primary_id PK, data_json, fetched_at)
bc_uploads          (replay_name PK, ballchasing_id, bc_url, uploaded_at)
```

### Performance Notes

- The RL Stats API sends ~2 KB per tick at 60 Hz — **~35 MB per 5-minute match** if you store every tick raw. Keep `tick_snapshots = false` unless you specifically need it.
- Player stats are cached in memory during a match and written to the DB **once at match end**, avoiding 360+ SQLite writes/second that would impact game performance.
- Live view data flows entirely through the in-memory fan-out hub — no DB reads on the hot path.

---

## Development

### Prerequisites

- Go 1.26+
- Windows (required — the embedded WebView2 window uses `go-webview2`, Windows-only)
- [WebView2 Runtime](https://developer.microsoft.com/en-us/microsoft-edge/webview2/) (pre-installed on Windows 11; use `open_in_browser = true` in `config.toml` to skip it)

### Build & Run

```
make build   # produces oof_rl.exe
make run     # build + run
make test    # run all tests
make cover   # generate coverage report (coverage.html)
```

Or directly:

```
go run .
go test ./...
```

### Releases

Releases are built automatically by GitHub Actions when a version tag is pushed:

```
git tag v1.2.3
git push origin v1.2.3
```

This runs all tests on `windows-latest`, then builds `oof_rl.exe` with `-H windowsgui -s -w` and publishes it as a GitHub Release with a SHA-256 checksum.

### Repository Layout

Files **not** committed (see `.gitignore`):

| Pattern | Reason |
|---------|--------|
| `captures/` | Raw packet captures — can be large and machine-specific |
| `*.db` | SQLite databases contain user data |
| `*.exe` | Compiled binaries belong in Releases |
| `*.pdf` | API documentation PDFs |
| `config.toml` | Machine-specific settings; may contain API keys |
| `coverage.*` | Build artifacts |

---

## What Works

- [x] Single binary — run `oof_rl.exe`, browser opens automatically
- [x] Auto-connect to RL TCP socket with 5s reconnect loop
- [x] Reconnect triggered immediately when config or INI is saved from Settings
- [x] Live dashboard: scoreboard, team scores, boost bars, supersonic indicator, demolished state, clock, overtime
- [x] Goal flash banner on `GoalScored` event
- [x] Storage toggle per event type (match metadata, player stats, goals, ball hits, ticks)
- [x] Player stats cached in memory, flushed to DB once at match end (no per-tick writes)
- [x] Match history with player filter dropdown
- [x] Match detail: per-player stats table + goal log
- [x] Player list with aggregate stats (goals, assists, saves, shots, demos, match count)
- [x] Player detail: aggregate + recent match list
- [x] Settings UI: edit all config fields, write DefaultStatsAPI.ini
- [x] Auto-detect RL install path (Steam and Epic common paths)
- [x] SQLite embedded DB — no external database needed
- [x] All web assets embedded in the binary (no separate files needed at runtime)
- [x] tracker.gg profile lookup with DB cache and rate limiting
- [x] Ballchasing.com integration: browse replays/groups, upload replay files
- [x] Raw packet capture: NDJSON dumps with metadata and event index for replay/testing
- [x] Platform-aware tracker lookups (Steam uses ID, others use display name; masked Switch players skipped)
- [x] Unit tests for all internal packages (`config`, `db`, `events`, `hub`, `rl`, `server`)
- [x] GitHub Actions CI: tests on every PR + release binary on version tags

---

## TODO

### UI Enhancements

- [ ] Use actual team colors from API (`ColorPrimary`/`ColorSecondary` fields in Teams data)
- [ ] Boost active indicator (`bBoosting`), on-wall indicator (`bOnWall`), in-air state
- [ ] Player speed display (available in `Speed` field per player)
- [ ] Ball speed in match header (available in `Ball.Speed` per tick)

### Known Issues / Polish

- [ ] `StatfeedEvent` not yet handled (awards/accolades feed)
- [ ] Ball hit storage has no rate limiting — at 120 Hz this is ~7,200 rows/minute
- [ ] Tick snapshot rate limiter declared in config but not enforced in `rl/client.go`
- [ ] No pagination on History or Players lists (capped at 200 matches)
- [ ] Player names can change across matches — history shows last known name only
- [ ] Settings: RL install path browse button (currently text input only)
- [ ] App port change requires a manual restart (by design, noted in UI)