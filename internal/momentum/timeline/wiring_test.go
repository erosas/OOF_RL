package timeline

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"OOF_RL/internal/momentum"
	"OOF_RL/internal/oofevents"
)

func TestWiringRoutesGameActionAfterMomentumServiceUpdate(t *testing.T) {
	bus, service, collector, wiring := newStartedWiring(t)
	defer wiring.Stop()
	defer bus.Stop()

	bus.PublishAuthoritative(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice"))
	waitFor(t, func() bool {
		snapshot := collector.Snapshot()
		return len(snapshot.Entries) == 1
	})

	entry := collector.Snapshot().Entries[0]
	serviceSnapshot := service.Snapshot()
	if entry.MomentumSequence != serviceSnapshot.Sequence {
		t.Fatalf("Timeline sampled sequence %d, want service sequence %d", entry.MomentumSequence, serviceSnapshot.Sequence)
	}
	if entry.MomentumSequence != 1 {
		t.Fatalf("Timeline sampled sequence = %d, want 1", entry.MomentumSequence)
	}
	if entry.Blue.MomentumInfluence <= 0 {
		t.Fatalf("Timeline sampled stale blue signal: %+v", entry.Blue)
	}
}

func TestWiringRoutesLifecycleEvents(t *testing.T) {
	bus, service, collector, wiring := newStartedWiring(t)
	defer wiring.Stop()
	defer bus.Stop()

	bus.PublishAuthoritative(oofevents.NewGameAction("match-1", oofevents.ActionGoal, oofevents.TeamBlue, "pid-a", "Alice"))
	waitFor(t, func() bool { return len(collector.Snapshot().Entries) == 1 })

	bus.PublishAuthoritative(oofevents.NewMatchStarted("match-2"))
	waitFor(t, func() bool { return collector.Snapshot().MatchGUID == "match-2" })
	started := collector.Snapshot()
	if len(started.Entries) != 0 || started.MatchEnded {
		t.Fatalf("collector after match.started = %+v", started)
	}
	if service.Status().Reason != "match.started:match-2" {
		t.Fatalf("service after match.started = %+v", service.Status())
	}

	bus.PublishAuthoritative(oofevents.NewGameAction("match-2", oofevents.ActionSave, oofevents.TeamOrange, "pid-b", "Bob"))
	waitFor(t, func() bool { return len(collector.Snapshot().Entries) == 1 })

	bus.PublishInferred(oofevents.NewMatchRestarted("match-3", "match-2"))
	waitFor(t, func() bool { return collector.Snapshot().MatchGUID == "match-3" })
	restarted := collector.Snapshot()
	if len(restarted.Entries) != 0 || restarted.MatchEnded {
		t.Fatalf("collector after match.restarted = %+v", restarted)
	}
	if service.Status().Reason != "match.restarted:match-3" {
		t.Fatalf("service after match.restarted = %+v", service.Status())
	}

	bus.PublishAuthoritative(oofevents.NewGameAction("match-3", oofevents.ActionGoal, oofevents.TeamBlue, "pid-a", "Alice"))
	waitFor(t, func() bool { return len(collector.Snapshot().Entries) == 1 })

	bus.PublishAuthoritative(oofevents.NewMatchEnded("match-3", 0))
	waitFor(t, func() bool { return collector.Snapshot().MatchEnded })
	ended := collector.Snapshot()
	if ended.MatchGUID != "match-3" || ended.EndedReason != "match.ended" || len(ended.Entries) != 1 {
		t.Fatalf("collector after match.ended = %+v", ended)
	}
	if service.Status().Active {
		t.Fatalf("service should be inactive after match.ended: %+v", service.Status())
	}

	bus.PublishAuthoritative(oofevents.NewGameAction("match-3", oofevents.ActionShot, oofevents.TeamOrange, "pid-c", "Cora"))
	time.Sleep(20 * time.Millisecond)
	if got := len(collector.Snapshot().Entries); got != 1 {
		t.Fatalf("collector recorded after match.ended, entries = %d", got)
	}

	bus.PublishAuthoritative(oofevents.NewMatchDestroyed())
	waitFor(t, func() bool {
		snapshot := collector.Snapshot()
		return snapshot.MatchGUID == "" && len(snapshot.Entries) == 0 && !snapshot.MatchEnded
	})
	if service.Status().Reason != "match.destroyed" {
		t.Fatalf("service after match.destroyed = %+v", service.Status())
	}
}

func TestWiringIgnoresWrongPayloadTypes(t *testing.T) {
	service := momentum.NewService(momentum.Config{Decay: 1})
	collector := NewCollector(service, Config{})
	wiring := &Wiring{service: service, collector: collector}

	wiring.handleGameAction(oofevents.NewMatchStarted("match-1"))
	wiring.handleMatchStarted(oofevents.NewMatchEnded("match-1", 0))
	wiring.handleMatchRestarted(oofevents.NewMatchDestroyed())
	wiring.handleMatchEnded(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice"))
	wiring.handleMatchDestroyed(oofevents.NewMatchStarted("match-1"))

	if service.Snapshot().Sequence != 0 {
		t.Fatalf("bad payload changed service state: %+v", service.Snapshot())
	}
	if len(collector.Snapshot().Entries) != 0 {
		t.Fatalf("bad payload changed collector state: %+v", collector.Snapshot())
	}
}

func TestWiringStopCancelsSubscriptions(t *testing.T) {
	bus, service, collector, wiring := newStartedWiring(t)
	defer bus.Stop()

	wiring.Stop()
	wiring.Stop()

	bus.PublishAuthoritative(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice"))
	time.Sleep(20 * time.Millisecond)
	if service.Snapshot().Sequence != 0 {
		t.Fatalf("stopped wiring routed service event: %+v", service.Snapshot())
	}
	if len(collector.Snapshot().Entries) != 0 {
		t.Fatalf("stopped wiring routed collector event: %+v", collector.Snapshot())
	}
}

func TestWiringDoesNotPublishOutputEvents(t *testing.T) {
	bus, _, collector, wiring := newStartedWiring(t)
	defer wiring.Stop()
	defer bus.Stop()

	var (
		mu     sync.Mutex
		seen   int
		allSub = bus.SubscribeAll(func(oofevents.OOFEvent) {
			mu.Lock()
			seen++
			mu.Unlock()
		})
	)
	defer allSub.Cancel()

	bus.PublishAuthoritative(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice"))
	waitFor(t, func() bool { return len(collector.Snapshot().Entries) == 1 })

	mu.Lock()
	defer mu.Unlock()
	if seen != 1 {
		t.Fatalf("seen events = %d, want only input event", seen)
	}
}

func TestWiringHasNoOverlayHUDDependency(t *testing.T) {
	for _, path := range []string{"collector.go", "wiring.go"} {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", path, err)
		}
		if strings.Contains(string(source), "internal/plugins/overlayhud") {
			t.Fatalf("%s must not import Overlay HUD", path)
		}
	}
}

func newStartedWiring(t *testing.T) (oofevents.Bus, *momentum.Service, *Collector, *Wiring) {
	t.Helper()

	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	service := momentum.NewService(momentum.Config{Decay: 1})
	collector := NewCollector(service, Config{})
	wiring := NewWiring(bus, service, collector)
	return bus, service, collector, wiring
}

func waitFor(t *testing.T, fn func() bool) {
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
