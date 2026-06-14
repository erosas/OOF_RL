# OOF RL

[![CI](https://github.com/erosas/OOF_RL/actions/workflows/ci.yml/badge.svg)](https://github.com/erosas/OOF_RL/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/erosas/OOF_RL/branch/main/graph/badge.svg)](https://codecov.io/gh/erosas/OOF_RL)
[![VirusTotal](https://img.shields.io/badge/VirusTotal-scanned-blue?logo=virustotal)](https://github.com/erosas/OOF_RL/releases/latest)

A local Rocket League companion: live scoreboard, match history, player ranks, session tracking, and an in-game overlay. Runs on your PC — no account, no installer, no setup beyond the four steps below.

## Quick start (≈30 seconds)

1. **Download** the latest `.zip` from [**Releases**](../../releases/latest).
2. **Extract** it and double-click **`oof_rl.exe`**.
3. In the app, open **Settings → RL API → Write INI**.
4. **Restart Rocket League.** That's it — your stats show up automatically.

> 💡 Press **F9** anytime to toggle the overlay on top of your game.

<sub>Windows 10/11. Needs WebView2 (preinstalled on Windows 11; auto-installed on Windows 10). First run creates a data folder at `%LOCALAPPDATA%\OOF_RL`.</sub>

**Stuck?** See [full setup & troubleshooting](docs/user/install.md). Tweaking settings by hand? See [Configuration](docs/user/configuration.md).

---

<details>
<summary><b>For developers</b> — build from source, plugin architecture, API, releases</summary>

<br>

### Build & run

Prerequisites: Go 1.26+, Windows (WebView2 is Windows-only).

```
go run .             # run from source
go test ./...        # host tests only
make test-all        # host + SDK + all plugin tests
make build           # produces oof_rl.exe
make release-package # produces dist/OOF_RL.zip with exe + public plugins
make all-plugins     # compile all WASM plugins and install to %LOCALAPPDATA%\OOF_RL\plugins
make cover           # generate coverage.html
```

### Plugin architecture

Features are delivered as WASM plugins compiled to `wasip1/wasm` and loaded at startup from `%LOCALAPPDATA%\OOF_RL\plugins`. Release packages bundle the public plugins beside `oof_rl.exe`; on startup the app installs or updates those bundled files in the app-data plugin directory before loading. Unknown app-data plugins are preserved. Each plugin is a single `.wasm` binary plus optional static assets, isolated in its own Go module under `plugins/<name>/` and built against the shared SDK at `sdk/`.

The release bundle ships Live, Match history, Ranks, Session, and Dashboard. Ballchasing is not bundled but still loads if manually installed or developer-built.

### Releases

Published automatically by GitHub Actions when a version tag is pushed. A plain tag is a stable release; a `-dev.N` suffix is a dev-channel prerelease.

```
git tag v1.2.3
git push origin v1.2.3
```

### Documentation

| Page | Description |
|------|-------------|
| [HTTP API](docs/dev/api.md) | REST endpoints and WebSocket event format |
| [WASM Plugin ABI](docs/dev/wasm-plugins.md) | Building plugins: ABI, SDK, lifecycle, testing |
| [Plugin Ownership & Trust Model](docs/dev/plugin-ownership.md) | Feature ownership matrix and WASM trust model |
| [Event Bus](docs/dev/event-bus.md) | OOFEvent model, Certainty, PluginBus interface |
| [MMR Providers](docs/dev/mmr-providers.md) | Adding or extending rank lookup providers |
| [Developer Mode](docs/dev/developer-mode.md) | pprof, statsviz, and developer settings |
| [Update Checker](docs/dev/auto-update.md) | Stable/dev channels, manual checks, trust boundary |
| [Release Readiness](docs/dev/release-readiness.md) | Pre-tag blockers, known risks, and release smoke checklist |

</details>
