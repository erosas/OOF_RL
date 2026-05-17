package debugassistant

import (
	"testing"

	"OOF_RL/internal/config"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/plugin"
)

type testReg struct {
	plugin.Registry
	cfg *config.Config
}

func (r *testReg) Config() *config.Config { return r.cfg }

func newBus(t *testing.T) oofevents.Bus {
	t.Helper()
	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(bus.Stop)
	return bus
}

func TestInitSetsConfigFromRegistry(t *testing.T) {
	cfg := config.Defaults()
	p := &Plugin{}
	if err := p.Init(nil, &testReg{cfg: &cfg}, nil); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if p.Cfg == nil {
		t.Fatal("Init should set Cfg from registry")
	}
}

func TestInitWiresSubscriptionsWhenEnabled(t *testing.T) {
	bus := newBus(t)
	cfg := config.Defaults()

	p := &Plugin{}
	if err := p.Init(bus.ForPlugin("debugassistant"), &testReg{cfg: &cfg}, nil); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if len(p.Subs) != 7 {
		t.Fatalf("Init should register 7 subscriptions when enabled, got %d", len(p.Subs))
	}
}

func TestInitSkipsSubscriptionsWhenDisabled(t *testing.T) {
	bus := newBus(t)
	cfg := config.Defaults()
	cfg.DisabledPlugins = []string{"debugassistant"}

	p := &Plugin{}
	if err := p.Init(bus.ForPlugin("debugassistant"), &testReg{cfg: &cfg}, nil); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if len(p.Subs) != 0 {
		t.Fatalf("disabled plugin should register no subscriptions, got %d", len(p.Subs))
	}
}

func TestInitNilRegAndNilBus(t *testing.T) {
	p := &Plugin{}
	if err := p.Init(nil, nil, nil); err != nil {
		t.Fatalf("Init with nil args should not error: %v", err)
	}
}

func TestNavTabHiddenWhenDisabled(t *testing.T) {
	cfg := config.Defaults()
	cfg.DisabledPlugins = []string{"debugassistant"}
	p := &Plugin{}
	_ = p.Init(nil, &testReg{cfg: &cfg}, nil)

	tab := p.NavTab()
	if tab.ID != "" {
		t.Fatalf("disabled plugin NavTab should be empty, got %+v", tab)
	}
}