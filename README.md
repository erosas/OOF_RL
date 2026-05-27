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

Full setup details: [docs/install.md](docs/install.md)

---

## Documentation

| Page | Description |
|------|-------------|
| [Architecture RFC](docs/architecture-rfc.md) | Locked platform decisions for plugin identity, lifecycle, and ownership |
| [Plugin Ownership & Trust Model](docs/plugin-ownership.md) | Feature ownership matrix and WASM plugin trust model |
| [Install & Setup](docs/install.md) | Full setup, RL Stats API configuration, settings reference |
| [Writing a Plugin](docs/plugins.md) | How to add a new tab or feature to OOF RL |
| [WASM Plugin ABI](docs/wasm-plugins.md) | Host imports, SDK helpers, and plugin lifecycle |
| [Configuration](docs/configuration.md) | All `config.toml` fields explained |
| [HTTP API](docs/api.md) | REST endpoints and WebSocket event format |

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

OOF RL features are delivered as [WASM plugins](docs/wasm-plugins.md) compiled to `wasip1/wasm` and loaded at startup from `%LOCALAPPDATA%\OOF_RL\plugins`. Each plugin is a single `.wasm` binary plus optional static assets. Plugins are isolated in their own Go module under `plugins/<name>/` and share a common SDK at `plugins/sdk`.

See [docs/plugins.md](docs/plugins.md) for the full plugin authoring guide.

Releases are published automatically by GitHub Actions when a version tag is pushed:

```
git tag v1.2.3
git push origin v1.2.3
```
