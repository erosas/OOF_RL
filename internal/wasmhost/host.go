// Package wasmhost loads .wasm plugin files and exposes each one as a
// plugin.Plugin so the server can treat them identically to compiled plugins.
package wasmhost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

const metaBufSize = 4 * 1024       // 4 KB — more than enough for metadata JSON
const respBufSize = 4 * 1024 * 1024 // 4 MB — large enough for binary payloads (e.g. screenshots)

type eventMsg struct {
	eventType string
	payload   []byte
}

// Plugin wraps a compiled WASM module and implements plugin.Plugin.
type Plugin struct {
	plugin.BasePlugin

	ctx       context.Context
	runtime   wazero.Runtime
	mod       api.Module
	meta      sdk.PluginMeta
	assetsDir string              // path to co-located assets directory, empty if none
	bus       oofevents.PluginBus // set during Init; used for event publishing
	database  *db.DB
	hub       *hub.Hub
	cfg       *config.Config

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
			log.Printf("wasmhost: create plugin data dir %s: %v", pluginDataDir, mkErr)
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

	p := &Plugin{ctx: ctx, runtime: r, database: database, hub: h, cfg: cfg}

	// Register host-provided functions. Instantiation resolves imports, so
	// the host module must exist before InstantiateModule is called.
	if _, err := r.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(p.hostLog).Export("host_log").
		NewFunctionBuilder().WithFunc(p.hostPublishEvent).Export("host_publish_event").
		NewFunctionBuilder().WithFunc(p.hostDBExec).Export("host_db_exec").
		NewFunctionBuilder().WithFunc(p.hostDBQuery).Export("host_db_query").
		NewFunctionBuilder().WithFunc(p.hostHTTPFetch).Export("host_http_fetch").
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
func (p *Plugin) DBPrefix() string   { return "" }
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

func (p *Plugin) Routes(mux *http.ServeMux) {
	for _, route := range p.meta.Routes {
		path := route.Path
		if path == "" {
			continue
		}
		method := strings.ToUpper(strings.TrimSpace(route.Method))
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if method != "" && r.Method != method {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			p.serveHTTP(w, r)
		})
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
		case plugin.SettingTypeText, plugin.SettingTypeNumber, plugin.SettingTypeCheckbox, plugin.SettingTypePassword, plugin.SettingTypeSelect:
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
		out[i] = plugin.Setting{
			Key:         s.Key,
			Label:       s.Label,
			Description: s.Description,
			Type:        t,
			Default:     s.Default,
			Options:     options,
			Placeholder: s.Placeholder,
			Developer:   s.Developer,
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

func validateRouteMeta(routes []sdk.RouteMeta) error {
	seenPath := make(map[string]struct{}, len(routes))
	for _, r := range routes {
		path := strings.TrimSpace(r.Path)
		if path == "" {
			return fmt.Errorf("route path is required")
		}
		if !strings.HasPrefix(path, "/") {
			return fmt.Errorf("route path must start with '/': %q", path)
		}
		if _, exists := seenPath[path]; exists {
			return fmt.Errorf("duplicate route path %q", path)
		}
		seenPath[path] = struct{}{}
		method := strings.ToUpper(strings.TrimSpace(r.Method))
		if method == "" {
			continue
		}
		switch method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions:
		default:
			return fmt.Errorf("unsupported route method %q for %q", method, path)
		}
	}
	return nil
}

func validateDeclaredEventsMeta(events []sdk.DeclaredEvent) error {
	seen := make(map[string]struct{}, len(events))
	for _, e := range events {
		typeName := strings.TrimSpace(e.Type)
		if typeName == "" {
			return fmt.Errorf("declared event type is required")
		}
		if _, exists := seen[typeName]; exists {
			return fmt.Errorf("duplicate declared event type %q", typeName)
		}
		seen[typeName] = struct{}{}
		if e.Certainty < sdk.Authoritative || e.Certainty > sdk.Signal {
			return fmt.Errorf("invalid certainty %d for declared event %q", e.Certainty, typeName)
		}
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

func (p *Plugin) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if p.fnHandleHTTP == nil {
		http.Error(w, "plugin has no HTTP handler", http.StatusNotImplemented)
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	var bodyStr string
	if r.Body != nil {
		if b, err := io.ReadAll(r.Body); err == nil {
			bodyStr = string(b)
		}
	}
	req := sdk.HTTPRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: bodyStr}
	reqJSON, _ := json.Marshal(req)

	reqPtr, reqSize, err := p.writeGuest(reqJSON)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer p.free(reqPtr, reqSize)

	outPtr, err := p.malloc(respBufSize)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer p.free(outPtr, respBufSize)

	res, err := p.fnHandleHTTP.Call(p.ctx,
		api.EncodeU32(reqPtr), api.EncodeU32(uint32(len(reqJSON))),
		api.EncodeU32(outPtr), api.EncodeU32(respBufSize),
	)
	if err != nil {
		log.Printf("[wasm:%s] plugin_handle_http: %v", p.meta.ID, err)
		http.Error(w, "plugin error", http.StatusInternalServerError)
		return
	}

	respData, ok := p.mod.Memory().Read(outPtr, api.DecodeU32(res[0]))
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var resp sdk.HTTPResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		log.Printf("[wasm:%s] parse response: %v", p.meta.ID, err)
		http.Error(w, "plugin error", http.StatusInternalServerError)
		return
	}

	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}
	w.WriteHeader(resp.Status)
	fmt.Fprint(w, resp.Body)
}

// hostPublishEvent is called by the guest via the "host_publish_event" import.
// It publishes a RawEvent onto the bus so other plugins can subscribe to it.
func (p *Plugin) hostPublishEvent(_ context.Context, m api.Module, certainty, typePtr, typeLen, payloadPtr, payloadLen uint32) {
	if p.bus == nil {
		return
	}
	typeBytes, ok := m.Memory().Read(typePtr, typeLen)
	if !ok {
		log.Printf("[wasm:%s] host_publish_event: type read failed", p.meta.ID)
		return
	}
	payload, ok := m.Memory().Read(payloadPtr, payloadLen)
	if !ok {
		log.Printf("[wasm:%s] host_publish_event: payload read failed", p.meta.ID)
		return
	}
	e := oofevents.RawEvent{
		Base: oofevents.Base{
			EventType: string(typeBytes),
			At:        time.Now(),
			Cert:      oofevents.Certainty(certainty),
		},
		Payload: json.RawMessage(payload),
	}
	switch e.Cert {
	case oofevents.Authoritative:
		p.bus.PublishAuthoritative(e)
	case oofevents.Inferred:
		p.bus.PublishInferred(e)
	case oofevents.Signal:
		p.bus.PublishSignal(e)
	default:
		log.Printf("[wasm:%s] host_publish_event: unknown certainty %d, dropping", p.meta.ID, certainty)
	}
}

// hostLog is called by the guest via the "host_log" import and writes to the host's logger.
func (p *Plugin) hostLog(_ context.Context, m api.Module, _, ptr, length uint32) {
	data, _ := m.Memory().Read(ptr, length)
	log.Printf("[wasm:%s] %s", p.meta.ID, data)
}

// hostDBExec executes a SQL statement with JSON-encoded args ([]string).
// Writes the rows-affected int64 as JSON to outPtr. Returns bytes written, 0 on error.
func (p *Plugin) hostDBExec(_ context.Context, m api.Module, sqlPtr, sqlLen, argsPtr, argsLen, outPtr, outMax uint32) uint32 {
	if p.database == nil {
		return 0
	}
	sqlBytes, ok := m.Memory().Read(sqlPtr, sqlLen)
	if !ok {
		return 0
	}
	argsBytes, ok := m.Memory().Read(argsPtr, argsLen)
	if !ok {
		return 0
	}
	var args []any
	if err := json.Unmarshal(argsBytes, &args); err != nil {
		log.Printf("[wasm:%s] host_db_exec: parse args: %v", p.meta.ID, err)
		return 0
	}
	result, err := p.database.Conn().Exec(string(sqlBytes), args...)
	if err != nil {
		log.Printf("[wasm:%s] host_db_exec: %v", p.meta.ID, err)
		return 0
	}
	n, _ := result.RowsAffected()
	out, err := json.Marshal(n)
	if err != nil {
		return 0
	}
	return p.writeResult(m, out, outPtr, outMax)
}

// hostDBQuery executes a SQL query with JSON-encoded args and writes the result
// rows as a JSON array of column→value maps to outPtr.
func (p *Plugin) hostDBQuery(_ context.Context, m api.Module, sqlPtr, sqlLen, argsPtr, argsLen, outPtr, outMax uint32) uint32 {
	if p.database == nil {
		return 0
	}
	sqlBytes, ok := m.Memory().Read(sqlPtr, sqlLen)
	if !ok {
		return 0
	}
	argsBytes, ok := m.Memory().Read(argsPtr, argsLen)
	if !ok {
		return 0
	}
	var args []any
	if err := json.Unmarshal(argsBytes, &args); err != nil {
		log.Printf("[wasm:%s] host_db_query: parse args: %v", p.meta.ID, err)
		return 0
	}
	rows, err := p.database.Conn().Query(string(sqlBytes), args...)
	if err != nil {
		log.Printf("[wasm:%s] host_db_query: %v", p.meta.ID, err)
		return 0
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return 0
	}
	var result []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			log.Printf("[wasm:%s] host_db_query: scan row: %v", p.meta.ID, err)
			return 0
		}
		row := make(map[string]any, len(cols))
		for i, col := range cols {
			switch v := vals[i].(type) {
			case []byte:
				row[col] = string(v)
			case time.Time:
				row[col] = v.Format(time.RFC3339)
			default:
				row[col] = v
			}
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[wasm:%s] host_db_query: rows: %v", p.meta.ID, err)
		return 0
	}
	if result == nil {
		result = []map[string]any{}
	}
	out, err := json.Marshal(result)
	if err != nil {
		return 0
	}
	return p.writeResult(m, out, outPtr, outMax)
}

// hostHTTPFetch performs an outbound HTTP request.
// reqPtr/reqLen is a JSON object: {method, url, headers?, body?}.
// Writes a JSON-encoded HTTPFetchResult to outPtr.
func (p *Plugin) hostHTTPFetch(_ context.Context, m api.Module, reqPtr, reqLen, outPtr, outMax uint32) uint32 {
	reqBytes, ok := m.Memory().Read(reqPtr, reqLen)
	if !ok {
		return 0
	}
	var req sdk.HTTPFetchRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		log.Printf("[wasm:%s] host_http_fetch: parse req: %v", p.meta.ID, err)
		return 0
	}
	if req.Method == "" {
		req.Method = http.MethodGet
	}

	var bodyReader io.Reader
	if len(req.BodyBytes) > 0 {
		bodyReader = bytes.NewReader(req.BodyBytes)
	} else if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	}
	httpReq, err := http.NewRequest(req.Method, req.URL, bodyReader)
	if err != nil {
		return p.writeHTTPFetchError(m, "bad request: "+err.Error(), outPtr, outMax)
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return p.writeHTTPFetchError(m, err.Error(), outPtr, outMax)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, int64(outMax)+1)
	bodyBuilder, err := io.ReadAll(limited)
	if err != nil {
		return p.writeHTTPFetchError(m, "read body: "+err.Error(), outPtr, outMax)
	}
	if uint32(len(bodyBuilder)) > outMax {
		return p.writeHTTPFetchError(m, fmt.Sprintf("response body exceeds buffer (%d bytes)", outMax), outPtr, outMax)
	}

	respHeaders := make(map[string]string, len(resp.Header))
	for k := range resp.Header {
		respHeaders[k] = resp.Header.Get(k)
	}

	result := sdk.HTTPFetchResult{
		Status:  resp.StatusCode,
		Headers: respHeaders,
		Body:    string(bodyBuilder),
	}
	out, _ := json.Marshal(result)
	return p.writeResult(m, out, outPtr, outMax)
}

func (p *Plugin) writeHTTPFetchError(m api.Module, msg string, outPtr, outMax uint32) uint32 {
	result := sdk.HTTPFetchResult{Error: msg}
	out, _ := json.Marshal(result)
	return p.writeResult(m, out, outPtr, outMax)
}

// hostBroadcastWS sends a raw byte message to all connected WebSocket clients.
func (p *Plugin) hostBroadcastWS(_ context.Context, m api.Module, ptr, length uint32) {
	if p.hub == nil {
		return
	}
	data, ok := m.Memory().Read(ptr, length)
	if !ok {
		return
	}
	msg := make([]byte, len(data))
	copy(msg, data)
	p.hub.Broadcast(msg)
}

// hostGetConfig looks up a config key and writes the value string to outPtr.
func (p *Plugin) hostGetConfig(_ context.Context, m api.Module, keyPtr, keyLen, outPtr, outMax uint32) uint32 {
	if p.cfg == nil {
		return 0
	}
	keyBytes, ok := m.Memory().Read(keyPtr, keyLen)
	if !ok {
		return 0
	}
	val := p.cfg.Lookup(string(keyBytes))
	return p.writeResult(m, []byte(val), outPtr, outMax)
}

// writeResult copies data into guest memory at outPtr.
// Returns the number of bytes written, or 0 if data exceeds outMax (truncation would
// corrupt JSON-encoded results, so it is treated as an error).
func (p *Plugin) writeResult(m api.Module, data []byte, outPtr, outMax uint32) uint32 {
	if uint32(len(data)) > outMax {
		log.Printf("[wasm:%s] writeResult: result size %d exceeds buffer %d", p.meta.ID, len(data), outMax)
		return 0
	}
	if !m.Memory().Write(outPtr, data) {
		return 0
	}
	return uint32(len(data))
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
