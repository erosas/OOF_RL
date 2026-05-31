package core

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"OOF_RL/internal/config"
	"OOF_RL/internal/events"
	"OOF_RL/internal/histstore"
	"OOF_RL/internal/httputil"
	"OOF_RL/internal/mmr"
	"OOF_RL/internal/plugin"
)

// -- App config --

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		httputil.WriteJSON(w, s.cfg)
	case http.MethodPost:
		if err := json.NewDecoder(r.Body).Decode(s.cfg); err != nil {
			httputil.JSONError(w, 400, err.Error())
			return
		}
		s.cfg.DisabledPlugins = sanitizeDisabledPlugins(s.cfg.DisabledPlugins)
		if err := config.Save(s.cfgPath, *s.cfg); err != nil {
			httputil.JSONError(w, 500, err.Error())
			return
		}
		s.reconnect()
		httputil.WriteJSON(w, s.cfg)
	default:
		httputil.JSONError(w, 405, "method not allowed")
	}
}

func (s *Server) handleINI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ini, err := config.ReadINI(s.cfg.RLInstallPath)
		if err != nil {
			note := "DefaultStatsAPI.ini not found — it will be created when you save."
			if s.cfg.RLInstallPath == "" {
				note = "RL install path not set. Configure it above and save first."
			}
			httputil.WriteJSON(w, map[string]any{
				"PacketSendRate": 0.0,
				"Port":           49123,
				"note":           note,
				"error":          true,
			})
			return
		}
		httputil.WriteJSON(w, ini)
	case http.MethodPost:
		var ini config.INISettings
		if err := json.NewDecoder(r.Body).Decode(&ini); err != nil {
			httputil.JSONError(w, 400, err.Error())
			return
		}
		if err := config.WriteINI(s.cfg.RLInstallPath, ini); err != nil {
			httputil.JSONError(w, 500, err.Error())
			return
		}
		s.reconnect()
		httputil.WriteJSON(w, map[string]string{"status": "ok", "note": "INI saved. Restart Rocket League for changes to take effect."})
	default:
		httputil.JSONError(w, 405, "method not allowed")
	}
}

// -- Plugin meta endpoints --

func (s *Server) handleNav(w http.ResponseWriter, r *http.Request) {
	// History is host-core: its tab is always present, never disabled.
	tabs := []plugin.NavTab{{ID: "history", Label: "History", Order: 20}}
	for _, p := range s.activePlugins() {
		if tab := p.NavTab(); tab.ID != "" {
			tabs = append(tabs, tab)
		}
	}
	sort.Slice(tabs, func(i, j int) bool { return tabs[i].Order < tabs[j].Order })
	httputil.WriteJSON(w, tabs)
}

// handlePluginView serves GET /api/plugins/{pluginID}/view — returns the plugin's
// view.html fragment so the frontend can inject it into the page dynamically.
func (s *Server) handlePluginView(w http.ResponseWriter, r *http.Request) {
	if pluginID, relPath, ok := parsePluginDataPath(r.URL.Path); ok {
		s.handlePluginData(w, r, pluginID, relPath)
		return
	}

	pluginID := strings.TrimPrefix(r.URL.Path, "/api/plugins/")
	pluginID = strings.TrimSuffix(pluginID, "/view")
	pluginID = strings.Trim(pluginID, "/")
	if pluginID == "" {
		httputil.JSONError(w, 400, "missing plugin id")
		return
	}

	// History is host-core: serve its view from embedded histstore assets.
	if pluginID == "history" {
		b, err := histstore.Assets.ReadFile("assets/view.html")
		if err != nil {
			httputil.JSONError(w, 404, "view not found")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
		return
	}

	target := s.findPluginTarget(pluginID)
	if target == nil {
		httputil.JSONError(w, 404, "plugin not found")
		return
	}
	assets := target.Assets()
	if assets == nil {
		httputil.JSONError(w, 404, "plugin has no assets")
		return
	}
	b, err := fs.ReadFile(assets, "view.html")
	if err != nil {
		httputil.JSONError(w, 404, "view.html not found")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(b)
}

func parsePluginDataPath(path string) (pluginID string, relPath string, ok bool) {
	const prefix = "/api/plugins/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	tail := strings.TrimPrefix(path, prefix)
	parts := strings.SplitN(tail, "/data/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	pluginID = strings.Trim(parts[0], "/")
	relPath = strings.Trim(parts[1], "/")
	if pluginID == "" || relPath == "" {
		return "", "", false
	}
	return pluginID, relPath, true
}

func (s *Server) handlePluginData(w http.ResponseWriter, r *http.Request, pluginID, relPath string) {
	if s.findPluginTarget(pluginID) == nil {
		httputil.JSONError(w, 404, "plugin not found")
		return
	}
	relPath = strings.ReplaceAll(relPath, "\\", "/")
	for _, seg := range strings.Split(relPath, "/") {
		if seg == ".." {
			httputil.JSONError(w, 400, "invalid path")
			return
		}
	}
	base := filepath.Join(s.cfg.DataDir, "plugin_data", pluginID, "public")
	baseClean := filepath.Clean(base)
	full := filepath.Join(baseClean, filepath.FromSlash(relPath))
	fullClean := filepath.Clean(full)
	if fullClean != baseClean && !strings.HasPrefix(fullClean, baseClean+string(os.PathSeparator)) {
		httputil.JSONError(w, 400, "invalid path")
		return
	}
	info, err := os.Stat(fullClean)
	if err != nil || info.IsDir() {
		httputil.JSONError(w, 404, "file not found")
		return
	}
	http.ServeFile(w, r, fullClean)
}

// -- Settings --

func (s *Server) coreSettingsSchema() []plugin.Setting {
	return []plugin.Setting{
		{
			Key:         "storage.ball_hit_events",
			Label:       "Ball hit events",
			Type:        plugin.SettingTypeCheckbox,
			Default:     "false",
			Description: "Store every ball touch. Can generate large amounts of data.",
		},
		{
			Key:         "storage.raw_packets",
			Label:       "Capture raw packets",
			Type:        plugin.SettingTypeCheckbox,
			Default:     "false",
			Description: "Save raw UDP packets to disk under captures/ in the data directory.",
		},
	}
}

func (s *Server) applyCoreSettings(values map[string]string) {
	if v, ok := values["storage.ball_hit_events"]; ok {
		s.cfg.Storage.BallHitEvents = v == "true"
	}
	if v, ok := values["storage.raw_packets"]; ok {
		s.cfg.Storage.RawPackets = v == "true"
	}
}

func (s *Server) handleSettingsSchema(w http.ResponseWriter, r *http.Request) {
	disabled := s.disabled
	blobs := make([]plugin.PluginSettingsBlob, 0, len(s.plugins)+1)
	for _, p := range s.plugins {
		tab := p.NavTab()
		title := tab.Label
		if title == "" {
			title = p.ID()
		}
		blobs = append(blobs, plugin.PluginSettingsBlob{
			PluginID: p.ID(),
			ViewID:   tab.ID,
			Title:    title,
			Enabled:  !isPluginDisabledInSet(p.ID(), disabled),
			Requires: p.Requires(),
			Settings: p.SettingsSchema(),
		})
	}
	blobs = append(blobs, plugin.PluginSettingsBlob{
		PluginID: "core",
		Title:    "Advanced",
		Enabled:  true,
		Settings: s.coreSettingsSchema(),
	})
	httputil.WriteJSON(w, blobs)
}

func (s *Server) handleDBOpenFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.JSONError(w, 405, "method not allowed")
		return
	}
	dir := s.cfg.DataDir
	if err := exec.Command("explorer.exe", dir).Start(); err != nil {
		httputil.JSONError(w, 500, err.Error())
		return
	}
	httputil.WriteJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleOpenExternal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.JSONError(w, 405, "method not allowed")
		return
	}
	if !requestComesFromApp(r) {
		httputil.JSONError(w, 403, "cross-origin requests are not allowed")
		return
	}
	if ct := strings.ToLower(r.Header.Get("Content-Type")); !strings.HasPrefix(ct, "application/json") {
		httputil.JSONError(w, 415, "content type must be application/json")
		return
	}
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httputil.JSONError(w, 400, err.Error())
		return
	}
	target, err := validateExternalURL(body.URL)
	if err != nil {
		httputil.JSONError(w, 400, err.Error())
		return
	}
	if err := exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", target).Start(); err != nil {
		httputil.JSONError(w, 500, err.Error())
		return
	}
	httputil.WriteJSON(w, map[string]string{"status": "ok"})
}

func requestComesFromApp(r *http.Request) bool {
	if origin := r.Header.Get("Origin"); origin != "" {
		return headerURLMatchesHost(origin, r.Host)
	}
	if referer := r.Header.Get("Referer"); referer != "" {
		return headerURLMatchesHost(referer, r.Host)
	}
	return true
}

func headerURLMatchesHost(raw string, host string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return strings.EqualFold(u.Host, host)
}

func validateExternalURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("url is required")
	}
	if len(raw) > 2048 {
		return "", fmt.Errorf("url is too long")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("only http and https URLs can be opened")
	}
	if u.Host == "" {
		return "", fmt.Errorf("url host is required")
	}
	return u.String(), nil
}

// handleSettings dispatches a flat key→value map to every plugin's ApplySettings,
// then persists config to disk.
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.JSONError(w, 405, "method not allowed")
		return
	}
	var values map[string]string
	if err := json.NewDecoder(r.Body).Decode(&values); err != nil {
		httputil.JSONError(w, 400, err.Error())
		return
	}
	s.applyCoreSettings(values)
	for k, v := range values {
		s.cfg.Set(k, v)
	}
	for _, p := range s.plugins {
		if err := p.ApplySettings(values); err != nil {
			httputil.JSONError(w, 500, fmt.Sprintf("plugin %s: %v", p.ID(), err))
			log.Printf("[settings] plugin %s ApplySettings error: %v", p.ID(), err)
			return
		}
	}
	if err := config.Save(s.cfgPath, *s.cfg); err != nil {
		httputil.JSONError(w, 500, err.Error())
		return
	}
	s.reconnect()
	httputil.WriteJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleDataDir(w http.ResponseWriter, r *http.Request) {
	httputil.WriteJSON(w, map[string]string{"path": s.cfg.DataDir})
}

// DispatchEvent translates a raw RL envelope into typed OOF events and publishes
// them on the bus. All plugin event handling is via bus subscriptions.
func (s *Server) DispatchEvent(env events.Envelope) {
	s.translator.Translate(env)
}

// -- Tracker profile --

func (s *Server) handleTrackerProfile(w http.ResponseWriter, r *http.Request) {
	if s.mmrProvider == nil {
		httputil.JSONError(w, 503, "tracker service unavailable")
		return
	}

	id := r.URL.Query().Get("id")
	playerName := r.URL.Query().Get("name")
	if id == "" {
		httputil.JSONError(w, 400, "missing id")
		return
	}

	sep := strings.IndexAny(id, "|:_")
	if sep < 1 {
		httputil.JSONError(w, 400, "invalid id format, expected platform|id")
		return
	}
	rawPlatform := strings.ToLower(id[:sep])
	primaryID := id[sep+1:]
	if end := strings.IndexAny(primaryID, "|:_"); end >= 0 {
		primaryID = primaryID[:end]
	}

	if mmr.IsAllAsterisks(primaryID) || (rawPlatform != "steam" && mmr.IsAllAsterisks(playerName)) {
		httputil.JSONError(w, 400, "masked player name")
		return
	}

	platform := mmr.NormalizePlatform(rawPlatform)
	displayName := playerName
	if displayName == "" {
		displayName = primaryID
	}
	identity := mmr.PlayerIdentity{
		PrimaryID:   primaryID,
		DisplayName: displayName,
		Platform:    platform,
	}

	ttl := time.Duration(s.cfg.TrackerCacheTTLMinutes) * time.Minute
	if ttl < 2*time.Minute {
		ttl = 2 * time.Minute
	}
	dataJSON, fetchedAt, found, err := s.db.GetTrackerCache(id)
	if err == nil && found && time.Since(fetchedAt) < ttl {
		var ranks []mmr.PlaylistRank
		if json.Unmarshal([]byte(dataJSON), &ranks) == nil {
			log.Printf("[tracker] %s — cache hit (age %s)", id, time.Since(fetchedAt).Round(time.Second))
			trackerWriteRankResponse(w, true, fetchedAt, ranks, "")
			return
		}
	}

	ranks, err := s.mmrProvider.Lookup(identity)
	if err != nil {
		log.Printf("[tracker] lookup failed for %s: %v", id, err)
		httputil.JSONError(w, 502, err.Error())
		return
	}

	now := time.Now().UTC()
	if b, merr := json.Marshal(ranks); merr == nil {
		_ = s.db.UpsertTrackerCache(id, string(b))
	}
	trackerWriteRankResponse(w, false, now, ranks, s.mmrProvider.Name())
}

func trackerWriteRankResponse(w http.ResponseWriter, cached bool, fetchedAt time.Time, ranks []mmr.PlaylistRank, source string) {
	httputil.WriteJSON(w, map[string]any{
		"cached":     cached,
		"fetched_at": fetchedAt.UTC().Format(time.RFC3339),
		"source":     source,
		"ranks":      ranks,
	})
}
