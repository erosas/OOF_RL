package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"OOF_RL/internal/config"
	"OOF_RL/internal/db"
	"OOF_RL/internal/events"
	"OOF_RL/internal/hub"
	"OOF_RL/internal/momentum/timeline"
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

func TestServerExposesMomentumTimelineProvider(t *testing.T) {
	srv, _ := newTimelineTestServer(t)

	if srv.Timeline() == nil {
		t.Fatal("Timeline() returned nil")
	}
	var _ timeline.SnapshotProvider = srv.Timeline()
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

func TestServerTimelineProviderSnapshotReturnsCopy(t *testing.T) {
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
		return len(srv.Timeline().Snapshot().Entries) == 1
	})

	snapshot := srv.Timeline().Snapshot()
	snapshot.Entries[0].PlayerName = "mutated"
	snapshot.Entries = append(snapshot.Entries, timeline.TimelineEntry{PlayerName: "extra"})

	next := srv.Timeline().Snapshot()
	if len(next.Entries) != 1 {
		t.Fatalf("Timeline provider entry count = %d, want 1", len(next.Entries))
	}
	if next.Entries[0].PlayerName != "Alice" {
		t.Fatalf("Timeline provider player name = %q, want Alice", next.Entries[0].PlayerName)
	}
}

func TestMomentumTimelineSnapshotRouteEmptySnapshot(t *testing.T) {
	srv, _ := newTimelineTestServer(t)
	mux := http.NewServeMux()
	srv.Register(mux)

	w := getTimelineSnapshot(t, mux)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
	var snapshot timeline.TimelineSnapshot
	if err := json.Unmarshal(w.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if snapshot.MatchGUID != "" || len(snapshot.Entries) != 0 || snapshot.MatchEnded {
		t.Fatalf("empty snapshot = %+v", snapshot)
	}
}

func TestMomentumTimelineSnapshotRoutePopulatedSnapshot(t *testing.T) {
	srv, _ := newTimelineTestServer(t)
	mux := http.NewServeMux()
	srv.Register(mux)

	srv.DispatchEvent(events.Envelope{
		Event: "StatfeedEvent",
		Data: []byte(`{
			"MatchGuid":"match-1",
			"EventName":"Shot",
			"MainTarget":{"Name":"Alice","PrimaryId":"pid-a","Shortcut":1,"TeamNum":0}
		}`),
	})
	waitForTimelineTest(t, func() bool {
		return len(srv.Timeline().Snapshot().Entries) == 1
	})

	w := getTimelineSnapshot(t, mux)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
	var snapshot timeline.TimelineSnapshot
	if err := json.Unmarshal(w.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if snapshot.MatchGUID != "match-1" || len(snapshot.Entries) != 1 {
		t.Fatalf("populated snapshot = %+v", snapshot)
	}
	entry := snapshot.Entries[0]
	if entry.PlayerName != "Alice" || entry.Action != oofevents.ActionShot || entry.Blue.MomentumInfluence <= 0 {
		t.Fatalf("entry = %+v", entry)
	}
}

func TestMomentumTimelineSnapshotRouteLifecycleState(t *testing.T) {
	srv, bus := newTimelineTestServer(t)
	mux := http.NewServeMux()
	srv.Register(mux)

	bus.PublishAuthoritative(oofevents.NewGameAction("match-1", oofevents.ActionGoal, oofevents.TeamBlue, "pid-a", "Alice"))
	waitForTimelineTest(t, func() bool {
		return len(srv.Timeline().Snapshot().Entries) == 1
	})
	bus.PublishAuthoritative(oofevents.NewMatchEnded("match-1", 0))
	waitForTimelineTest(t, func() bool {
		return srv.Timeline().Snapshot().MatchEnded
	})

	w := getTimelineSnapshot(t, mux)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
	var snapshot timeline.TimelineSnapshot
	if err := json.Unmarshal(w.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !snapshot.MatchEnded || snapshot.EndedReason != "match.ended" || len(snapshot.Entries) != 1 {
		t.Fatalf("ended snapshot = %+v", snapshot)
	}
}

func TestMomentumTimelineSnapshotRouteMethodNotAllowed(t *testing.T) {
	srv, _ := newTimelineTestServer(t)
	mux := http.NewServeMux()
	srv.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/internal/momentum-timeline-snapshot", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", w.Code)
	}
}

func TestMomentumTimelinePreviewRoute(t *testing.T) {
	srv, _ := newTimelineTestServer(t)
	mux := http.NewServeMux()
	srv.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/internal/momentum-timeline-preview", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/html; charset=utf-8", ct)
	}
	body := w.Body.String()
	for _, want := range []string{
		"Momentum Timeline Preview",
		"/internal/momentum-timeline-snapshot",
		"fetch(endpoint",
		"Entries",
		"Signals",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("preview body missing %q", want)
		}
	}
}

func TestMomentumTimelinePreviewRouteMethodNotAllowed(t *testing.T) {
	srv, _ := newTimelineTestServer(t)
	mux := http.NewServeMux()
	srv.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/internal/momentum-timeline-preview", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", w.Code)
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

func getTimelineSnapshot(t *testing.T, mux *http.ServeMux) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/internal/momentum-timeline-snapshot", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}
