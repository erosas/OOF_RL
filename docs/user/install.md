# Install & Setup

## Requirements

- Windows 10 or 11
- [WebView2 Runtime](https://developer.microsoft.com/en-us/microsoft-edge/webview2/) — pre-installed on Windows 11; auto-installed on Windows 10 via Windows Update. If the app fails to open, download and run the Evergreen Bootstrapper from that page.
- Rocket League (Steam or Epic)

---

## 1. Download

Grab `oof_rl.exe` from the [Releases](../../releases) page. No installer — just put it anywhere you like and double-click.

On first launch the app creates its data directory at `%LOCALAPPDATA%\OOF_RL\` containing:

| File | Description |
|------|-------------|
| `config.toml` | App settings |
| `oof_rl.db` | SQLite match database |
| `oof_rl.log` | Log output (detached from console) |
| `captures/` | Raw packet captures (if enabled) |

---

## 2. Enable the RL Stats API

Rocket League has a built-in stats broadcasting API that OOF RL reads from. You need to enable it once.

**The easy way:** Open OOF RL, go to **Settings → RL API**, fill in your RL install path if needed, and click **Write INI**. Then restart Rocket League.

**Manually:** Create or edit this file:

```
%USERPROFILE%\Documents\My Games\Rocket League\TAGame\Config\DefaultStatsAPI.ini
```

(If you use OneDrive for Documents, the path starts with `%USERPROFILE%\OneDrive\Documents\...`)

```ini
[TAGame.MatchStatsExporter_TA]
PacketSendRate=60
Port=49123
```

- `PacketSendRate` must be greater than 0. 60 is recommended; 120 is the maximum.
- `Port` defaults to 49123. Only change it if something else is using that port.
- The section name `[TAGame.MatchStatsExporter_TA]` must be exact — other names are silently ignored by RL.
- **Restart Rocket League after any INI change.**

---

## 3. Run

Double-click `oof_rl.exe`. The app:

- Opens a desktop window (WebView2)
- Connects to the RL TCP socket immediately and retries every 5 seconds
- Shows a green status dot in the top-right corner when connected

You can also run multiple Rocket League sessions or restart RL freely — OOF RL reconnects automatically.

---

## Overlay

OOF RL includes a borderless overlay window that floats above your game. Toggle it with **F9** (configurable in Settings).

- **Drag** it by the handle at the top
- **Resize** it from the bottom-right corner
- **Opacity** slider in the overlay controls its transparency
- **Hold mode** keeps it visible only while the hotkey is held down

The overlay shows the same live view as the main window but stays on top of full-screen windowed mode.

---

## Troubleshooting

**OOF RL opens but the status dot is always red**
- Check that `DefaultStatsAPI.ini` exists and has `PacketSendRate=60`
- Confirm Rocket League was restarted after writing the INI
- Check that nothing else is listening on port 49123

**"Failed to create webview" error at startup**
- Install the [WebView2 Runtime](https://developer.microsoft.com/en-us/microsoft-edge/webview2/)

**Blank window or white screen**
- Check `%LOCALAPPDATA%\OOF_RL\oof_rl.log` for errors

**Ballchasing auto-upload isn't working**
- Make sure you've entered a valid API key in **Settings → Ballchasing**
- Replays are uploaded 5 seconds after `MatchDestroyed` fires — wait a moment after leaving the post-match screen
