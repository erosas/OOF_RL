.PHONY: build run rebuild test cover icon profile profile-heap profile-goroutine \
        all-plugins release-package test-all test-plugins test-sdk $(addprefix wasm/, $(PLUGINS)) $(addprefix test-plugin/, $(PLUGINS))

PORT    ?= 8080
PLUGINS := live ballchasing ranks session dashboard
VERSION ?=

# WASM plugins are installed into the same data directory the app reads at runtime.
# LOCALAPPDATA is inherited from the Windows environment (e.g. C:\Users\you\AppData\Local).
PLUGINS_DIR := $(LOCALAPPDATA)/OOF_RL/plugins

build:
	go build -o oof_rl.exe .

release-package:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/package-release.ps1 -Version "$(VERSION)"

# Regenerates icon.ico, then embeds it together with Windows VERSIONINFO
# (CompanyName/ProductName/FileDescription/version) into rsrc.syso. The version
# block is required: an exe with no version resource is a common antivirus
# false-positive trigger. See docs/dev/antivirus.md.
#   go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
icon:
	go run ./tools/genicon
	$(shell go env GOPATH)/bin/goversioninfo -o rsrc.syso -icon icon.ico versioninfo.json

run: build
	./oof_rl.exe

# Host tests only
test:
	go test ./...

# Packages intentionally excluded from coverage (keep alphabetical):
#   OOF_RL                    — main package; app entry point and OS glue only
#   OOF_RL/internal/overlay   — Windows WebView2/CGO UI; requires a real window, not unit-testable
#   OOF_RL/internal/rl        — live WebSocket to the RL game process; requires the game running
#   OOF_RL/tools/genicon      — one-off build tool; no application logic worth testing
COVER_PKGS := $(shell go list ./... | grep -Ev "^(OOF_RL|OOF_RL/internal/overlay|OOF_RL/internal/rl|OOF_RL/tools/genicon)$$")

cover:
	go test -coverprofile=coverage.out $(COVER_PKGS)
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report written to coverage.html"

# --- WASM plugin targets ---

# Build a single WASM plugin and install it directly into the app data directory.
# GOOS/GOARCH are exported by make so the go command inherits them without shell env-var syntax.
wasm/%: export GOOS := wasip1
wasm/%: export GOARCH := wasm
wasm/%:
	@powershell -NoProfile -Command "New-Item -ItemType Directory -Force '$(PLUGINS_DIR)' | Out-Null"
	go -C plugins/$* build -buildmode=c-shared -o "$(PLUGINS_DIR)/$*.wasm" .
	@powershell -NoProfile -Command "if (Test-Path 'plugins/$*/assets') { New-Item -ItemType Directory -Force '$(PLUGINS_DIR)/$*' | Out-Null; Copy-Item -Recurse -Force 'plugins/$*/assets/*' '$(PLUGINS_DIR)/$*' }"

# Build all WASM plugins
all-plugins: $(addprefix wasm/, $(PLUGINS))

# Clean all bundled plugin .wasm files, rebuild the app, then rebuild each plugin.
# Comment out individual plugin lines below to test a build without that plugin.
rebuild:
	@powershell -NoProfile -Command "Get-ChildItem '$(PLUGINS_DIR)\*.wasm' -ErrorAction SilentlyContinue | Remove-Item -Force"
	go build -o oof_rl.exe .
	$(MAKE) wasm/live
	$(MAKE) wasm/ranks
	$(MAKE) wasm/session
	$(MAKE) wasm/dashboard

# Test the SDK (pure Go, no WASM runtime)
test-sdk:
	go -C sdk test ./...

# Test a single plugin's logic: make test-plugin/live
test-plugin/%:
	go -C plugins/$* test ./...

# Test all plugins
test-plugins: test-sdk $(addprefix test-plugin/, $(PLUGINS))

# Test everything: host + SDK + plugins
test-all: test test-plugins

# --- Profiling (app must be running with dev_mode=true) ---
# NOTE: CPU profiling (profile target) crashes the app on Windows because the
# Go runtime profiler interrupts CGO threads used by WebView2. Use profile-trace
# for CPU-equivalent analysis, or profile-heap / profile-goroutine instead.

profile:
	go tool pprof -http=:9090 http://127.0.0.1:$(PORT)/debug/pprof/profile?seconds=30

profile-heap:
	go tool pprof -http=:9090 http://127.0.0.1:$(PORT)/debug/pprof/heap

profile-goroutine:
	go tool pprof -http=:9090 http://127.0.0.1:$(PORT)/debug/pprof/goroutine

# Execution trace: safe alternative to CPU profiling; does not interrupt CGO threads.
# Opens the trace viewer after collecting SECONDS seconds of data (default 15).
SECONDS ?= 15
profile-trace:
	curl -s -o /tmp/oof_trace.out "http://127.0.0.1:$(PORT)/debug/pprof/trace?seconds=$(SECONDS)"
	go tool trace /tmp/oof_trace.out
