# Configuration

OOF RL stores its settings in `%LOCALAPPDATA%\OOF_RL\config.toml`. The file is created automatically with defaults on first run. All fields are also editable from the **Settings** tab in the app.

---

## App Settings

```toml
app_port = 8080
data_dir = ""         # defaults to %LOCALAPPDATA%\OOF_RL
rl_install_path = ""  # auto-detected on first run; used to write DefaultStatsAPI.ini
```

| Field | Default | Description |
|-------|---------|-------------|
| `app_port` | `8080` | HTTP port for the local server. If the port is taken, OOF RL tries the next 19 ports automatically. Requires a restart to change. |
| `data_dir` | `%LOCALAPPDATA%\OOF_RL` | Where the database, log, and captures are stored. |
| `rl_install_path` | auto | Path to the Rocket League install directory (the folder containing `Binaries\`). Used only to write `DefaultStatsAPI.ini`. |

---

## Tracker Cache

```toml
tracker_cache_ttl_minutes = 5
```

| Field | Default | Min | Description |
|-------|---------|-----|-------------|
| `tracker_cache_ttl_minutes` | `5` | `2` | How long tracker.gg rank results are cached in the database before re-fetching. |

---

## Overlay

```toml
overlay_hotkey   = "F9"
overlay_x        = -1
overlay_y        = -1
overlay_width    = 860
overlay_height   = 620
overlay_opacity  = 1.0
overlay_hold_mode = false
```

| Field | Default | Description |
|-------|---------|-------------|
| `overlay_hotkey` | `"F9"` | Key that toggles (or holds) the overlay. Supported keys: F1–F12, Insert, Delete, Home, End, PageUp, PageDown, Pause, ScrollLock. |
| `overlay_x`, `overlay_y` | `-1` | Overlay position in screen pixels. `-1` = centered on the primary monitor. |
| `overlay_width`, `overlay_height` | `860 × 620` | Overlay size in pixels. |
| `overlay_opacity` | `1.0` | Opacity from 0.04 (nearly invisible) to 1.0 (fully opaque). |
| `overlay_hold_mode` | `false` | `true` = overlay is visible only while the hotkey is held. `false` = hotkey toggles show/hide. |

---

## Storage

```toml
[storage]
ball_hit_events    = false
tick_snapshots     = false
tick_snapshot_rate = 1.0
raw_packets        = false
```

| Field | Default | Description |
|-------|---------|-------------|
| `ball_hit_events` | `false` | Record every ball touch. High volume: up to 7,200 rows/minute at 120 Hz. |
| `tick_snapshots` | `false` | Store the full game state JSON at regular intervals. Very high volume: ~35 MB per 5-minute match at 60 Hz. |
| `tick_snapshot_rate` | `1.0` | Snapshots per second when `tick_snapshots = true`. |
| `raw_packets` | `false` | Save raw RL TCP packets to disk. Each match gets a directory under `captures/` with `packets_normalized_*.ndjson`, `packets_wire_*.ndjson`, `capture_meta.json`, and `capture_index.json`. Useful for debugging or offline replay. |

Match metadata, player stats, and goal events are always stored — there is no toggle for them.

---

## Ballchasing

```toml
ballchasing_api_key = ""
```

Your [ballchasing.com](https://ballchasing.com) API key. Get one at ballchasing.com/upload after creating an account. Without this key, the Ballchasing tab will show an error and auto-upload will not run.

---

## Plugin Visibility

```toml
disabled_plugins = []
```

A list of plugin IDs to hide from the nav bar. Example: `disabled_plugins = ["bc", "session"]`. Disabled plugins still load — they just don't appear in the UI.
