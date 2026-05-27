package core_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"OOF_RL/internal/config"
	"OOF_RL/internal/mmr"
	"OOF_RL/internal/plugin"
)

// --- /api/config ---

func TestGetConfig(t *testing.T) {
	mux, cfg := newTestMux(t)
	cfg.AppPort = 8080

	w := get(mux, "/api/config")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var got map[string]any
	json.Unmarshal(w.Body.Bytes(), &got)
	if _, ok := got["app_port"]; !ok {
		t.Error("expected app_port in response")
	}
	if _, ok := got["storage"]; !ok {
		t.Error("expected storage in response")
	}
}

func TestPostConfig(t *testing.T) {
	mux, _ := newTestMux(t)

	newCfg := config.Defaults()
	newCfg.AppPort = 9999

	w := postJSON(mux, "/api/config", newCfg)
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — body: %s", w.Code, w.Body.String())
	}
	var returned map[string]any
	json.Unmarshal(w.Body.Bytes(), &returned)
	if port, _ := returned["app_port"].(float64); int(port) != 9999 {
		t.Errorf("app_port: got %v, want 9999", returned["app_port"])
	}
}

func TestPostConfigSanitizesHostCoreDisabledPlugins(t *testing.T) {
	mux, _ := newTestMux(t)

	newCfg := config.Defaults()
	newCfg.DisabledPlugins = []string{"history", "test_disabled", "history"}

	w := postJSON(mux, "/api/config", newCfg)
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — body: %s", w.Code, w.Body.String())
	}
	var returned config.Config
	if err := json.Unmarshal(w.Body.Bytes(), &returned); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(returned.DisabledPlugins) != 1 || returned.DisabledPlugins[0] != "test_disabled" {
		t.Fatalf("disabled_plugins: got %v", returned.DisabledPlugins)
	}
}

func TestPostConfigBadJSON(t *testing.T) {
	mux, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestConfigMethodNotAllowed(t *testing.T) {
	mux, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/config", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want 405", w.Code)
	}
}

// --- /api/config/ini ---

func TestGetINIWhenNotFound(t *testing.T) {
	t.Setenv("USERPROFILE", "")
	mux, _ := newTestMux(t)
	w := get(mux, "/api/config/ini")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if errFlag, _ := resp["error"].(bool); !errFlag {
		t.Errorf("expected error:true in response, got: %v", resp)
	}
}

func TestPostINI(t *testing.T) {
	t.Setenv("USERPROFILE", "")
	mux, cfg := newTestMux(t)
	cfg.RLInstallPath = t.TempDir()

	w := postJSON(mux, "/api/config/ini", config.INISettings{PacketSendRate: 60, Port: 49123})
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — body: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status:ok, got %v", resp)
	}
}

func TestPostINIBadJSON(t *testing.T) {
	mux, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodPost, "/api/config/ini", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestINIMethodNotAllowed(t *testing.T) {
	mux, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/config/ini", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want 405", w.Code)
	}
}

// --- /api/nav ---

func TestGetNav(t *testing.T) {
	mux, _ := newTestMux(t)
	w := get(mux, "/api/nav")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var tabs []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &tabs); err != nil {
		t.Fatalf("parse: %v", err)
	}
}

func TestGetNavExcludesDisabledPluginByPluginID(t *testing.T) {
	cfg := config.Defaults()
	cfg.DisabledPlugins = []string{"test_disabled"}
	srv, _ := newTestServerWithConfig(t, cfg)
	srv.Use(&testPlugin{id: "test_enabled", nav: plugin.NavTab{ID: "enabled-view", Label: "Enabled", Order: 1}})
	srv.Use(&testPlugin{id: "test_disabled", nav: plugin.NavTab{ID: "disabled-view", Label: "Disabled", Order: 2}})

	mux := http.NewServeMux()
	srv.Register(mux)
	w := get(mux, "/api/nav")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var tabs []plugin.NavTab
	if err := json.Unmarshal(w.Body.Bytes(), &tabs); err != nil {
		t.Fatalf("parse: %v", err)
	}
	// Expect history (host-core) + test_enabled = 2 tabs.
	if len(tabs) != 2 {
		t.Fatalf("tab count: got %d, want 2", len(tabs))
	}
	ids := map[string]bool{}
	for _, tab := range tabs {
		ids[tab.ID] = true
	}
	if !ids["enabled-view"] {
		t.Fatalf("expected enabled-view tab, got %+v", tabs)
	}
	if ids["disabled-view"] {
		t.Fatalf("disabled-view should be excluded, got %+v", tabs)
	}
}

func TestGetNavAlwaysIncludesHistoryAsHostCore(t *testing.T) {
	// History is host-core: its tab comes from the host, not a plugin.
	// It must appear even with no plugins registered and even if "history"
	// appears in DisabledPlugins.
	cfg := config.Defaults()
	cfg.DisabledPlugins = []string{"history"}
	srv, _ := newTestServerWithConfig(t, cfg)

	mux := http.NewServeMux()
	srv.Register(mux)
	w := get(mux, "/api/nav")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var tabs []plugin.NavTab
	if err := json.Unmarshal(w.Body.Bytes(), &tabs); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(tabs) != 1 || tabs[0].ID != "history" {
		t.Fatalf("tabs: got %+v", tabs)
	}
}

// --- /api/settings/schema ---

func TestGetSettingsSchema(t *testing.T) {
	mux, _ := newTestMux(t)
	w := get(mux, "/api/settings/schema")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var blobs []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &blobs); err != nil {
		t.Fatalf("parse: %v", err)
	}
}

func TestGetSettingsSchemaMarksDisabledPluginButKeepsItListed(t *testing.T) {
	cfg := config.Defaults()
	cfg.DisabledPlugins = []string{"test_disabled"}
	srv, _ := newTestServerWithConfig(t, cfg)
	srv.Use(&testPlugin{id: "test_enabled", nav: plugin.NavTab{ID: "enabled-view", Label: "Enabled", Order: 1}})
	srv.Use(&testPlugin{id: "test_disabled", nav: plugin.NavTab{ID: "disabled-view", Label: "Disabled", Order: 2}})

	mux := http.NewServeMux()
	srv.Register(mux)
	w := get(mux, "/api/settings/schema")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var blobs []plugin.PluginSettingsBlob
	if err := json.Unmarshal(w.Body.Bytes(), &blobs); err != nil {
		t.Fatalf("parse: %v", err)
	}

	byID := make(map[string]plugin.PluginSettingsBlob, len(blobs))
	for _, b := range blobs {
		byID[b.PluginID] = b
	}

	disabled, ok := byID["test_disabled"]
	if !ok {
		t.Fatal("missing disabled plugin in settings schema")
	}
	if disabled.Enabled {
		t.Fatal("disabled plugin should be listed with enabled=false")
	}
	enabled, ok := byID["test_enabled"]
	if !ok {
		t.Fatal("missing enabled plugin in settings schema")
	}
	if !enabled.Enabled {
		t.Fatal("enabled plugin should be listed with enabled=true")
	}
}

func TestGetSettingsSchemaDoesNotIncludeHistory(t *testing.T) {
	// History is host-core with no configurable settings; it must not appear
	// in the plugin settings schema.
	mux, _ := newTestMux(t)
	w := get(mux, "/api/settings/schema")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var blobs []plugin.PluginSettingsBlob
	if err := json.Unmarshal(w.Body.Bytes(), &blobs); err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, b := range blobs {
		if b.PluginID == "history" {
			t.Fatal("history should not appear in plugin settings schema")
		}
	}
}

// --- /api/settings ---

func TestPostSettings(t *testing.T) {
	mux, cfg := newTestMux(t)

	w := postJSON(mux, "/api/settings", map[string]string{
		"ballchasing_api_key": "test-key",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d — body: %s", w.Code, w.Body.String())
	}
	if cfg.BallchasingAPIKey != "test-key" {
		t.Errorf("bc key: got %q, want test-key", cfg.BallchasingAPIKey)
	}

	newCfg := config.Defaults()
	newCfg.TrackerCacheTTLMinutes = 30
	w = postJSON(mux, "/api/config", newCfg)
	if w.Code != http.StatusOK {
		t.Fatalf("config status: got %d — body: %s", w.Code, w.Body.String())
	}
	if cfg.TrackerCacheTTLMinutes != 30 {
		t.Errorf("tracker TTL: got %d, want 30", cfg.TrackerCacheTTLMinutes)
	}
}

func TestPostSettingsMethodNotAllowed(t *testing.T) {
	mux, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want 405", w.Code)
	}
}

func TestPostSettingsBadJSON(t *testing.T) {
	mux, _ := newTestMux(t)
	req := httptest.NewRequest(http.MethodPost, "/api/settings", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

// --- /api/tracker/profile ---

func TestTrackerProfileNoProvider(t *testing.T) {
	// newTestMux wires mmrProvider=nil, so the handler must return 503.
	mux, _ := newTestMux(t)
	w := get(mux, "/api/tracker/profile?id=steam|76561198025501695")
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want 503", w.Code)
	}
}

func TestTrackerProfileMissingID(t *testing.T) {
	mux := newTestMuxWithMMR(t, &mockMMRProvider{name: "mock"})
	w := get(mux, "/api/tracker/profile")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestTrackerProfileInvalidIDFormat(t *testing.T) {
	mux := newTestMuxWithMMR(t, &mockMMRProvider{name: "mock"})
	w := get(mux, "/api/tracker/profile?id=nocolon")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestTrackerProfileMaskedSteamPlayer(t *testing.T) {
	mux := newTestMuxWithMMR(t, &mockMMRProvider{name: "mock"})
	w := get(mux, "/api/tracker/profile?id=steam|***")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestTrackerProfileMaskedNonSteamPlayer(t *testing.T) {
	// Non-steam with masked display name should also be rejected.
	mux := newTestMuxWithMMR(t, &mockMMRProvider{name: "mock"})
	w := get(mux, "/api/tracker/profile?id=epic|someepicid&name=***")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestTrackerProfileLookupSuccess(t *testing.T) {
	provider := &mockMMRProvider{
		name: "mock",
		ranks: []mmr.PlaylistRank{
			{PlaylistID: 10, PlaylistName: "Ranked Duel 1v1", MMR: 800, TierName: "Gold III"},
		},
	}
	mux := newTestMuxWithMMR(t, provider)
	w := get(mux, "/api/tracker/profile?id=steam|76561198025501695&name=TestPlayer")
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200 — body: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if resp["cached"].(bool) {
		t.Error("first call should not be cached")
	}
	if resp["source"] != "mock" {
		t.Errorf("source: got %v, want mock", resp["source"])
	}
	if ranks, _ := resp["ranks"].([]any); len(ranks) != 1 {
		t.Errorf("ranks count: got %d, want 1", len(ranks))
	}
	if provider.calls != 1 {
		t.Errorf("provider calls: got %d, want 1", provider.calls)
	}
}

func TestTrackerProfileLookupError(t *testing.T) {
	provider := &mockMMRProvider{err: errors.New("network error")}
	mux := newTestMuxWithMMR(t, provider)
	w := get(mux, "/api/tracker/profile?id=steam|76561198025501695")
	if w.Code != http.StatusBadGateway {
		t.Errorf("status: got %d, want 502", w.Code)
	}
}

func TestTrackerProfileCacheHit(t *testing.T) {
	provider := &mockMMRProvider{
		name:  "mock",
		ranks: []mmr.PlaylistRank{{PlaylistID: 10, MMR: 800}},
	}
	mux := newTestMuxWithMMR(t, provider)

	// First request — cold cache, goes to provider.
	w1 := get(mux, "/api/tracker/profile?id=steam|76561198025501695")
	if w1.Code != http.StatusOK {
		t.Fatalf("first request: got %d — body: %s", w1.Code, w1.Body.String())
	}
	if provider.calls != 1 {
		t.Fatalf("first request: provider calls: got %d, want 1", provider.calls)
	}

	// Second request — should hit cache, no additional provider call.
	w2 := get(mux, "/api/tracker/profile?id=steam|76561198025501695")
	if w2.Code != http.StatusOK {
		t.Fatalf("second request: got %d", w2.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !resp["cached"].(bool) {
		t.Error("second call should be cached")
	}
	if provider.calls != 1 {
		t.Errorf("provider calls: got %d, want 1 (second should use cache)", provider.calls)
	}
}
