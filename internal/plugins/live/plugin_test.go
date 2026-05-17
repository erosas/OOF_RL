package live

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"OOF_RL/internal/db"
	"OOF_RL/internal/oofevents"
)

func newTestPlugin(t *testing.T) *Plugin {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(bus.Stop)
	p := &Plugin{}
	if err := p.Init(bus.ForPlugin("live"), nil, database); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return p
}

func TestLiveInitWiresSubscriptions(t *testing.T) {
	p := newTestPlugin(t)
	if len(p.Subs) != 2 {
		t.Fatalf("Init should register 2 subscriptions, got %d", len(p.Subs))
	}
}

func TestLiveHandleStateNoActiveMatch(t *testing.T) {
	p := newTestPlugin(t)

	w := httptest.NewRecorder()
	p.handleState(w, httptest.NewRequest("GET", "/api/live/state", nil))

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if resp["active"] != false {
		t.Fatalf("expected active:false when no match, got %v", resp)
	}
}

func TestLiveHandleStateWithActiveMatch(t *testing.T) {
	p := newTestPlugin(t)
	ev := oofevents.NewStateUpdated("guid-1", []oofevents.PlayerSnapshot{
		{Name: "Alice", PrimaryID: "pid-a", TeamNum: 0},
	}, oofevents.GameSnapshot{})
	p.onStateUpdated(ev)

	w := httptest.NewRecorder()
	p.handleState(w, httptest.NewRequest("GET", "/api/live/state", nil))

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if resp["active"] != true {
		t.Fatalf("expected active:true after state update, got %v", resp)
	}
}

func TestLiveOnMatchDestroyedClearsState(t *testing.T) {
	p := newTestPlugin(t)
	p.onStateUpdated(oofevents.NewStateUpdated("guid-1", nil, oofevents.GameSnapshot{}))
	p.onMatchDestroyed(oofevents.NewMatchDestroyed())

	p.mu.RLock()
	state := p.state
	p.mu.RUnlock()
	if state != nil {
		t.Fatal("onMatchDestroyed should clear state")
	}
}