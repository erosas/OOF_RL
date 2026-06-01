# OOF RL

[![CI](https://github.com/erosas/OOF_RL/actions/workflows/ci.yml/badge.svg)](https://github.com/erosas/OOF_RL/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/erosas/OOF_RL/branch/main/graph/badge.svg)](https://codecov.io/gh/erosas/OOF_RL)
[![VirusTotal](https://img.shields.io/badge/VirusTotal-scanned-blue?logo=virustotal)](https://github.com/erosas/OOF_RL/releases/latest)

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

## Overlay

Press **F9** (configurable) to toggle a borderless overlay window that floats above the game. Drag it by the handle at the top, resize from the bottom-right corner, and adjust opacity from the overlay controls. Hold mode keeps it visible only while the hotkey is held.

For full setup details and troubleshooting: [docs/user/install.md](docs/user/install.md)

---

## Documentation

**For users**

| Page | Description |
|------|-------------|
| [Install & Setup](docs/user/install.md) | RL Stats API setup, overlay controls, troubleshooting |
| [Configuration](docs/user/configuration.md) | All `config.toml` fields explained |

**For developers**

| Page | Description |
|------|-------------|
| [HTTP API](docs/dev/api.md) | REST endpoints and WebSocket event format |
| [WASM Plugin ABI](docs/dev/wasm-plugins.md) | Building plugins: ABI, SDK, lifecycle, testing |
| [Plugin Ownership & Trust Model](docs/dev/plugin-ownership.md) | Feature ownership matrix and WASM trust model |
| [Event Bus](docs/dev/event-bus.md) | OOFEvent model, Certainty, PluginBus interface |
| [MMR Providers](docs/dev/mmr-providers.md) | Adding or extending rank lookup providers |
| [Developer Mode](docs/dev/developer-mode.md) | pprof, statsviz, and developer settings |

---

## Development

Prerequisites: Go 1.26+, Windows (WebView2 is Windows-only)

```
go run .             # run from source
go test ./...        # host tests only
make test-all        # host + SDK + all plugin tests
make build           # produces oof_rl.exe
make all-plugins     # compile all WASM plugins and install to %LOCALAPPDATA%\OOF_RL\plugins
make cover           # generate coverage.html
```

### Plugin architecture

Features are delivered as WASM plugins compiled to `wasip1/wasm` and loaded at startup from `%LOCALAPPDATA%\OOF_RL\plugins`. Each plugin is a single `.wasm` binary plus optional static assets, isolated in its own Go module under `plugins/<name>/` and built against the shared SDK at `sdk/`.

See [docs/dev/wasm-plugins.md](docs/dev/wasm-plugins.md) for the full plugin authoring guide.

Releases are published automatically by GitHub Actions when a version tag is pushed:

```
git tag v1.2.3
git push origin v1.2.3
```
