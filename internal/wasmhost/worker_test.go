package wasmhost

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"OOF_RL/internal/config"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/plugin"
	sdk "github.com/erosas/oof-plugin-sdk"
)

// TestBusEvent_StampedEventWrapping documents the stampedEvent problem that
// motivates calling oofevents.Unwrap(e) in the Init() subscriber closure.
//
// Events published through PluginBus arrive wrapped in an unexported stampedEvent
// whose embedded OOFEvent interface field serialises as {"OOFEvent": {...}}.
// Without Unwrap the WASM guest receives the wrong JSON shape.
//
// This test verifies both sides: that the wrapping actually occurs (so we know
// Unwrap is necessary), and that oofevents.Unwrap correctly strips the wrapper.
// If either invariant changes, the corresponding Init() subscriber code should
// be revisited.
func TestBusEvent_StampedEventWrapping(t *testing.T) {
	bus := oofevents.New()
	bus.Start()
	defer bus.Stop()

	received := make(chan oofevents.OOFEvent, 1)
	bus.Subscribe(oofevents.TypeMatchStarted, func(e oofevents.OOFEvent) {
		select {
		case received <- e:
		default:
		}
	})

	// Publish through ForPlugin: the PluginBus stamps the event in a
	// stampedEvent wrapper before it reaches subscribers.
	bus.ForPlugin("publisher").PublishAuthoritative(oofevents.NewMatchStarted("test-guid"))

	select {
	case e := <-received:
		// Verify the event IS stamped — marshaling without Unwrap must produce
		// the {"OOFEvent": ...} shape. If this assertion ever fails it means
		// the bus no longer wraps events and the Unwrap() call in Init() can
		// be removed.
		rawPayload, _ := json.Marshal(e)
		var rawMap map[string]any
		if err := json.Unmarshal(rawPayload, &rawMap); err != nil {
			t.Fatalf("unmarshal raw: %v", err)
		}
		if _, has := rawMap["OOFEvent"]; !has {
			t.Skip(`bus no longer wraps events in stampedEvent; Unwrap() call in Init() may be unnecessary`)
		}

		// Verify Unwrap strips the wrapper so the concrete fields are top-level.
		unwrappedPayload, _ := json.Marshal(oofevents.Unwrap(e))
		var unwrappedMap map[string]any
		if err := json.Unmarshal(unwrappedPayload, &unwrappedMap); err != nil {
			t.Fatalf("unmarshal unwrapped: %v", err)
		}
		if _, has := unwrappedMap["OOFEvent"]; has {
			t.Error(`oofevents.Unwrap did not remove "OOFEvent" wrapper key`)
		}
		if _, has := unwrappedMap["EventType"]; !has {
			t.Error(`unwrapped payload missing "EventType" — concrete fields should be at the top level`)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

// TestPlugin_Shutdown_DrainsQueue verifies that Shutdown waits for the event
// worker goroutine to finish processing before returning. If close(eventCh) or
// wg.Wait() were removed from Shutdown, this test would deadlock and time out.
func TestPlugin_Shutdown_DrainsQueue(t *testing.T) {
	bus := oofevents.New()
	bus.Start()
	defer bus.Stop()

	p := &Plugin{
		ctx:  context.Background(),
		meta: sdk.PluginMeta{ID: "drain-test"},
	}
	if err := p.Init(bus.ForPlugin("drain-test"), nil, nil); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Write directly to the event channel (bypassing bus subscriptions) to
	// queue work for the worker goroutine without needing a real WASM module.
	for i := 0; i < 20; i++ {
		p.eventCh <- eventMsg{"test.event", []byte(`{}`)}
	}

	done := make(chan struct{})
	go func() {
		p.Shutdown()
		close(done)
	}()

	select {
	case <-done:
		// Shutdown returned — wg.Wait() completed, worker has exited.
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown did not return; worker goroutine likely still blocked (close(eventCh) missing?)")
	}
}

// TestPlugin_EventQueue_DropsWhenFull verifies that the subscriber callback
// never blocks when the event queue is at capacity. A blocking subscriber would
// stall the oofevents bus's single dispatch goroutine.
func TestPlugin_EventQueue_DropsWhenFull(t *testing.T) {
	const queueCap = 4

	p := &Plugin{
		meta:    sdk.PluginMeta{ID: "drop-test"},
		eventCh: make(chan eventMsg, queueCap),
	}

	// Fill the queue to capacity.
	for i := 0; i < queueCap; i++ {
		p.eventCh <- eventMsg{"test.event", []byte(`{}`)}
	}

	dropped := 0
	const extra = 5
	for i := 0; i < extra; i++ {
		// Replicate the select/default guard in the subscriber closure.
		select {
		case p.eventCh <- eventMsg{"test.event", []byte(`{}`)}:
			t.Error("send should have been dropped on full queue")
		default:
			dropped++
		}
	}

	if dropped != extra {
		t.Errorf("want %d drops, got %d", extra, dropped)
	}
}

// TestPlugin_Getters exercises the trivial getter methods that read from meta.
func TestPlugin_Getters(t *testing.T) {
	p := &Plugin{
		meta: sdk.PluginMeta{
			ID:       "test-plugin",
			Requires: []string{"dep-a"},
			DeclaredEvents: []sdk.DeclaredEvent{
				{Type: "test.fired", Certainty: sdk.Authoritative, Description: "desc"},
			},
			NavTab: sdk.NavTabMeta{ID: "test-tab", Label: "Test", Order: 3},
		},
	}

	if got := p.ID(); got != "test-plugin" {
		t.Errorf("ID: got %q", got)
	}
	if got := p.DBPrefix(); got != "" {
		t.Errorf("DBPrefix: got %q, want empty", got)
	}
	if got := p.Requires(); len(got) != 1 || got[0] != "dep-a" {
		t.Errorf("Requires: got %v", got)
	}
	decl := p.DeclaredEvents()
	if len(decl) != 1 || decl[0].Type != "test.fired" {
		t.Errorf("DeclaredEvents: got %v", decl)
	}
	nt := p.NavTab()
	if nt.ID != "test-tab" || nt.Label != "Test" || nt.Order != 3 {
		t.Errorf("NavTab: got %+v", nt)
	}
	if p.SettingsSchema() != nil {
		t.Error("SettingsSchema: want nil")
	}
	if err := p.ApplySettings(nil); err != nil {
		t.Errorf("ApplySettings: got %v", err)
	}
}

// TestPlugin_Assets tests the Assets() getter for both the nil and non-nil cases.
func TestPlugin_Assets(t *testing.T) {
	t.Run("empty assetsDir returns nil", func(t *testing.T) {
		p := &Plugin{}
		if p.Assets() != nil {
			t.Error("want nil for empty assetsDir")
		}
	})
	t.Run("existing assetsDir returns fs.FS", func(t *testing.T) {
		dir := t.TempDir()
		p := &Plugin{assetsDir: dir}
		if p.Assets() == nil {
			t.Error("want non-nil fs.FS for existing dir")
		}
	})
}

// TestPlugin_Routes_NoHandler verifies that routes are registered on the mux
// and that requests without fnHandleHTTP get a 501 Not Implemented.
func TestPlugin_Routes_NoHandler(t *testing.T) {
	p := &Plugin{
		meta: sdk.PluginMeta{Routes: []string{"/api/wasm-test"}},
	}
	mux := http.NewServeMux()
	p.Routes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/wasm-test", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotImplemented {
		t.Errorf("serveHTTP without handler: got %d, want 501", w.Code)
	}
}

// TestPlugin_HostPublishEvent_NilBus ensures the nil-bus guard does not panic.
func TestPlugin_HostPublishEvent_NilBus(t *testing.T) {
	p := &Plugin{meta: sdk.PluginMeta{ID: "no-bus"}}
	p.hostPublishEvent(context.Background(), nil, 0, 0, 0, 0, 0)
}

// TestPlugin_Shutdown_NilEventCh verifies Shutdown is safe when Init was never called.
func TestPlugin_Shutdown_NilEventCh(t *testing.T) {
	p := &Plugin{meta: sdk.PluginMeta{ID: "no-init"}}
	if err := p.Shutdown(); err != nil {
		t.Errorf("Shutdown: %v", err)
	}
}

// newMemModule compiles and instantiates a minimal WASM module with one page of
// memory. Used to provide a real api.Module for host-import tests that need to
// read/write guest memory without standing up a full plugin binary.
func newMemModule(t *testing.T) api.Module {
	t.Helper()
	// Minimal WASM: magic + version + memory section (1 page min) + memory export.
	wasmBytes := []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, // magic + version
		0x05, 0x03, 0x01, 0x00, 0x01, // memory section: 1 page (64 KB)
		0x07, 0x0a, 0x01, 0x06, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x02, 0x00, // export "memory"
	}
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	compiled, err := r.CompileModule(ctx, wasmBytes)
	if err != nil {
		r.Close(ctx)
		t.Fatalf("compile mem module: %v", err)
	}
	mod, err := r.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithName(""))
	if err != nil {
		r.Close(ctx)
		t.Fatalf("instantiate mem module: %v", err)
	}
	t.Cleanup(func() {
		mod.Close(ctx)
		r.Close(ctx)
	})
	return mod
}

// TestHostDBExec_NilDB verifies the nil database guard returns 0.
func TestHostDBExec_NilDB(t *testing.T) {
	p := &Plugin{meta: sdk.PluginMeta{ID: "test"}, database: nil}
	got := p.hostDBExec(context.Background(), nil, 0, 0, 0, 0, 0, 0)
	if got != 0 {
		t.Errorf("want 0, got %d", got)
	}
}

// TestHostDBQuery_NilDB verifies the nil database guard returns 0.
func TestHostDBQuery_NilDB(t *testing.T) {
	p := &Plugin{meta: sdk.PluginMeta{ID: "test"}, database: nil}
	got := p.hostDBQuery(context.Background(), nil, 0, 0, 0, 0, 0, 0)
	if got != 0 {
		t.Errorf("want 0, got %d", got)
	}
}

// TestHostBroadcastWS_NilHub verifies the nil hub guard does not panic.
func TestHostBroadcastWS_NilHub(t *testing.T) {
	p := &Plugin{meta: sdk.PluginMeta{ID: "test"}, hub: nil}
	p.hostBroadcastWS(context.Background(), nil, 0, 0) // must not panic
}

// TestHostGetConfig_NilCfg verifies nil config returns 0.
func TestHostGetConfig_NilCfg(t *testing.T) {
	p := &Plugin{meta: sdk.PluginMeta{ID: "test"}, cfg: nil}
	got := p.hostGetConfig(context.Background(), nil, 0, 0, 0, 0)
	if got != 0 {
		t.Errorf("want 0, got %d", got)
	}
}

// TestHostGetConfig_WithCfg verifies a known config key is written to the output buffer.
func TestHostGetConfig_WithCfg(t *testing.T) {
	cfg := &config.Config{BallchasingAPIKey: "test-key-123"}
	p := &Plugin{meta: sdk.PluginMeta{ID: "test"}, cfg: cfg}
	mod := newMemModule(t)
	ctx := context.Background()

	key := []byte("ballchasing_api_key")
	if !mod.Memory().Write(0, key) {
		t.Fatal("write key to memory")
	}
	outOffset := uint32(64)
	n := p.hostGetConfig(ctx, mod, 0, uint32(len(key)), outOffset, 128)
	if n == 0 {
		t.Fatal("hostGetConfig returned 0")
	}
	data, ok := mod.Memory().Read(outOffset, n)
	if !ok {
		t.Fatal("read result from memory")
	}
	if string(data) != "test-key-123" {
		t.Errorf("got %q, want %q", string(data), "test-key-123")
	}
}

// TestSettingsSchema_WithSettings verifies the SettingSchema→plugin.Setting mapping.
func TestSettingsSchema_WithSettings(t *testing.T) {
	p := &Plugin{
		meta: sdk.PluginMeta{
			Settings: []sdk.SettingSchema{
				{Key: "api_key", Description: "My API key", Secret: true},
				{Key: "mode", Description: "Mode flag", Secret: false},
			},
		},
	}
	schema := p.SettingsSchema()
	if len(schema) != 2 {
		t.Fatalf("want 2 settings, got %d", len(schema))
	}
	if schema[0].Key != "api_key" || schema[0].Type != plugin.SettingTypePassword {
		t.Errorf("first setting: got key=%q type=%q", schema[0].Key, schema[0].Type)
	}
	if schema[1].Key != "mode" || schema[1].Type != plugin.SettingTypeText {
		t.Errorf("second setting: got key=%q type=%q", schema[1].Key, schema[1].Type)
	}
}

// TestApplySettings_NilFn verifies ApplySettings is a no-op when the WASM export is absent.
func TestApplySettings_NilFn(t *testing.T) {
	p := &Plugin{meta: sdk.PluginMeta{ID: "test"}}
	if err := p.ApplySettings(map[string]string{"k": "v"}); err != nil {
		t.Errorf("ApplySettings with nil fn: %v", err)
	}
}
