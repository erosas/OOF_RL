package ranks

import (
	"testing"

	"OOF_RL/internal/oofevents"
)

func TestRanksInitWiresSubscriptions(t *testing.T) {
	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(bus.Stop)

	p := &Plugin{}
	if err := p.Init(bus.ForPlugin("ranks"), nil, nil); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if len(p.Subs) != 2 {
		t.Fatalf("Init should register 2 subscriptions, got %d", len(p.Subs))
	}
}

func TestRanksShutdownCancelsSubscriptions(t *testing.T) {
	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(bus.Stop)

	p := &Plugin{}
	_ = p.Init(bus.ForPlugin("ranks"), nil, nil)

	if err := p.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if len(p.Subs) != 0 {
		t.Fatal("Shutdown should clear subscriptions")
	}
}