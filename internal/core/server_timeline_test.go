package core

import (
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"OOF_RL/internal/config"
	"OOF_RL/internal/db"
	"OOF_RL/internal/events"
	"OOF_RL/internal/hub"
	"OOF_RL/internal/oofevents"
)

func newTimelineTestServer(t *testing.T) (*Server, oofevents.Bus) {
	t.Helper()

	tmpDir := t.TempDir()
	database, err := db.Open(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	cfg := config.Defaults()
	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(bus.Stop)

	srv := NewServer(filepath.Join(tmpDir, "config.toml"), &cfg, database, hub.New(), http.NotFoundHandler(), func() {}, nil, bus)
	return srv, bus
}

func TestServerRegistersMomentumTimelineRuntime(t *testing.T) {
	srv, _ := newTimelineTestServer(t)

	if srv.timeline == nil {
		t.Fatal("server timeline collector is nil")
	}
	if srv.timelineW == nil {
		t.Fatal("server timeline wiring is nil")
	}
}

func TestServerTimelineRuntimeReceivesTranslatedGameActionsAfterMomentum(t *testing.T) {
	srv, _ := newTimelineTestServer(t)

	srv.DispatchEvent(events.Envelope{
		Event: "StatfeedEvent",
		Data: []byte(`{
			"MatchGuid":"match-1",
			"EventName":"Shot",
			"MainTarget":{"Name":"Alice","PrimaryId":"pid-a","Shortcut":1,"TeamNum":0}
		}`),
	})

	waitForTimelineTest(t, func() bool {
		return len(srv.timeline.Snapshot().Entries) == 1
	})

	entry := srv.timeline.Snapshot().Entries[0]
	momentumSnapshot := srv.Momentum().Snapshot()
	if entry.MomentumSequence != momentumSnapshot.Sequence {
		t.Fatalf("Timeline sampled sequence %d, want Momentum sequence %d", entry.MomentumSequence, momentumSnapshot.Sequence)
	}
	if entry.Blue.MomentumInfluence <= 0 {
		t.Fatalf("Timeline sampled stale Momentum state: %+v", entry)
	}
}

func TestServerTimelineRuntimeRoutesLifecycle(t *testing.T) {
	srv, bus := newTimelineTestServer(t)

	bus.PublishAuthoritative(oofevents.NewGameAction("match-1", oofevents.ActionGoal, oofevents.TeamBlue, "pid-a", "Alice"))
	waitForTimelineTest(t, func() bool {
		return len(srv.timeline.Snapshot().Entries) == 1
	})

	bus.PublishAuthoritative(oofevents.NewMatchStarted("match-2"))
	waitForTimelineTest(t, func() bool {
		snapshot := srv.timeline.Snapshot()
		return snapshot.MatchGUID == "match-2" && len(snapshot.Entries) == 0
	})

	bus.PublishAuthoritative(oofevents.NewGameAction("match-2", oofevents.ActionShot, oofevents.TeamOrange, "pid-b", "Bob"))
	waitForTimelineTest(t, func() bool {
		return len(srv.timeline.Snapshot().Entries) == 1
	})

	bus.PublishAuthoritative(oofevents.NewMatchEnded("match-2", 1))
	waitForTimelineTest(t, func() bool {
		return srv.timeline.Snapshot().MatchEnded
	})

	bus.PublishAuthoritative(oofevents.NewMatchDestroyed())
	waitForTimelineTest(t, func() bool {
		snapshot := srv.timeline.Snapshot()
		return snapshot.MatchGUID == "" && len(snapshot.Entries) == 0 && !snapshot.MatchEnded
	})
}

func TestServerShutdownRuntimeStopsTimelineWiring(t *testing.T) {
	srv, _ := newTimelineTestServer(t)

	srv.ShutdownRuntime()
	srv.DispatchEvent(events.Envelope{
		Event: "StatfeedEvent",
		Data: []byte(`{
			"MatchGuid":"match-1",
			"EventName":"Shot",
			"MainTarget":{"Name":"Alice","PrimaryId":"pid-a","Shortcut":1,"TeamNum":0}
		}`),
	})
	time.Sleep(20 * time.Millisecond)

	if srv.Momentum().Snapshot().Sequence != 0 {
		t.Fatalf("Momentum updated after runtime shutdown: %+v", srv.Momentum().Snapshot())
	}
	if len(srv.timeline.Snapshot().Entries) != 0 {
		t.Fatalf("Timeline updated after runtime shutdown: %+v", srv.timeline.Snapshot())
	}
}

func waitForTimelineTest(t *testing.T, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}
