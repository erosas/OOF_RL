// Package wasmhost loads .wasm plugin files and exposes each one as a
// plugin.Plugin so the server can treat them identically to compiled plugins.
package wasmhost

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"OOF_RL/internal/config"
	"OOF_RL/internal/db"
	"OOF_RL/internal/hub"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/plugin"

	sdk "github.com/erosas/oof-plugin-sdk"
)

const metaBufSize = 4 * 1024 // 4 KB — more than enough for metadata JSON

type eventMsg struct {
	eventType string
	payload   []byte
}

// Plugin wraps a compiled WASM module and implements plugin.Plugin.
type Plugin struct {
	plugin.BasePlugin

	ctx        context.Context
	runtime    wazero.Runtime
	mod        api.Module
	meta       sdk.PluginMeta
	assetsDir  string              // path to co-located assets directory, empty if none
	bus        oofevents.PluginBus // set during Init; used for event publishing
	database   *db.DB
	hub        *hub.Hub
	cfg        *config.Config
	httpClient *http.Client
	mounts     map[string]string // WASI virtual path prefix → real OS directory

	fnMalloc        api.Function
	fnFree          api.Function
	fnInit          api.Function
	fnOnEvent       api.Function
	fnHandleHTTP    api.Function
	fnShutdown      api.Function
	fnApplySettings api.Function

	// mu serializes all calls into the WASM module. wazero module instances are
	// not goroutine-safe; concurrent HTTP handlers and the event worker goroutine
	// both call into the same module and must not overlap.
	mu sync.Mutex

	eventCh chan eventMsg
	wg      sync.WaitGroup
}

// Load reads the .wasm file at path and returns an initialised Plugin.
// If a directory named <pluginID> exists adjacent to the .wasm file it is
// used to serve static assets (view.html, JS, etc).
// database, h, and cfg may be nil (e.g. in tests) — host imports that require
// them will be no-ops when their dependency is absent.
//
// Each plugin receives two WASI-mounted directories:
//   - /replays → the configured Rocket League replay directory
//   - /data    → <data_dir>/plugin_data/<plugin_id>/  (created if absent)
func Load(path string, database *db.DB, h *hub.Hub, cfg *config.Config) (*Plugin, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("wasmhost: read %s: %w", path, err)
	}

	// Derive plugin ID from filename convention (<id>.wasm) so we can create
	// the per-plugin data directory before instantiation.
	pluginID := strings.TrimSuffix(filepath.Base(path), ".wasm")

	var replayDir, pluginDataDir string
	if cfg != nil {
		replayDir = cfg.Lookup("replay_dir")
		pluginDataDir = filepath.Join(cfg.Lookup("data_dir"), "plugin_data", pluginID)
		if mkErr := os.MkdirAll(pluginDataDir, 0755); mkErr != nil {
			log.Printf("wasmhost: create plugin data dir for plugin %q failed: %v", pluginID, mkErr)
			pluginDataDir = ""
		}
	}

	p, err := loadBytes(b, database, h, cfg, replayDir, pluginDataDir)
	if err != nil {
		return nil, err
	}
	assetsDir := filepath.Join(filepath.Dir(path), p.meta.ID)
	if info, statErr := os.Stat(assetsDir); statErr == nil && info.IsDir() {
		p.assetsDir = assetsDir
	}
	return p, nil
}

func loadBytes(wasmBytes []byte, database *db.DB, h *hub.Hub, cfg *config.Config, replayDir, pluginDataDir string) (*Plugin, error) {
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	mounts := map[string]string{}
	if replayDir != "" {
		mounts["/replays"] = replayDir
	}
	if pluginDataDir != "" {
		mounts["/data"] = pluginDataDir
	}
	p := &Plugin{
		ctx:      ctx,
		runtime:  r,
		database: database,
		hub:      h,
		cfg:      cfg,
		mounts:   mounts,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Register host-provided functions. Instantiation resolves imports, so
	// the host module must exist before InstantiateModule is called.
	if _, err := r.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(p.hostLog).Export("host_log").
		NewFunctionBuilder().WithFunc(p.hostPublishEvent).Export("host_publish_event").
		NewFunctionBuilder().WithFunc(p.hostDBExec).Export("host_db_exec").
		NewFunctionBuilder().WithFunc(p.hostDBQuery).Export("host_db_query").
		NewFunctionBuilder().WithFunc(p.hostHTTPFetch).Export("host_http_fetch").
		NewFunctionBuilder().WithFunc(p.hostHTTPDownload).Export("host_http_download").
		NewFunctionBuilder().WithFunc(p.hostUploadFile).Export("host_upload_file").
		NewFunctionBuilder().WithFunc(p.hostBroadcastWS).Export("host_broadcast_ws").
		NewFunctionBuilder().WithFunc(p.hostGetConfig).Export("host_get_config").
		Instantiate(ctx); err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("wasmhost: host module: %w", err)
	}

	compiled, err := r.CompileModule(ctx, wasmBytes)
	if err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("wasmhost: compile: %w", err)
	}

	// Mount /replays and /data into the WASM sandbox. Plugins use standard os
	// package calls to access these directories; no other paths are visible.
	fsCfg := wazero.NewFSConfig()
	if replayDir != "" {
		fsCfg = fsCfg.WithDirMount(replayDir, "/replays")
	}
	if pluginDataDir != "" {
		fsCfg = fsCfg.WithDirMount(pluginDataDir, "/data")
	}

	// Plugins must be built with -buildmode=c-shared (WASI reactor).
	// _initialize fully starts the Go runtime; _start would exit the module
	// after main() returns, preventing subsequent exported-function calls.
	// Omit WithStdout/WithStderr — plugin logging goes through host_log.
	mod, err := r.InstantiateModule(ctx, compiled,
		wazero.NewModuleConfig().
			WithStartFunctions("_initialize").
			WithFSConfig(fsCfg).
			WithSysWalltime().
			WithName(""))
	if err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("wasmhost: instantiate: %w", err)
	}
	p.mod = mod

	p.fnMalloc = mod.ExportedFunction("malloc")
	p.fnFree = mod.ExportedFunction("free")
	p.fnInit = mod.ExportedFunction("plugin_init")
	p.fnOnEvent = mod.ExportedFunction("plugin_on_event")
	p.fnHandleHTTP = mod.ExportedFunction("plugin_handle_http")
	p.fnShutdown = mod.ExportedFunction("plugin_shutdown")
	p.fnApplySettings = mod.ExportedFunction("plugin_apply_settings")

	if err := p.readMeta(); err != nil {
		mod.Close(ctx)
		r.Close(ctx)
		return nil, err
	}
	return p, nil
}

// -- plugin.Plugin interface --

func (p *Plugin) ID() string         { return p.meta.ID }
func (p *Plugin) Requires() []string { return p.meta.Requires }

func (p *Plugin) DeclaredEvents() []oofevents.EventDeclaration {
	out := make([]oofevents.EventDeclaration, len(p.meta.DeclaredEvents))
	for i, d := range p.meta.DeclaredEvents {
		out[i] = oofevents.EventDeclaration{
			Type:        d.Type,
			Certainty:   oofevents.Certainty(d.Certainty),
			Description: d.Description,
		}
	}
	return out
}

func (p *Plugin) Assets() fs.FS {
	if p.assetsDir == "" {
		return nil
	}
	return os.DirFS(p.assetsDir)
}

func (p *Plugin) NavTab() plugin.NavTab {
	return plugin.NavTab{
		ID:    p.meta.NavTab.ID,
		Label: p.meta.NavTab.Label,
		Order: p.meta.NavTab.Order,
	}
}

func (p *Plugin) SettingsSchema() []plugin.Setting {
	if len(p.meta.Settings) == 0 {
		return nil
	}
	out := make([]plugin.Setting, len(p.meta.Settings))
	for i, s := range p.meta.Settings {
		t := plugin.SettingType(s.Type)
		switch t {
		case plugin.SettingTypeText, plugin.SettingTypeNumber, plugin.SettingTypeCheckbox, plugin.SettingTypePassword, plugin.SettingTypeSelect, plugin.SettingTypeAction:
		default:
			t = plugin.SettingTypeText
		}
		if s.Secret && s.Type == "" {
			t = plugin.SettingTypePassword
		}
		options := make([]plugin.SelectOption, 0, len(s.Options))
		for _, opt := range s.Options {
			options = append(options, plugin.SelectOption{Value: opt.Value, Label: opt.Label})
		}
		label := s.Label
		if label == "" {
			label = s.Key
		}
		out[i] = plugin.Setting{
			Key:          s.Key,
			Label:        label,
			Description:  s.Description,
			Type:         t,
			Default:      s.Default,
			Options:      options,
			Placeholder:  s.Placeholder,
			ActionPath:   s.ActionPath,
			ActionMethod: s.ActionMethod,
			StatusPath:   s.StatusPath,
			DownloadPath: s.DownloadPath,
			Developer:    s.Developer,
		}
	}
	return out
}

func (p *Plugin) ApplySettings(settings map[string]string) error {
	if p.fnApplySettings == nil {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	cfgJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("wasm:%s ApplySettings marshal: %w", p.meta.ID, err)
	}
	ptr, size, err := p.writeGuest(cfgJSON)
	if err != nil {
		return fmt.Errorf("wasm:%s ApplySettings write: %w", p.meta.ID, err)
	}
	defer p.free(ptr, size)
	res, err := p.fnApplySettings.Call(p.ctx, api.EncodeU32(ptr), api.EncodeU32(uint32(len(cfgJSON))))
	if err != nil {
		return fmt.Errorf("wasm:%s plugin_apply_settings: %w", p.meta.ID, err)
	}
	if api.DecodeU32(res[0]) != 0 {
		return fmt.Errorf("wasm:%s plugin_apply_settings: returned error code %d", p.meta.ID, api.DecodeU32(res[0]))
	}
	return nil
}

// Init stores the bus, calls plugin_init, delivers initial settings via
// plugin_apply_settings, then subscribes to declared event types.
func (p *Plugin) Init(bus oofevents.PluginBus, _ plugin.Registry, _ *db.DB) error {
	p.bus = bus

	if p.fnInit != nil {
		cfgJSON := []byte("{}")
		cfgPtr, cfgSize, err := p.writeGuest(cfgJSON)
		if err != nil {
			return fmt.Errorf("wasm:%s plugin_init write: %w", p.meta.ID, err)
		}
		defer p.free(cfgPtr, cfgSize)
		res, err := p.fnInit.Call(p.ctx, api.EncodeU32(cfgPtr), api.EncodeU32(uint32(len(cfgJSON))))
		if err != nil {
			return fmt.Errorf("wasm:%s plugin_init: %w", p.meta.ID, err)
		}
		if api.DecodeU32(res[0]) != 0 {
			return fmt.Errorf("wasm:%s plugin_init: returned error code %d", p.meta.ID, api.DecodeU32(res[0]))
		}
	}

	// Deliver initial settings resolved from host config.
	if len(p.meta.Settings) > 0 && p.cfg != nil {
		resolved := make(map[string]string, len(p.meta.Settings))
		for _, s := range p.meta.Settings {
			resolved[s.Key] = p.cfg.Lookup(s.Key)
		}
		if err := p.ApplySettings(resolved); err != nil {
			log.Printf("[wasm:%s] ApplySettings on init: %v", p.meta.ID, err)
		}
	}

	// Worker goroutine: dequeues events and calls into WASM off the bus dispatch goroutine.
	// The bus requires subscribers to be non-blocking; WASM calls can take arbitrary time.
	p.eventCh = make(chan eventMsg, 64)
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for msg := range p.eventCh {
			p.dispatchEvent(msg.eventType, msg.payload)
		}
	}()

	for _, eventType := range p.meta.Events {
		p.AddSub(bus.Subscribe(eventType, func(e oofevents.OOFEvent) {
			// Unwrap strips the stampedEvent wrapper so the guest sees flat JSON fields.
			payload, err := json.Marshal(oofevents.Unwrap(e))
			if err != nil {
				log.Printf("[wasm:%s] marshal event %s: %v", p.meta.ID, eventType, err)
				return
			}
			select {
			case p.eventCh <- eventMsg{eventType, payload}:
			default:
				log.Printf("[wasm:%s] event queue full, dropping %s", p.meta.ID, eventType)
			}
		}))
	}
	return nil
}

func (p *Plugin) Shutdown() error {
	p.BasePlugin.Shutdown() // unsubscribes all; no new events will be enqueued after this
	if p.eventCh != nil {
		close(p.eventCh)
		p.wg.Wait()
	}
	if p.fnShutdown != nil {
		if _, err := p.fnShutdown.Call(p.ctx); err != nil {
			log.Printf("[wasm:%s] shutdown: %v", p.meta.ID, err)
		}
	}
	if p.mod != nil {
		p.mod.Close(p.ctx)
	}
	if p.runtime != nil {
		p.runtime.Close(p.ctx)
	}
	return nil
}

// -- internal --

func (p *Plugin) readMeta() error {
	fn := p.mod.ExportedFunction("plugin_metadata")
	if fn == nil {
		return fmt.Errorf("wasmhost: missing plugin_metadata export")
	}

	outPtr, err := p.malloc(metaBufSize)
	if err != nil {
		return fmt.Errorf("wasmhost: metadata alloc: %w", err)
	}
	defer p.free(outPtr, metaBufSize)

	res, err := fn.Call(p.ctx, api.EncodeU32(outPtr), api.EncodeU32(metaBufSize))
	if err != nil {
		return fmt.Errorf("wasmhost: plugin_metadata: %w", err)
	}

	n := api.DecodeU32(res[0])
	data, ok := p.mod.Memory().Read(outPtr, n)
	if !ok {
		return fmt.Errorf("wasmhost: plugin_metadata memory read failed")
	}

	if err := json.Unmarshal(data, &p.meta); err != nil {
		return fmt.Errorf("wasmhost: plugin_metadata unmarshal: %w", err)
	}
	if err := validateRouteMeta(p.meta.Routes); err != nil {
		return fmt.Errorf("wasmhost: plugin_metadata routes: %w", err)
	}
	if err := validateDeclaredEventsMeta(p.meta.DeclaredEvents); err != nil {
		return fmt.Errorf("wasmhost: plugin_metadata declared_events: %w", err)
	}
	return nil
}

func (p *Plugin) dispatchEvent(eventType string, payload []byte) {
	if p.fnOnEvent == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	typePtr, typeSize, err := p.writeGuest([]byte(eventType))
	if err != nil {
		log.Printf("[wasm:%s] write event type: %v", p.meta.ID, err)
		return
	}
	defer p.free(typePtr, typeSize)

	payloadPtr, payloadSize, err := p.writeGuest(payload)
	if err != nil {
		log.Printf("[wasm:%s] write event payload: %v", p.meta.ID, err)
		return
	}
	defer p.free(payloadPtr, payloadSize)

	if _, err := p.fnOnEvent.Call(p.ctx,
		api.EncodeU32(typePtr), api.EncodeU32(uint32(len(eventType))),
		api.EncodeU32(payloadPtr), api.EncodeU32(uint32(len(payload))),
	); err != nil {
		log.Printf("[wasm:%s] plugin_on_event: %v", p.meta.ID, err)
	}
}

func (p *Plugin) malloc(size uint32) (uint32, error) {
	if p.fnMalloc == nil {
		return 0, fmt.Errorf("plugin missing required malloc export")
	}
	res, err := p.fnMalloc.Call(p.ctx, api.EncodeU32(size))
	if err != nil {
		return 0, err
	}
	return api.DecodeU32(res[0]), nil
}

func (p *Plugin) free(ptr, size uint32) {
	if p.fnFree != nil {
		p.fnFree.Call(p.ctx, api.EncodeU32(ptr), api.EncodeU32(size))
	}
}

func (p *Plugin) writeGuest(data []byte) (ptr, size uint32, err error) {
	size = uint32(len(data))
	ptr, err = p.malloc(size)
	if err != nil {
		return 0, 0, err
	}
	if !p.mod.Memory().Write(ptr, data) {
		p.free(ptr, size)
		return 0, 0, fmt.Errorf("memory write failed")
	}
	return ptr, size, nil
}

// resolveWASIPath maps a plugin's WASI virtual path (e.g. "/replays/foo.replay")
// to the real OS path using the mount table populated at load time.
// Returns an error if the path is not within a known mount or attempts traversal.
func (p *Plugin) resolveWASIPath(wasiPath string) (string, error) {
	for prefix, realDir := range p.mounts {
		if wasiPath != prefix && !strings.HasPrefix(wasiPath, prefix+"/") {
			continue
		}
		rel := strings.TrimPrefix(wasiPath, prefix)
		resolved := filepath.Join(realDir, filepath.FromSlash(rel))
		// Guard against path traversal: resolved must stay inside realDir.
		realDirClean := filepath.Clean(realDir)
		if resolved != realDirClean && !strings.HasPrefix(resolved, realDirClean+string(filepath.Separator)) {
			return "", fmt.Errorf("wasmhost: path escapes mount %s: %s", prefix, wasiPath)
		}
		return resolved, nil
	}
	return "", fmt.Errorf("wasmhost: path not in any WASI mount: %s", wasiPath)
}
