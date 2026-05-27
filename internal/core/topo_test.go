package core

import (
	"io/fs"
	"net/http"
	"strings"
	"testing"

	"OOF_RL/internal/db"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/plugin"
)

// stubPlugin implements plugin.Plugin with only ID and Requires wired.
type stubPlugin struct {
	plugin.BasePlugin
	id       string
	requires []string
}

func (s *stubPlugin) ID() string         { return s.id }
func (s *stubPlugin) Requires() []string { return s.requires }
func (s *stubPlugin) SettingsSchema() []plugin.Setting            { return nil }
func (s *stubPlugin) ApplySettings(_ map[string]string) error     { return nil }
func (s *stubPlugin) NavTab() plugin.NavTab                       { return plugin.NavTab{} }
func (s *stubPlugin) Routes(_ *http.ServeMux)                     {}
func (s *stubPlugin) Assets() fs.FS                               { return nil }
func (s *stubPlugin) Init(_ oofevents.PluginBus, _ plugin.Registry, _ *db.DB) error { return nil }

func mkp(id string, requires ...string) plugin.Plugin {
	return &stubPlugin{id: id, requires: requires}
}

func ids(plugins []plugin.Plugin) []string {
	out := make([]string, len(plugins))
	for i, p := range plugins {
		out[i] = p.ID()
	}
	return out
}

func TestTopoSort_Empty(t *testing.T) {
	got, err := topoSort(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty, got %v", ids(got))
	}
}

func TestTopoSort_NoDeps(t *testing.T) {
	got, err := topoSort([]plugin.Plugin{mkp("a"), mkp("b"), mkp("c")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 plugins, got %v", ids(got))
	}
}

func TestTopoSort_LinearChain(t *testing.T) {
	// c requires b, b requires a — input is intentionally shuffled
	got, err := topoSort([]plugin.Plugin{mkp("c", "b"), mkp("a"), mkp("b", "a")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	order := ids(got)
	if order[0] != "a" || order[1] != "b" || order[2] != "c" {
		t.Fatalf("wrong order: %v", order)
	}
}

func TestTopoSort_Diamond(t *testing.T) {
	// b and c both require a; d requires b and c
	got, err := topoSort([]plugin.Plugin{
		mkp("d", "b", "c"),
		mkp("c", "a"),
		mkp("b", "a"),
		mkp("a"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	order := ids(got)
	if order[0] != "a" {
		t.Fatalf("a must be first, got %v", order)
	}
	if order[len(order)-1] != "d" {
		t.Fatalf("d must be last, got %v", order)
	}
}

func TestTopoSort_UnknownDep(t *testing.T) {
	_, err := topoSort([]plugin.Plugin{mkp("a", "ghost")})
	if err == nil {
		t.Fatal("expected error for unknown dependency")
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Errorf("error should name the missing plugin, got: %v", err)
	}
}

func TestTopoSort_Cycle(t *testing.T) {
	_, err := topoSort([]plugin.Plugin{mkp("a", "b"), mkp("b", "a")})
	if err == nil {
		t.Fatal("expected error for cycle")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error should mention cycle, got: %v", err)
	}
}

func TestTopoSort_SelfDep(t *testing.T) {
	_, err := topoSort([]plugin.Plugin{mkp("a", "a")})
	if err == nil {
		t.Fatal("expected error for self-dependency")
	}
}