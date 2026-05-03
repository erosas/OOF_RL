# OOF RL

A local Rocket League companion app. Connects to the official RL Stats API, tracks match history in an embedded SQLite database, and shows live stats in a desktop window — all from a single `.exe`, no install required.

**Plugins included:** Live scoreboard · Match history · Player ranks · Session tracker · Ballchasing.com upload

---

## Install

1. Download `oof_rl.exe` from the [Releases](../../releases) page
2. Double-click it — a window opens automatically
3. Go to **Settings** and click **Write INI** to enable the RL Stats API, then restart Rocket League

That's it. No installer, no runtime dependencies, no database server.

> First run creates `config.toml` and `oof_rl.db` in `%LOCALAPPDATA%\OOF_RL`.  
> WebView2 is required — it ships with Windows 11 and is auto-installed on Windows 10 via Windows Update.

Full setup details: [docs/install.md](docs/install.md)

---

## Documentation

| Page | Description |
|------|-------------|
| [Install & Setup](docs/install.md) | Full setup, RL Stats API configuration, settings reference |
| [Writing a Plugin](docs/plugins.md) | How to add a new tab or feature to OOF RL |
| [Configuration](docs/configuration.md) | All `config.toml` fields explained |
| [HTTP API](docs/api.md) | REST endpoints and WebSocket event format |

---

## Development

Prerequisites: Go 1.22+, Windows (WebView2 is Windows-only)

```
go run .          # run from source
go test ./...     # run all tests
make build        # produces oof_rl.exe
make cover        # generate coverage.html
```

Releases are published automatically by GitHub Actions when a version tag is pushed:

```
git tag v1.2.3
git push origin v1.2.3
```
