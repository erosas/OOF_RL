.PHONY: build run test cover icon profile profile-heap profile-goroutine \
        all-plugins test-plugins test-sdk $(addprefix wasm/, $(PLUGINS)) $(addprefix test-plugin/, $(PLUGINS))

PORT    ?= 8080
PLUGINS := live ballchasing ranks session history dashboard debugassistant

# WASM plugins are installed into the same data directory the app reads at runtime.
# LOCALAPPDATA is inherited from the Windows environment (e.g. C:\Users\you\AppData\Local).
PLUGINS_DIR := $(LOCALAPPDATA)/OOF_RL/plugins

build:
	go build -o oof_rl.exe .

icon:
	go run ./tools/genicon

	$(shell go env GOPATH)/bin/rsrc -ico icon.ico -o rsrc.syso

run: build
	./oof_rl.exe

# Host tests only
test:
	go test ./...

cover:
	go test -coverprofile=coverage.out ./...
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

# Test the SDK (pure Go, no WASM runtime)
test-sdk:
	go -C plugins/sdk test ./...

# Test a single plugin's logic: make test-plugin/live
test-plugin/%:
	go -C plugins/$* test ./...

# Test all plugins
test-plugins: test-sdk $(addprefix test-plugin/, $(PLUGINS))

# Test everything: host + SDK + plugins
test-all: test test-plugins

# --- Profiling (app must be running) ---

profile:
	go tool pprof -http=:9090 http://localhost:$(PORT)/debug/pprof/profile?seconds=30

profile-heap:
	go tool pprof -http=:9090 http://localhost:$(PORT)/debug/pprof/heap

profile-goroutine:
	go tool pprof -http=:9090 http://localhost:$(PORT)/debug/pprof/goroutine