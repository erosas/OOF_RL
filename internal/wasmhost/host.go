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
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	sdk "github.com/erosas/oof-plugin-sdk"
	"OOF_RL/internal/db"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/plugin"
)

const metaBufSize = 4 * 1024  // 4 KB — more than enough for metadata JSON
const respBufSize = 64 * 1024 // 64 KB — max HTTP response size

// Plugin wraps a compiled WASM module and implements plugin.Plugin.
type Plugin struct {
	plugin.BasePlugin

	ctx       context.Context
	runtime   wazero.Runtime
	mod       api.Module
	meta      sdk.PluginMeta
	assetsDir string              // path to co-located assets directory, empty if none
	bus       oofevents.PluginBus // set during Init; used for event publishing

	fnMalloc     api.Function
	fnFree       api.Function
	fnInit       api.Function
	fnOnEvent    api.Function
	fnHandleHTTP api.Function
	fnShutdown   api.Function
}

// Load reads the .wasm file at path and returns an initialised Plugin.
// If a directory named <pluginID> exists adjacent to the .wasm file it is
// used to serve static assets (view.html, JS, etc).
func Load(path string) (*Plugin, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("wasmhost: read %s: %w", path, err)
	}
	p, err := loadBytes(b)
	if err != nil {
		return nil, err
	}
	assetsDir := filepath.Join(filepath.Dir(path), p.meta.ID)
	if info, statErr := os.Stat(assetsDir); statErr == nil && info.IsDir() {
		p.assetsDir = assetsDir
	}
	return p, nil
}

func loadBytes(wasmBytes []byte) (*Plugin, error) {
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	p := &Plugin{ctx: ctx, runtime: r}

	// Register host-provided functions. All imports must be declared before
	// the guest module is compiled/instantiated.
	if _, err := r.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(p.hostLog).Export("host_log").
		NewFunctionBuilder().WithFunc(p.hostPublishEvent).Export("host_publish_event").
		Instantiate(ctx); err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("wasmhost: host module: %w", err)
	}

	compiled, err := r.CompileModule(ctx, wasmBytes)
	if err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("wasmhost: compile: %w", err)
	}

	// Plugins must be built with -buildmode=c-shared (WASI reactor).
	// _initialize fully starts the Go runtime; _start would exit the module
	// after main() returns, preventing subsequent exported-function calls.
	// Omit WithStdout/WithStderr — plugin logging goes through host_log.
	mod, err := r.InstantiateModule(ctx, compiled,
		wazero.NewModuleConfig().
			WithStartFunctions("_initialize").
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
		route := route
		mux.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			p.serveHTTP(w, r)
		})
	}
}

func (p *Plugin) SettingsSchema() []plugin.Setting        { return nil }
func (p *Plugin) ApplySettings(_ map[string]string) error { return nil }

// Init stores the bus, calls plugin_init with an empty config, then subscribes
// to the event types declared in the plugin's metadata.
func (p *Plugin) Init(bus oofevents.PluginBus, _ plugin.Registry, _ *db.DB) error {
	p.bus = bus

	if p.fnInit != nil {
		cfgJSON := []byte("{}")
		cfgPtr, err := p.writeGuest(cfgJSON)
		if err != nil {
			return fmt.Errorf("wasm:%s plugin_init write: %w", p.meta.ID, err)
		}
		defer p.free(cfgPtr, uint32(len(cfgJSON)))
		res, err := p.fnInit.Call(p.ctx, api.EncodeU32(cfgPtr), api.EncodeU32(uint32(len(cfgJSON))))
		if err != nil {
			return fmt.Errorf("wasm:%s plugin_init: %w", p.meta.ID, err)
		}
		if api.DecodeU32(res[0]) != 0 {
			return fmt.Errorf("wasm:%s plugin_init: returned error code %d", p.meta.ID, api.DecodeU32(res[0]))
		}
	}

	for _, eventType := range p.meta.Events {
		eventType := eventType
		p.AddSub(bus.Subscribe(eventType, func(e oofevents.OOFEvent) {
			payload, err := json.Marshal(e)
			if err != nil {
				log.Printf("[wasm:%s] marshal event %s: %v", p.meta.ID, eventType, err)
				return
			}
			p.dispatchEvent(eventType, payload)
		}))
	}
	return nil
}

func (p *Plugin) Shutdown() error {
	p.BasePlugin.Shutdown()
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
	return nil
}

func (p *Plugin) dispatchEvent(eventType string, payload []byte) {
	if p.fnOnEvent == nil {
		return
	}

	typePtr, err := p.writeGuest([]byte(eventType))
	if err != nil {
		log.Printf("[wasm:%s] write event type: %v", p.meta.ID, err)
		return
	}
	defer p.free(typePtr, uint32(len(eventType)))

	payloadPtr, err := p.writeGuest(payload)
	if err != nil {
		log.Printf("[wasm:%s] write event payload: %v", p.meta.ID, err)
		return
	}
	defer p.free(payloadPtr, uint32(len(payload)))

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

	req := sdk.HTTPRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery}
	reqJSON, _ := json.Marshal(req)

	reqPtr, err := p.writeGuest(reqJSON)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer p.free(reqPtr, uint32(len(reqJSON)))

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

func (p *Plugin) hostLog(_ context.Context, m api.Module, _, ptr, length uint32) {
	data, _ := m.Memory().Read(ptr, length)
	log.Printf("[wasm:%s] %s", p.meta.ID, data)
}

func (p *Plugin) malloc(size uint32) (uint32, error) {
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

func (p *Plugin) writeGuest(data []byte) (uint32, error) {
	ptr, err := p.malloc(uint32(len(data)))
	if err != nil {
		return 0, err
	}
	if !p.mod.Memory().Write(ptr, data) {
		p.free(ptr, uint32(len(data)))
		return 0, fmt.Errorf("memory write failed")
	}
	return ptr, nil
}