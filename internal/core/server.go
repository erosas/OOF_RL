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

	"github.com/gorilla/websocket"

	"OOF_RL/internal/config"
	"OOF_RL/internal/db"
	"OOF_RL/internal/events"
	"OOF_RL/internal/histstore"
	"OOF_RL/internal/httputil"
	"OOF_RL/internal/hub"
	"OOF_RL/internal/mmr"
	"OOF_RL/internal/momentum"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/plugin"
	"OOF_RL/internal/rlevents"
	"OOF_RL/internal/wasmhost"
)

var upgrader = websocket.Upgrader{
	// Accept only connections from localhost. An absent Origin header (e.g.
	// from the embedded WebView2) is allowed; any explicit non-localhost origin
	// is rejected to prevent cross-site WebSocket hijacking.
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		h := u.Hostname()
		return h == "localhost" || h == "127.0.0.1"
	},
}

type Server struct {
	cfg          *config.Config
	cfgPath      string
	db           *db.DB
	hub          *hub.Hub
	fs           http.Handler
	reconnect    func()
	mmrProvider  mmr.Provider
	plugins      []plugin.Plugin
	bus          oofevents.Bus
	translator   *rlevents.Translator
	momentum     *momentum.Service
	momentumW    *momentum.Wiring
	histStore    *histstore.Store
	histRecorder *histstore.Recorder
}

func isHostCorePluginID(pluginID string) bool {
	return pluginID == "history"
}

func NewServer(cfgPath string, cfg *config.Config, database *db.DB, h *hub.Hub, static http.Handler, reconnect func(), mmrProvider mmr.Provider, bus oofevents.Bus) *Server {
	rlBus := bus.ForPlugin("") // RL translator convention: empty plugin ID
	momentumService := momentum.NewService(momentum.DefaultConfig())
	s := &Server{
		cfgPath:     cfgPath,
		cfg:         cfg,
		db:          database,
		hub:         h,
		fs:          static,
		reconnect:   reconnect,
		mmrProvider: mmrProvider,
		bus:         bus,
		translator:  rlevents.New(rlBus),
		momentum:    momentumService,
		momentumW:   momentum.NewWiring(bus.ForPlugin("momentum"), momentumService),
	}
	if database != nil {
		if err := histstore.Migrate(database); err != nil {
			log.Printf("[core] histstore migrate: %v", err)
		}
		s.histStore = histstore.NewStore(database)
		s.histRecorder = histstore.NewRecorder(s.histStore, cfg)
	}
	return s
}

// Momentum returns the read-only app-owned Momentum snapshot provider.
func (s *Server) Momentum() momentum.SnapshotProvider {
	return s.momentum
}

func (s *Server) Config() *config.Config { return s.cfg }

// InitPlugins sorts plugins by their declared dependencies, then calls Init on
// each in dependency order. Must be called before the RL client delivers events.
func (s *Server) InitPlugins() error {
	if s.histRecorder != nil {
		s.histRecorder.Subscribe(s.bus.ForPlugin("history"))
	}
	active := s.activePlugins()
	if err := s.validateActivePluginDependencies(active); err != nil {
		return fmt.Errorf("plugin dependency: %w", err)
	}
	sorted, err := topoSort(active)
	if err != nil {
		return fmt.Errorf("plugin dependency: %w", err)
	}
	for _, p := range sorted {
		if err := p.Init(s.bus.ForPlugin(p.ID()), s, s.db); err != nil {
			return fmt.Errorf("plugin %s Init: %w", p.ID(), err)
		}
	}
	return nil
}

func (s *Server) validateActivePluginDependencies(active []plugin.Plugin) error {
	disabled := s.disabledPluginSet()
	activeSet := make(map[string]struct{}, len(active))
	for _, p := range active {
		activeSet[p.ID()] = struct{}{}
	}
	allSet := make(map[string]struct{}, len(s.plugins))
	for _, p := range s.plugins {
		allSet[p.ID()] = struct{}{}
	}
	for _, p := range active {
		for _, req := range p.Requires() {
			if _, ok := activeSet[req]; ok {
				continue
			}
			if _, known := allSet[req]; known && isPluginDisabledInSet(req, disabled) {
				return fmt.Errorf("plugin %q requires disabled plugin %q", p.ID(), req)
			}
		}
	}
	return nil
}

// topoSort returns plugins in dependency order (required plugins first).
// Returns an error if a required plugin is not loaded or a cycle is detected.
func topoSort(plugins []plugin.Plugin) ([]plugin.Plugin, error) {
	byID := make(map[string]plugin.Plugin, len(plugins))
	for _, p := range plugins {
		byID[p.ID()] = p
	}

	// dependants[X] = list of plugin IDs that require X
	dependants := make(map[string][]string)
	inDegree := make(map[string]int, len(plugins))
	for _, p := range plugins {
		if _, seen := inDegree[p.ID()]; !seen {
			inDegree[p.ID()] = 0
		}
		for _, req := range p.Requires() {
			if _, ok := byID[req]; !ok {
				return nil, fmt.Errorf("plugin %q requires unknown plugin %q", p.ID(), req)
			}
			inDegree[p.ID()]++
			dependants[req] = append(dependants[req], p.ID())
		}
	}

	queue := make([]plugin.Plugin, 0, len(plugins))
	for _, p := range plugins {
		if inDegree[p.ID()] == 0 {
			queue = append(queue, p)
		}
	}

	sorted := make([]plugin.Plugin, 0, len(plugins))
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		sorted = append(sorted, p)
		for _, depID := range dependants[p.ID()] {
			inDegree[depID]--
			if inDegree[depID] == 0 {
				queue = append(queue, byID[depID])
			}
		}
	}

	if len(sorted) != len(plugins) {
		return nil, fmt.Errorf("circular dependency detected")
	}
	return sorted, nil
}

// ShutdownPlugins calls Shutdown on every registered plugin in reverse order,
// then stops the history recorder.
func (s *Server) ShutdownPlugins() {
	for i := len(s.plugins) - 1; i >= 0; i-- {
		p := s.plugins[i]
		if err := p.Shutdown(); err != nil {
			log.Printf("[core] plugin %s Shutdown: %v", p.ID(), err)
		}
	}
	if s.histRecorder != nil {
		s.histRecorder.Stop()
	}
}

// ShutdownRuntime stops app-owned runtime services that are not plugins.
func (s *Server) ShutdownRuntime() {
	if s.momentumW != nil {
		s.momentumW.Stop()
	}
}

// Get implements plugin.Registry.
func (s *Server) Get(id string) (plugin.Plugin, bool) {
	for _, p := range s.plugins {
		if p.ID() == id {
			return p, true
		}
	}
	return nil, false
}

// List implements plugin.Registry.
func (s *Server) List() []plugin.Plugin {
	out := make([]plugin.Plugin, len(s.plugins))
	copy(out, s.plugins)
	return out
}

// Use registers a plugin. Call before Register.
func (s *Server) Use(p plugin.Plugin) {
	s.plugins = append(s.plugins, p)
}

func (s *Server) disabledPluginSet() map[string]struct{} {
	disabled := make(map[string]struct{}, len(s.cfg.DisabledPlugins))
	for _, id := range s.cfg.DisabledPlugins {
		disabled[id] = struct{}{}
	}
	return disabled
}

func (s *Server) isPluginDisabled(pluginID string) bool {
	return isPluginDisabledInSet(pluginID, s.disabledPluginSet())
}

func isPluginDisabledInSet(pluginID string, disabled map[string]struct{}) bool {
	if isHostCorePluginID(pluginID) {
		return false
	}
	_, ok := disabled[pluginID]
	return ok
}

func sanitizeDisabledPlugins(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || isHostCorePluginID(id) {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (s *Server) activePlugins() []plugin.Plugin {
	disabled := s.disabledPluginSet()
	active := make([]plugin.Plugin, 0, len(s.plugins))
	for _, p := range s.plugins {
		if isPluginDisabledInSet(p.ID(), disabled) {
			continue
		}
		active = append(active, p)
	}
	return active
}

func (s *Server) findPluginTarget(pluginID string) plugin.Plugin {
	for _, p := range s.activePlugins() {
		if p.ID() == pluginID {
			return p
		}
	}
	return nil
}

func (s *Server) LoadPlugins() error {
	for id, factory := range plugin.Factories() {
		p := factory()
		if p.ID() != id {
			log.Printf("[core] plugin %q: ID mismatch (factory key %q vs plugin.ID() %q)", id, id, p.ID())
		}
		if _, exists := s.Get(p.ID()); exists {
			return fmt.Errorf("duplicate plugin ID %q", p.ID())
		}
		s.Use(p)
	}
	return nil
}

// LoadWASMPlugins scans dir for *.wasm files and registers each as a plugin.
// Missing or empty dir is silently ignored.
func (s *Server) LoadWASMPlugins(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("wasm plugins dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".wasm") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		p, err := wasmhost.Load(path, s.db, s.hub, s.cfg)
		if err != nil {
			log.Printf("[core] wasm load %s: %v", e.Name(), err)
			continue
		}
		if _, exists := s.Get(p.ID()); exists {
			log.Printf("[core] wasm load %s: plugin ID %q already registered, skipping", e.Name(), p.ID())
			p.Shutdown()
			continue
		}
		s.Use(p)
		log.Printf("[core] loaded wasm plugin %q from %s", p.ID(), e.Name())
	}
	return nil
}

// Register wires all routes onto mux: core endpoints first, then each plugin,
// then the static file fallback.
func (s *Server) Register(mux *http.ServeMux) {
	// Seed the registered-routes set with every core pattern so plugins can be
	// checked against it before their Routes() call touches the mux.
	registered := map[string]string{
		"/ws":                  "core",
		"/api/config":          "core",
		"/api/config/ini":      "core",
		"/api/nav":             "core",
		"/api/plugins/":        "core",
		"/api/players":         "core",
		"/api/matches":         "core",
		"/api/matches/":        "core",
		"/api/settings/schema": "core",
		"/api/settings":        "core",
		"/api/tracker/profile": "core",
		"/api/db/open-folder":  "core",
		"/api/data-dir":        "core",
	}

	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/config/ini", s.handleINI)
	mux.HandleFunc("/api/nav", s.handleNav)
	mux.HandleFunc("/api/plugins/", s.handlePluginView)
	if s.histStore != nil {
		mux.HandleFunc("/api/players", s.histStore.HandlePlayers)
		mux.HandleFunc("/api/matches", s.histStore.HandleMatches)
		mux.HandleFunc("/api/matches/", s.histStore.HandleMatchDetail)
	}
	mux.HandleFunc("/api/settings/schema", s.handleSettingsSchema)
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/api/tracker/profile", s.handleTrackerProfile)
	mux.HandleFunc("/api/db/open-folder", s.handleDBOpenFolder)
	mux.HandleFunc("/api/data-dir", s.handleDataDir)

	for _, p := range s.activePlugins() {
		// Plugins that declare their routes (WASM plugins) are checked for
		// conflicts before any mux registration occurs. Native plugins return
		// nil from RoutePaths() and are trusted without a pre-check.
		conflict := false
		for _, path := range p.RoutePaths() {
			if owner, ok := registered[path]; ok {
				log.Printf("[core] plugin %q: route %q conflicts with %q — plugin routes not registered", p.ID(), path, owner)
				conflict = true
				break
			}
		}
		if conflict {
			continue
		}
		for _, path := range p.RoutePaths() {
			registered[path] = p.ID()
		}

		p.Routes(mux)
		if assets := p.Assets(); assets != nil {
			prefix := "/plugins/" + p.ID() + "/"
			mux.Handle(prefix, http.StripPrefix(prefix, http.FileServer(http.FS(assets))))
		}
	}
	mux.Handle("/", s.fs)
}

// -- WebSocket --

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	s.hub.Register(conn)
	defer func() {
		s.hub.Unregister(conn)
		conn.Close()
	}()
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

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
	tabs := make([]plugin.NavTab, 0, len(s.plugins))
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
	disabled := s.disabledPluginSet()
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

// handleSettings dispatches a flat key→value map to every plugin's ApplySettings,
// then persists config to disk. Used by the dynamic Settings page (Phase 9).
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
// them on the bus. All plugin event handling is now via bus subscriptions.
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
