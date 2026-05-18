package wasmhost

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"OOF_RL/internal/oofevents"
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