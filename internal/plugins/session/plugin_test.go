package session

import (
	"path/filepath"
	"testing"

	"OOF_RL/internal/db"
	"OOF_RL/internal/oofevents"
)

func newTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestSessionInitRunsMigration(t *testing.T) {
	p := &Plugin{}
	if err := p.Init(nil, nil, newTestDB(t)); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if p.store == nil {
		t.Fatal("Init should set store when database is provided")
	}
}

func TestSessionInitWithBusWiresSubscription(t *testing.T) {
	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(bus.Stop)

	p := &Plugin{}
	if err := p.Init(bus.ForPlugin("session"), nil, newTestDB(t)); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if len(p.Subs) != 1 {
		t.Fatalf("Init should register 1 subscription, got %d", len(p.Subs))
	}
}

func TestSessionInitNilBusNoSubscriptions(t *testing.T) {
	p := &Plugin{}
	if err := p.Init(nil, nil, newTestDB(t)); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if len(p.Subs) != 0 {
		t.Fatalf("Init with nil bus should not register subscriptions, got %d", len(p.Subs))
	}
}