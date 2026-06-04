package core

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/websocket"

	"OOF_RL/internal/config"
	"OOF_RL/internal/db"
	"OOF_RL/internal/histstore"
	"OOF_RL/internal/hub"
	"OOF_RL/internal/mmr"
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
	cfg                  *config.Config
	cfgPath              string
	db                   *db.DB
	hub                  *hub.Hub
	fs                   http.Handler
	reconnect    func()
	mmrProvider  mmr.Provider
	plugins      []plugin.Plugin
	disabled     map[string]struct{} // computed once from cfg.DisabledPlugins
	activeCached []plugin.Plugin
	activeByID   map[string]plugin.Plugin
	activeDirty  bool
	bus          oofevents.Bus
	translator   *rlevents.Translator
	histStore    *histstore.Store
	histRecorder *histstore.Recorder
}

func isHostCorePluginID(pluginID string) bool {
	return pluginID == "history"
}

func NewServer(cfgPath string, cfg *config.Config, database *db.DB, h *hub.Hub, static http.Handler, reconnect func(), mmrProvider mmr.Provider, bus oofevents.Bus) *Server {
	disabled := make(map[string]struct{}, len(cfg.DisabledPlugins))
	for _, id := range cfg.DisabledPlugins {
		disabled[id] = struct{}{}
	}
	s := &Server{
		cfgPath:     cfgPath,
		cfg:         cfg,
		db:          database,
		hub:         h,
		fs:          static,
		reconnect:   reconnect,
		mmrProvider: mmrProvider,
		disabled:    disabled,
		activeDirty: true,
		bus:        bus,
		translator: rlevents.New(bus.ForPlugin("")), // empty ID = RL translator convention
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
	disabled := s.disabled
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
	s.activeDirty = true
}

func isPluginDisabledInSet(pluginID string, disabled map[string]struct{}) bool {
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
	if s.activeDirty {
		s.rebuildActivePluginCache()
	}
	return s.activeCached
}

func (s *Server) findPluginTarget(pluginID string) plugin.Plugin {
	s.activePlugins()
	if p, ok := s.activeByID[pluginID]; ok {
		return p
	}
	return nil
}

func (s *Server) rebuildActivePluginCache() {
	active := make([]plugin.Plugin, 0, len(s.plugins))
	byID := make(map[string]plugin.Plugin, len(s.plugins))
	for _, p := range s.plugins {
		if isPluginDisabledInSet(p.ID(), s.disabled) {
			continue
		}
		active = append(active, p)
		byID[p.ID()] = p
	}
	s.activeCached = active
	s.activeByID = byID
	s.activeDirty = false
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
		"/plugins/history/":    "core",
	}

	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/config/ini", s.handleINI)
	mux.HandleFunc("/api/overlay/opacity", s.handleOverlayOpacityPreview)
	mux.HandleFunc("/api/nav", s.handleNav)
	mux.HandleFunc("/api/plugins/", s.handlePluginView)
	if s.histStore != nil {
		mux.HandleFunc("/api/players", s.histStore.HandlePlayers)
		mux.HandleFunc("/api/matches", s.histStore.HandleMatches)
		mux.HandleFunc("/api/matches/", s.histStore.HandleMatchDetail)
	}
	mux.HandleFunc("/api/settings/schema", s.handleSettingsSchema)
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/api/tracker/profile", mmr.Handler(s.mmrProvider))
	mux.HandleFunc("/api/db/open-folder", s.handleDBOpenFolder)
	mux.HandleFunc("/api/data-dir", s.handleDataDir)

	// History is host-core: mount its UI assets directly.
	if histAssets, err := fs.Sub(histstore.Assets, "assets"); err == nil {
		mux.Handle("/plugins/history/", http.StripPrefix("/plugins/history/", http.FileServer(http.FS(histAssets))))
	}

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
			if owner, ok := registered[prefix]; ok {
				log.Printf("[core] plugin %q: asset prefix %q conflicts with %q — plugin assets not registered", p.ID(), prefix, owner)
			} else {
				registered[prefix] = p.ID()
				mux.Handle(prefix, http.StripPrefix(prefix, http.FileServer(http.FS(assets))))
			}
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
