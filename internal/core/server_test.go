package core_test

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"OOF_RL/internal/config"
	"OOF_RL/internal/core"
	"OOF_RL/internal/db"
	"OOF_RL/internal/hub"
	"OOF_RL/internal/mmr"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/plugin"
)

func newTestMux(t *testing.T) (*http.ServeMux, *config.Config) {
	t.Helper()
	tmpDir := t.TempDir()
	database, err := db.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	cfg := config.Defaults()
	cfg.RLInstallPath = ""
	cfg.DevMode = true
	h := hub.New()

	cfgPath := filepath.Join(tmpDir, "config.toml")
	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(bus.Stop)
	srv := core.NewServer(cfgPath, &cfg, database, h, http.NotFoundHandler(), func() {}, nil, bus)
	mux := http.NewServeMux()
	srv.Register(mux)
	return mux, &cfg
}

func newTestServer(t *testing.T) (*core.Server, oofevents.Bus) {
	t.Helper()
	tmpDir := t.TempDir()
	database, err := db.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	cfg := config.Defaults()
	h := hub.New()
	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(bus.Stop)

	srv := core.NewServer(filepath.Join(tmpDir, "config.toml"), &cfg, database, h, http.NotFoundHandler(), func() {}, nil, bus)
	return srv, bus
}

func newTestServerWithConfig(t *testing.T, cfg config.Config) (*core.Server, *config.Config) {
	t.Helper()
	tmpDir := t.TempDir()
	database, err := db.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	h := hub.New()
	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(bus.Stop)

	srv := core.NewServer(filepath.Join(tmpDir, "config.toml"), &cfg, database, h, http.NotFoundHandler(), func() {}, nil, bus)
	return srv, &cfg
}

type testPlugin struct {
	plugin.BasePlugin
	id            string
	nav           plugin.NavTab
	assets        fstest.MapFS
	routeOK       bool
	requires      []string
	initCount     int
	initErr       error
	declaredPaths []string // returned by RoutePaths; nil means BasePlugin default (nil = trusted)
}

func (p *testPlugin) ID() string       { return p.id }
func (p *testPlugin) Requires() []string { return p.requires }

func (p *testPlugin) Init(_ oofevents.PluginBus, _ plugin.Registry, _ *db.DB) error {
	p.initCount++
	return p.initErr
}

func (p *testPlugin) NavTab() plugin.NavTab { return p.nav }

func (p *testPlugin) RoutePaths() []string { return p.declaredPaths }

func (p *testPlugin) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/test/"+p.id, func(w http.ResponseWriter, _ *http.Request) {
		p.routeOK = true
		w.WriteHeader(http.StatusNoContent)
	})
}

func (p *testPlugin) Assets() fs.FS {
	if p.assets == nil {
		return nil
	}
	return p.assets
}

func (p *testPlugin) SettingsSchema() []plugin.Setting {
	return []plugin.Setting{{Key: p.id + ".enabled", Label: "Enabled", Type: plugin.SettingTypeCheckbox, Default: "true"}}
}

func (p *testPlugin) ApplySettings(map[string]string) error { return nil }

// mockMMRProvider is a test double for mmr.Provider.
type mockMMRProvider struct {
	name  string
	ranks []mmr.PlaylistRank
	err   error
	calls int
}

func (m *mockMMRProvider) Name() string                                           { return m.name }
func (m *mockMMRProvider) Supports(_ mmr.Platform) bool                           { return true }
func (m *mockMMRProvider) Lookup(_ mmr.PlayerIdentity) ([]mmr.PlaylistRank, error) {
	m.calls++
	return m.ranks, m.err
}

// newTestMuxWithMMR creates a test mux wired to an mmr.Provider.
func newTestMuxWithMMR(t *testing.T, provider mmr.Provider) *http.ServeMux {
	t.Helper()
	tmpDir := t.TempDir()
	database, err := db.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	cfg := config.Defaults()
	h := hub.New()
	cfgPath := filepath.Join(tmpDir, "config.toml")
	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(bus.Stop)
	srv := core.NewServer(cfgPath, &cfg, database, h, http.NotFoundHandler(), func() {}, provider, bus)
	mux := http.NewServeMux()
	srv.Register(mux)
	return mux
}

func get(mux *http.ServeMux, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func postJSON(mux *http.ServeMux, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestServerConfigReturnsConfig(t *testing.T) {
	srv, _ := newTestServer(t)
	if srv.Config() == nil {
		t.Fatal("Config() should return a non-nil config")
	}
}

func TestServerLoadPlugins(t *testing.T) {
	srv, _ := newTestServer(t)
	if err := srv.LoadPlugins(); err != nil {
		t.Fatalf("LoadPlugins: %v", err)
	}
}

// --- InitPlugins / ShutdownPlugins / Get ---

func TestInitPluginsAndShutdownPlugins(t *testing.T) {
	srv, _ := newTestServer(t)
	if err := srv.LoadPlugins(); err != nil {
		t.Fatalf("LoadPlugins: %v", err)
	}
	if err := srv.InitPlugins(); err != nil {
		t.Fatalf("InitPlugins: %v", err)
	}
	srv.ShutdownPlugins()
}

func TestInitPluginsSkipsDisabledPlugins(t *testing.T) {
	cfg := config.Defaults()
	cfg.DisabledPlugins = []string{"test_disabled"}
	srv, _ := newTestServerWithConfig(t, cfg)
	enabled := &testPlugin{id: "test_enabled", nav: plugin.NavTab{ID: "enabled-view", Label: "Enabled", Order: 1}}
	disabled := &testPlugin{id: "test_disabled", nav: plugin.NavTab{ID: "disabled-view", Label: "Disabled", Order: 2}}
	srv.Use(enabled)
	srv.Use(disabled)

	if err := srv.InitPlugins(); err != nil {
		t.Fatalf("InitPlugins: %v", err)
	}
	if enabled.initCount != 1 {
		t.Fatalf("enabled init count: got %d, want 1", enabled.initCount)
	}
	if disabled.initCount != 0 {
		t.Fatalf("disabled init count: got %d, want 0", disabled.initCount)
	}
}

func TestInitPluginsFailsWhenEnabledPluginRequiresDisabledPlugin(t *testing.T) {
	cfg := config.Defaults()
	cfg.DisabledPlugins = []string{"dep"}
	srv, _ := newTestServerWithConfig(t, cfg)
	dep := &testPlugin{id: "dep", nav: plugin.NavTab{ID: "dep-view", Label: "Dep", Order: 1}}
	consumer := &testPlugin{id: "consumer", nav: plugin.NavTab{ID: "consumer-view", Label: "Consumer", Order: 2}, requires: []string{"dep"}}
	srv.Use(dep)
	srv.Use(consumer)

	err := srv.InitPlugins()
	if err == nil {
		t.Fatal("expected InitPlugins to fail when enabled plugin requires disabled plugin")
	}
	if !strings.Contains(err.Error(), `requires disabled plugin "dep"`) {
		t.Fatalf("unexpected error: %v", err)
	}
	if consumer.initCount != 0 {
		t.Fatalf("consumer init count: got %d, want 0", consumer.initCount)
	}
	if dep.initCount != 0 {
		t.Fatalf("dep init count: got %d, want 0", dep.initCount)
	}
}

func TestServerGet(t *testing.T) {
	srv, _ := newTestServer(t)
	srv.Use(&testPlugin{id: "some-plugin", nav: plugin.NavTab{ID: "some-view", Label: "Some", Order: 1}})
	if _, ok := srv.Get("some-plugin"); !ok {
		t.Error("Get(some-plugin): want found")
	}
	if _, ok := srv.Get("nonexistent-plugin"); ok {
		t.Error("Get(nonexistent-plugin): want not found")
	}
}

// --- LoadWASMPlugins ---

func TestLoadWASMPlugins_MissingDir(t *testing.T) {
	srv, _ := newTestServer(t)
	if err := srv.LoadWASMPlugins(filepath.Join(t.TempDir(), "nonexistent")); err != nil {
		t.Fatalf("want nil for missing dir, got: %v", err)
	}
}

func TestLoadWASMPlugins_EmptyDir(t *testing.T) {
	srv, _ := newTestServer(t)
	if err := srv.LoadWASMPlugins(t.TempDir()); err != nil {
		t.Fatalf("LoadWASMPlugins empty dir: %v", err)
	}
	if len(srv.List()) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(srv.List()))
	}
}

func TestLoadWASMPlugins_InvalidWasm(t *testing.T) {
	srv, _ := newTestServer(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.wasm"), []byte("not wasm"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Bad WASM is logged and skipped; no error is returned to the caller.
	if err := srv.LoadWASMPlugins(dir); err != nil {
		t.Fatalf("want nil (bad wasm skipped), got: %v", err)
	}
	if len(srv.List()) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(srv.List()))
	}
}

// --- Register: disabled plugin routes and assets ---

func TestRegisterSkipsDisabledPluginRoutesAndAssets(t *testing.T) {
	cfg := config.Defaults()
	cfg.DisabledPlugins = []string{"test_disabled"}
	srv, _ := newTestServerWithConfig(t, cfg)
	enabled := &testPlugin{
		id:  "test_enabled",
		nav: plugin.NavTab{ID: "enabled-view", Label: "Enabled", Order: 1},
		assets: fstest.MapFS{
			"asset.txt": &fstest.MapFile{Data: []byte("enabled")},
		},
	}
	disabled := &testPlugin{
		id:  "test_disabled",
		nav: plugin.NavTab{ID: "disabled-view", Label: "Disabled", Order: 2},
		assets: fstest.MapFS{
			"asset.txt": &fstest.MapFile{Data: []byte("disabled")},
		},
	}
	srv.Use(enabled)
	srv.Use(disabled)

	mux := http.NewServeMux()
	srv.Register(mux)

	enabledRoute := get(mux, "/api/test/test_enabled")
	if enabledRoute.Code != http.StatusNoContent {
		t.Fatalf("enabled route status: got %d, want 204", enabledRoute.Code)
	}
	disabledRoute := get(mux, "/api/test/test_disabled")
	if disabledRoute.Code != http.StatusNotFound {
		t.Fatalf("disabled route status: got %d, want 404", disabledRoute.Code)
	}
	if !enabled.routeOK {
		t.Fatal("enabled route handler was not invoked")
	}
	if disabled.routeOK {
		t.Fatal("disabled route handler should not be invoked")
	}

	enabledAsset := get(mux, "/plugins/test_enabled/asset.txt")
	if enabledAsset.Code != http.StatusOK {
		t.Fatalf("enabled asset status: got %d, want 200", enabledAsset.Code)
	}
	disabledAsset := get(mux, "/plugins/test_disabled/asset.txt")
	if disabledAsset.Code != http.StatusNotFound {
		t.Fatalf("disabled asset status: got %d, want 404", disabledAsset.Code)
	}
}

// --- LoadPlugins duplicate ID ---

func TestLoadPluginsDuplicateIDReturnsError(t *testing.T) {
	const dupID = "test-dup-plugin"
	plugin.Register(dupID, func() plugin.Plugin {
		return &testPlugin{id: dupID, nav: plugin.NavTab{ID: "dup-view", Label: "Dup", Order: 1}}
	})
	t.Cleanup(func() { plugin.Unregister(dupID) })
	srv, _ := newTestServer(t)
	srv.Use(&testPlugin{id: dupID})
	err := srv.LoadPlugins()
	if err == nil {
		t.Fatal("LoadPlugins: want error for duplicate plugin ID, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate plugin ID") {
		t.Fatalf("LoadPlugins: unexpected error: %v", err)
	}
}

// --- Register route conflict detection ---

func TestRegisterSkipsPluginWithConflictingCoreRoute(t *testing.T) {
	srv, _ := newTestServerWithConfig(t, config.Defaults())
	// Plugin declares a path that conflicts with the core /api/config route.
	conflicting := &testPlugin{
		id:            "conflict_plugin",
		nav:           plugin.NavTab{ID: "conflict-view", Label: "Conflict", Order: 99},
		declaredPaths: []string{"/api/config"},
	}
	srv.Use(conflicting)

	mux := http.NewServeMux()
	srv.Register(mux)

	// Core /api/config must still be reachable.
	w := get(mux, "/api/config")
	if w.Code != http.StatusOK {
		t.Fatalf("/api/config: got %d, want 200 after route-conflict skip", w.Code)
	}
	// The conflicting plugin's own route must NOT have been registered.
	w = get(mux, "/api/test/conflict_plugin")
	if w.Code != http.StatusNotFound {
		t.Fatalf("conflict plugin own route: got %d, want 404", w.Code)
	}
}

func TestRegisterSkipsPluginWithConflictingPluginRoute(t *testing.T) {
	srv, _ := newTestServerWithConfig(t, config.Defaults())
	first := &testPlugin{
		id:            "plugin_a",
		nav:           plugin.NavTab{ID: "a-view", Label: "A", Order: 1},
		declaredPaths: []string{"/api/shared-route"},
	}
	second := &testPlugin{
		id:            "plugin_b",
		nav:           plugin.NavTab{ID: "b-view", Label: "B", Order: 2},
		declaredPaths: []string{"/api/shared-route"},
	}
	srv.Use(first)
	srv.Use(second)

	mux := http.NewServeMux()
	srv.Register(mux)

	// First plugin's own route (/api/test/plugin_a) should be registered.
	w := get(mux, "/api/test/plugin_a")
	if w.Code != http.StatusNoContent {
		t.Fatalf("plugin_a own route: got %d, want 204", w.Code)
	}
	// Second plugin's own route (/api/test/plugin_b) must NOT be registered.
	w = get(mux, "/api/test/plugin_b")
	if w.Code != http.StatusNotFound {
		t.Fatalf("plugin_b own route after conflict: got %d, want 404", w.Code)
	}
}