package plugin_test

import (
	"io/fs"
	"net/http"
	"testing"

	"OOF_RL/internal/db"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/plugin"
)

// stubSub records whether Cancel was called.
type stubSub struct{ cancelled bool }

func (s *stubSub) Cancel() { s.cancelled = true }

// stubPlugin is a minimal plugin.Plugin implementation for registry tests.
type stubPlugin struct{ plugin.BasePlugin }

func (s *stubPlugin) ID() string                                               { return "stub" }
func (s *stubPlugin) Requires() []string                                       { return nil }
func (s *stubPlugin) NavTab() plugin.NavTab                                    { return plugin.NavTab{} }
func (s *stubPlugin) Routes(_ *http.ServeMux)                                  {}
func (s *stubPlugin) Assets() fs.FS                                            { return nil }
func (s *stubPlugin) Init(_ oofevents.PluginBus, _ plugin.Registry, _ *db.DB) error { return nil }
func (s *stubPlugin) SettingsSchema() []plugin.Setting                         { return nil }
func (s *stubPlugin) ApplySettings(_ map[string]string) error                  { return nil }
func (s *stubPlugin) DeclaredEvents() []oofevents.EventDeclaration             { return nil }

func TestRegisterAndFactories(t *testing.T) {
	const id = "test-register-unique-abc123"
	plugin.Register(id, func() plugin.Plugin { return &stubPlugin{} })

	factories := plugin.Factories()
	f, ok := factories[id]
	if !ok {
		t.Fatalf("Factories() missing registered id %q", id)
	}
	if p := f(); p == nil {
		t.Fatal("factory returned nil")
	}
}

func TestFactoriesReturnsCopy(t *testing.T) {
	const id = "test-factories-copy-xyz789"
	plugin.Register(id, func() plugin.Plugin { return &stubPlugin{} })

	f1 := plugin.Factories()
	delete(f1, id)

	f2 := plugin.Factories()
	if _, ok := f2[id]; !ok {
		t.Fatal("deleting from returned map should not remove entry from registry")
	}
}

func TestBasePluginShutdownCancelsSubscriptions(t *testing.T) {
	var p plugin.BasePlugin
	s1 := &stubSub{}
	s2 := &stubSub{}
	p.AddSub(s1)
	p.AddSub(s2)

	if err := p.Shutdown(); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if !s1.cancelled || !s2.cancelled {
		t.Fatal("Shutdown should cancel all subscriptions")
	}
	if len(p.Subs) != 0 {
		t.Fatal("Shutdown should clear Subs slice")
	}
}

func TestBasePluginShutdownIdempotent(t *testing.T) {
	var p plugin.BasePlugin
	s := &stubSub{}
	p.AddSub(s)
	_ = p.Shutdown()

	if err := p.Shutdown(); err != nil {
		t.Fatal("second Shutdown should not error")
	}
}

func TestBasePluginAddSub(t *testing.T) {
	var p plugin.BasePlugin
	p.AddSub(&stubSub{})
	p.AddSub(&stubSub{})
	if len(p.Subs) != 2 {
		t.Fatalf("AddSub: want 2 subs, got %d", len(p.Subs))
	}
}

func TestBasePluginDeclaredEventsNil(t *testing.T) {
	var p plugin.BasePlugin
	if p.DeclaredEvents() != nil {
		t.Fatal("DeclaredEvents should return nil")
	}
}