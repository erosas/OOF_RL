package ballchasing

import (
	"path/filepath"
	"testing"
	"time"

	"OOF_RL/internal/config"
	"OOF_RL/internal/db"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/plugin"
)

type testReg struct {
	plugin.Registry
	cfg *config.Config
}

func (r *testReg) Config() *config.Config { return r.cfg }

func newTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestBallchasingInitRunsMigration(t *testing.T) {
	cfg := config.Defaults()
	p := &Plugin{startupTime: time.Now()}
	if err := p.Init(nil, &testReg{cfg: &cfg}, newTestDB(t)); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if p.store == nil {
		t.Fatal("Init should create store when database is provided")
	}
}

func TestBallchasingInitSetsConfig(t *testing.T) {
	cfg := config.Defaults()
	cfg.BallchasingAPIKey = "test-key"
	p := &Plugin{startupTime: time.Now()}
	if err := p.Init(nil, &testReg{cfg: &cfg}, newTestDB(t)); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if p.Cfg == nil || p.Cfg.BallchasingAPIKey != "test-key" {
		t.Fatalf("Init should set Cfg from registry, got %v", p.Cfg)
	}
}

func TestBallchasingInitWithBusWiresSubscriptions(t *testing.T) {
	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(bus.Stop)

	cfg := config.Defaults()
	p := &Plugin{startupTime: time.Now()}
	if err := p.Init(bus.ForPlugin("ballchasing"), &testReg{cfg: &cfg}, newTestDB(t)); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if len(p.Subs) != 2 {
		t.Fatalf("Init should register 2 subscriptions, got %d", len(p.Subs))
	}
}

func TestBallchasingInitNilArgs(t *testing.T) {
	p := &Plugin{startupTime: time.Now()}
	if err := p.Init(nil, nil, nil); err != nil {
		t.Fatalf("Init with nil args should not error: %v", err)
	}
}