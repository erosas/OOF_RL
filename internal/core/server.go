package core

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
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
	"OOF_RL/internal/momentum/timeline"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/plugin"
	"OOF_RL/internal/rlevents"
	"OOF_RL/internal/wasmhost"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
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
	timeline     *timeline.Collector
	timelineW    *timeline.Wiring
	histStore    *histstore.Store
	histRecorder *histstore.Recorder
}

func NewServer(cfgPath string, cfg *config.Config, database *db.DB, h *hub.Hub, static http.Handler, reconnect func(), mmrProvider mmr.Provider, bus oofevents.Bus) *Server {
	rlBus := bus.ForPlugin("") // RL translator convention: empty plugin ID
	momentumService := momentum.NewService(momentum.DefaultConfig())
	timelineCollector := timeline.NewCollector(momentumService, timeline.Config{})
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
		timeline:    timelineCollector,
		timelineW:   timeline.NewWiring(bus.ForPlugin("momentum-timeline"), momentumService, timelineCollector),
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

// Timeline returns the read-only app-owned Timeline snapshot provider.
func (s *Server) Timeline() timeline.SnapshotProvider {
	return s.timeline
}

func (s *Server) Config() *config.Config { return s.cfg }

// InitPlugins sorts plugins by their declared dependencies, then calls Init on
// each in dependency order. Must be called before the RL client delivers events.
func (s *Server) InitPlugins() error {
	if s.histRecorder != nil {
		s.histRecorder.Subscribe(s.bus.ForPlugin("history"))
	}
	sorted, err := topoSort(s.plugins)
	if err != nil {
		return fmt.Errorf("plugin dependency: %w", err)
	}
	s.plugins = sorted
	for _, p := range s.plugins {
		if err := p.Init(s.bus.ForPlugin(p.ID()), s, s.db); err != nil {
			return fmt.Errorf("plugin %s Init: %w", p.ID(), err)
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
	if s.timelineW != nil {
		s.timelineW.Stop()
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

func (s *Server) LoadPlugins() error {
	for id, factory := range plugin.Factories() {
		p := factory()
		if p.ID() != id {
			log.Printf("[core] warning: plugin ID mismatch for %q: got %q", id, p.ID())
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
		s.Use(p)
		log.Printf("[core] loaded wasm plugin %q from %s", p.ID(), e.Name())
	}
	return nil
}

// Register wires all routes onto mux: core endpoints first, then each plugin,
// then the static file fallback.
func (s *Server) Register(mux *http.ServeMux) {
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
	mux.HandleFunc("/internal/momentum-timeline-snapshot", s.handleMomentumTimelineSnapshot)
	for _, p := range s.plugins {
		p.Routes(mux)
		if assets := p.Assets(); assets != nil {
			prefix := "/plugins/" + p.NavTab().ID + "/"
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
			http.Error(w, err.Error(), 400)
			return
		}
		if err := config.Save(s.cfgPath, *s.cfg); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		s.reconnect()
		httputil.WriteJSON(w, s.cfg)
	default:
		http.Error(w, "method not allowed", 405)
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
		http.Error(w, "method not allowed", 405)
	}
}

// -- Plugin meta endpoints --

func (s *Server) handleNav(w http.ResponseWriter, r *http.Request) {
	disabled := make(map[string]bool, len(s.cfg.DisabledPlugins))
	for _, id := range s.cfg.DisabledPlugins {
		disabled[id] = true
	}
	tabs := make([]plugin.NavTab, 0, len(s.plugins))
	for _, p := range s.plugins {
		if disabled[p.ID()] {
			continue
		}
		if tab := p.NavTab(); tab.ID != "" {
			tabs = append(tabs, tab)
		}
	}
	sort.Slice(tabs, func(i, j int) bool { return tabs[i].Order < tabs[j].Order })
	httputil.WriteJSON(w, tabs)
}

// handlePluginView serves GET /api/plugins/{id}/view — returns the plugin's
// view.html fragment so the frontend can inject it into the page dynamically.
func (s *Server) handlePluginView(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/plugins/")
	id = strings.TrimSuffix(id, "/view")
	id = strings.Trim(id, "/")
	if id == "" {
		http.Error(w, "missing plugin id", 400)
		return
	}
	var target plugin.Plugin
	for _, p := range s.plugins {
		if p.NavTab().ID == id {
			target = p
			break
		}
	}
	if target == nil {
		http.Error(w, "plugin not found", 404)
		return
	}
	assets := target.Assets()
	if assets == nil {
		http.Error(w, "plugin has no assets", 404)
		return
	}
	b, err := fs.ReadFile(assets, "view.html")
	if err != nil {
		http.Error(w, "view.html not found", 404)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(b)
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
	disabled := make(map[string]bool, len(s.cfg.DisabledPlugins))
	for _, id := range s.cfg.DisabledPlugins {
		disabled[id] = true
	}
	blobs := make([]plugin.PluginSettingsBlob, 0, len(s.plugins)+1)
	for _, p := range s.plugins {
		tab := p.NavTab()
		title := tab.Label
		if title == "" {
			title = p.ID()
		}
		blobs = append(blobs, plugin.PluginSettingsBlob{
			PluginID: p.ID(),
			NavTabID: tab.ID,
			Title:    title,
			Enabled:  !disabled[p.ID()],
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
		http.Error(w, "method not allowed", 405)
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
		http.Error(w, "method not allowed", 405)
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

func (s *Server) handleMomentumTimelineSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	provider := s.Timeline()
	if provider == nil {
		http.Error(w, "momentum timeline unavailable", http.StatusServiceUnavailable)
		return
	}
	httputil.WriteJSON(w, provider.Snapshot())
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
		http.Error(w, "missing id", 400)
		return
	}

	sep := strings.IndexAny(id, "|:_")
	if sep < 1 {
		http.Error(w, "invalid id format, expected platform|id", 400)
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
