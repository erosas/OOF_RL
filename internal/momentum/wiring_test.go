package momentum

import (
	"sync"
	"testing"
	"time"

	"OOF_RL/internal/oofevents"
)

func TestWiringRoutesGameActionEvents(t *testing.T) {
	bus, service, wiring := newStartedWiring(t)
	defer wiring.Stop()
	defer bus.Stop()

	bus.PublishAuthoritative(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice"))
	waitFor(t, func() bool {
		return service.Snapshot().Sequence == 1
	})

	snapshot := service.Snapshot()
	if snapshot.Teams[oofevents.TeamBlue].MomentumInfluence <= 0 {
		t.Fatalf("blue signal not updated: %+v", snapshot.Teams[oofevents.TeamBlue])
	}
}

func TestWiringRoutesLifecycleEvents(t *testing.T) {
	bus, service, wiring := newStartedWiring(t)
	defer wiring.Stop()
	defer bus.Stop()

	bus.PublishAuthoritative(oofevents.NewGameAction("match-1", oofevents.ActionGoal, oofevents.TeamBlue, "pid-a", "Alice"))
	waitFor(t, func() bool { return service.Snapshot().Sequence == 1 })

	bus.PublishAuthoritative(oofevents.NewMatchStarted("match-2"))
	waitFor(t, func() bool { return service.Snapshot().Sequence == 0 })
	if service.Status().Reason != "match.started:match-2" {
		t.Fatalf("status after match.started = %+v", service.Status())
	}

	bus.PublishAuthoritative(oofevents.NewGameAction("match-2", oofevents.ActionGoal, oofevents.TeamOrange, "pid-b", "Bob"))
	waitFor(t, func() bool { return service.Snapshot().Sequence == 1 })

	bus.PublishInferred(oofevents.NewMatchRestarted("match-3", "match-2"))
	waitFor(t, func() bool { return service.Status().Reason == "match.restarted:match-3" })
	if service.Snapshot().Sequence != 0 {
		t.Fatalf("sequence after match.restarted = %d, want 0", service.Snapshot().Sequence)
	}

	bus.PublishAuthoritative(oofevents.NewGameAction("match-3", oofevents.ActionGoal, oofevents.TeamBlue, "pid-a", "Alice"))
	waitFor(t, func() bool { return service.Snapshot().Sequence == 1 })

	bus.PublishAuthoritative(oofevents.NewMatchEnded("match-3", 0))
	waitFor(t, func() bool { return !service.Status().Active })

	frozen := service.Snapshot()
	bus.PublishAuthoritative(oofevents.NewGameAction("match-3", oofevents.ActionGoal, oofevents.TeamOrange, "pid-b", "Bob"))
	time.Sleep(20 * time.Millisecond)
	if service.Snapshot().Sequence != frozen.Sequence {
		t.Fatalf("inactive service accepted game action after match.ended")
	}

	bus.PublishAuthoritative(oofevents.NewMatchDestroyed())
	waitFor(t, func() bool { return service.Status().Reason == "match.destroyed" })
	if service.Snapshot().Sequence != 0 {
		t.Fatalf("sequence after match.destroyed = %d, want 0", service.Snapshot().Sequence)
	}
}

func TestWiringIgnoresWrongPayloadTypes(t *testing.T) {
	service := NewService(Config{Decay: 1})
	wiring := &Wiring{service: service}

	wiring.handleGameAction(oofevents.NewMatchStarted("match-1"))
	wiring.handleMatchStarted(oofevents.NewMatchEnded("match-1", 0))
	wiring.handleMatchRestarted(oofevents.NewMatchDestroyed())
	wiring.handleMatchEnded(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice"))
	wiring.handleMatchDestroyed(oofevents.NewMatchStarted("match-1"))

	if service.Snapshot().Sequence != 0 {
		t.Fatalf("bad payload changed service state: %+v", service.Snapshot())
	}
}

func TestWiringStopCancelsSubscriptions(t *testing.T) {
	bus, service, wiring := newStartedWiring(t)
	defer bus.Stop()

	wiring.Stop()
	wiring.Stop()

	bus.PublishAuthoritative(oofevents.NewGameAction("match-1", oofevents.ActionShot, oofevents.TeamBlue, "pid-a", "Alice"))
	time.Sleep(20 * time.Millisecond)
	if service.Snapshot().Sequence != 0 {
		t.Fatalf("stopped wiring still routed event: %+v", service.Snapshot())
	}
}

func TestWiringDoesNotPublishOutputEvents(t *testing.T) {
	bus, service, wiring := newStartedWiring(t)
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
	waitFor(t, func() bool { return service.Snapshot().Sequence == 1 })

	mu.Lock()
	defer mu.Unlock()
	if seen != 1 {
		t.Fatalf("seen events = %d, want only input event", seen)
	}
}

func newStartedWiring(t *testing.T) (oofevents.Bus, *Service, *Wiring) {
	t.Helper()

	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	service := NewService(Config{Decay: 1})
	wiring := NewWiring(bus, service)
	return bus, service, wiring
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
