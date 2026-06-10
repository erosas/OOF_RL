package core_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"OOF_RL/internal/config"
	"OOF_RL/internal/plugin"
)

// --- /api/plugins/<id>/view ---

func TestHandlePluginView_MissingID(t *testing.T) {
	mux, _ := newTestMux(t)
	w := get(mux, "/api/plugins/")
	if w.Code != http.StatusBadRequest {
		t.Errorf("missing id: got %d, want 400", w.Code)
	}
}

func TestHandlePluginView_NotFound(t *testing.T) {
	mux, _ := newTestMux(t)
	w := get(mux, "/api/plugins/nonexistent-xyz/view")
	if w.Code != http.StatusNotFound {
		t.Errorf("not found: got %d, want 404", w.Code)
	}
}

func TestHandlePluginViewResolvesByPluginID(t *testing.T) {
	cfg := config.Defaults()
	srv, _ := newTestServerWithConfig(t, cfg)
	srv.Use(&testPlugin{
		id:  "test_plugin",
		nav: plugin.NavTab{ID: "test-view", Label: "Test", Order: 1},
		assets: fstest.MapFS{
			"view.html": &fstest.MapFile{Data: []byte("<div>ok</div>")},
		},
	})

	mux := http.NewServeMux()
	srv.Register(mux)
	w := get(mux, "/api/plugins/test_plugin/view")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, body: %s", w.Code, w.Body.String())
	}
	if got := strings.TrimSpace(w.Body.String()); got != "<div>ok</div>" {
		t.Fatalf("body: got %q", got)
	}
}

func TestHandlePluginViewDisabledPluginReturnsNotFound(t *testing.T) {
	cfg := config.Defaults()
	cfg.DisabledPlugins = []string{"test_plugin"}
	srv, _ := newTestServerWithConfig(t, cfg)
	srv.Use(&testPlugin{
		id:  "test_plugin",
		nav: plugin.NavTab{ID: "test-view", Label: "Test", Order: 1},
		assets: fstest.MapFS{
			"view.html": &fstest.MapFile{Data: []byte("<div>ok</div>")},
		},
	})

	mux := http.NewServeMux()
	srv.Register(mux)
	w := get(mux, "/api/plugins/test_plugin/view")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", w.Code)
	}
}

func TestHandlePluginViewByViewIDReturnsNotFound(t *testing.T) {
	cfg := config.Defaults()
	srv, _ := newTestServerWithConfig(t, cfg)
	srv.Use(&testPlugin{
		id:  "test_plugin",
		nav: plugin.NavTab{ID: "test-view", Label: "Test", Order: 1},
		assets: fstest.MapFS{
			"view.html": &fstest.MapFile{Data: []byte("<div>ok</div>")},
		},
	})

	mux := http.NewServeMux()
	srv.Register(mux)
	w := get(mux, "/api/plugins/test-view/view")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", w.Code)
	}
}

// --- /api/plugins/<id>/data ---

func TestHandlePluginDataServesPublicFile(t *testing.T) {
	cfg := config.Defaults()
	cfg.DataDir = t.TempDir()
	srv, cfgPtr := newTestServerWithConfig(t, cfg)
	srv.Use(&testPlugin{id: "test_plugin", nav: plugin.NavTab{ID: "test-view", Label: "Test", Order: 1}})

	publicDir := filepath.Join(cfgPtr.DataDir, "plugin_data", "test_plugin", "public")
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(publicDir, "sample.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	mux := http.NewServeMux()
	srv.Register(mux)
	w := get(mux, "/api/plugins/test_plugin/data/sample.txt")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
	if strings.TrimSpace(w.Body.String()) != "hello" {
		t.Fatalf("body: got %q", w.Body.String())
	}
}

func TestHandlePluginDataRejectsTraversal(t *testing.T) {
	cfg := config.Defaults()
	cfg.DataDir = t.TempDir()
	srv, _ := newTestServerWithConfig(t, cfg)
	srv.Use(&testPlugin{id: "test_plugin", nav: plugin.NavTab{ID: "test-view", Label: "Test", Order: 1}})

	mux := http.NewServeMux()
	srv.Register(mux)
	w := get(mux, "/api/plugins/test_plugin/data/..\\secret.txt")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400", w.Code)
	}
}

func TestHandlePluginDataDisabledPluginReturnsNotFound(t *testing.T) {
	cfg := config.Defaults()
	cfg.DataDir = t.TempDir()
	cfg.DisabledPlugins = []string{"test_plugin"}
	srv, cfgPtr := newTestServerWithConfig(t, cfg)
	srv.Use(&testPlugin{id: "test_plugin", nav: plugin.NavTab{ID: "test-view", Label: "Test", Order: 1}})

	publicDir := filepath.Join(cfgPtr.DataDir, "plugin_data", "test_plugin", "public")
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(publicDir, "sample.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	mux := http.NewServeMux()
	srv.Register(mux)
	w := get(mux, "/api/plugins/test_plugin/data/sample.txt")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404", w.Code)
	}
}

func TestHandlePluginDataUnknownPluginReturnsNotFound(t *testing.T) {
	mux, _ := newTestMux(t)
	w := get(mux, "/api/plugins/nonexistent_xyz/data/file.txt")
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown plugin: got %d, want 404", w.Code)
	}
}

func TestHandlePluginDataMissingFileReturnsNotFound(t *testing.T) {
	cfg := config.Defaults()
	cfg.DataDir = t.TempDir()
	srv, _ := newTestServerWithConfig(t, cfg)
	srv.Use(&testPlugin{id: "test_plugin", nav: plugin.NavTab{ID: "test-view", Label: "Test", Order: 1}})

	mux := http.NewServeMux()
	srv.Register(mux)
	w := get(mux, "/api/plugins/test_plugin/data/missing.txt")
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing file: got %d, want 404", w.Code)
	}
}

// --- /api/data-dir ---

func TestHandleDataDir(t *testing.T) {
	mux, cfg := newTestMux(t)
	w := get(mux, "/api/data-dir")
	if w.Code != http.StatusOK {
		t.Fatalf("handleDataDir: got %d, want 200", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if resp["path"] != cfg.DataDir {
		t.Errorf("path: got %q, want %q", resp["path"], cfg.DataDir)
	}
}

// --- /ws ---

func TestHandleWS_NonUpgradable(t *testing.T) {
	mux, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code == http.StatusSwitchingProtocols {
		t.Error("did not expect 101 Switching Protocols for a plain HTTP request")
	}
}

// --- /api/db/open-folder ---

func TestHandleDBOpenFolder_MethodNotAllowed(t *testing.T) {
	// State-changing (launches explorer) — GET must be rejected so cross-site
	// GETs (img tags, prefetch) cannot trigger it.
	mux, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodGet, "/api/db/open-folder", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("handleDBOpenFolder GET: got %d, want 405", w.Code)
	}
}

// --- applyCoreSettings (via /api/settings) ---

func TestApplyCoreSettings(t *testing.T) {
	mux, cfg := newTestMux(t)
	w := postJSON(mux, "/api/settings", map[string]string{
		"storage.ball_hit_events": "true",
		"storage.raw_packets":     "true",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — body: %s", w.Code, w.Body.String())
	}
	if !cfg.Storage.BallHitEvents {
		t.Error("BallHitEvents: want true")
	}
	if !cfg.Storage.RawPackets {
		t.Error("RawPackets: want true")
	}
}