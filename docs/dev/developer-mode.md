# Developer Mode

Developer mode enables profiling and runtime inspection endpoints. It is off by default and not intended for end users.

## Enabling

Add to `%LOCALAPPDATA%\OOF_RL\config.toml`:

```toml
dev_mode = true
```

Then restart the app. There is no Settings UI toggle — the config file is the only way to enable it.

## What it enables

| Endpoint | Description |
|----------|-------------|
| `/debug/pprof/` | Go pprof profiling — CPU, heap, goroutines, block, mutex |
| `/debug/statsviz/` | Live runtime metrics visualizer (goroutines, GC, memory) |

The HTTP write timeout is also extended from 30 s to 90 s to allow pprof CPU profiles to stream to completion without racing the response deadline.

All endpoints are local-only — the server binds to `localhost` by default.

## Developer settings in plugins

Plugin settings marked `developer: true` in their schema are hidden in a collapsed **Developer** section in the Settings UI. They work the same as normal settings; the flag just keeps them out of the way for regular users.

## Profiling a plugin

1. Enable dev mode and restart.
2. Open `http://localhost:8080/debug/pprof/` in a browser.
3. Use `go tool pprof` for CPU or heap profiles:

```sh
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=15
go tool pprof http://localhost:8080/debug/pprof/heap
```