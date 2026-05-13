package oofevents_test

import (
	"sync"
	"testing"
	"time"

	"OOF_RL/internal/oofevents"
)


// testEvent is a minimal concrete event for testing.
type testEvent struct {
	oofevents.Base
}

func makeEvent(typ string, cert oofevents.Certainty, guid string) testEvent {
	return testEvent{oofevents.NewBase(typ, cert, guid)}
}

func startedBus(t *testing.T) oofevents.Bus {
	t.Helper()
	b := oofevents.New()
	if err := b.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(b.Stop)
	return b
}

func TestSubscribeReceivesEvent(t *testing.T) {
	b := startedBus(t)

	var got oofevents.OOFEvent
	var wg sync.WaitGroup
	wg.Add(1)
	sub := b.Subscribe("goal.scored", func(e oofevents.OOFEvent) {
		got = e
		wg.Done()
	})
	defer sub.Cancel()

	b.PublishAuthoritative(makeEvent("goal.scored", oofevents.Authoritative, "abc"))
	wg.Wait()

	if got.Type() != "goal.scored" {
		t.Fatalf("got type %q, want %q", got.Type(), "goal.scored")
	}
}

func TestSubscribeAllReceivesEveryEvent(t *testing.T) {
	b := startedBus(t)

	var mu sync.Mutex
	var received []string
	var wg sync.WaitGroup
	wg.Add(2)

	sub := b.SubscribeAll(func(e oofevents.OOFEvent) {
		mu.Lock()
		received = append(received, e.Type())
		mu.Unlock()
		wg.Done()
	})
	defer sub.Cancel()

	b.PublishAuthoritative(makeEvent("match.started", oofevents.Authoritative, "g1"))
	b.PublishAuthoritative(makeEvent("goal.scored", oofevents.Authoritative, "g1"))
	wg.Wait()

	if len(received) != 2 {
		t.Fatalf("got %d events, want 2: %v", len(received), received)
	}
}

func TestCancelledSubscriptionNotCalled(t *testing.T) {
	b := startedBus(t)

	called := false
	sub := b.Subscribe("goal.scored", func(e oofevents.OOFEvent) { called = true })
	sub.Cancel()

	// Publish and wait long enough for dispatch
	b.PublishAuthoritative(makeEvent("goal.scored", oofevents.Authoritative, "x"))
	time.Sleep(50 * time.Millisecond)

	if called {
		t.Fatal("handler called after Cancel")
	}
}

func TestWrongCertaintyPanics(t *testing.T) {
	b := startedBus(t)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for wrong certainty")
		}
	}()
	// Signal event published via PublishAuthoritative — should panic.
	b.PublishAuthoritative(makeEvent("forfeit.signal", oofevents.Signal, "x"))
}

func TestForPluginStampsSource(t *testing.T) {
	b := startedBus(t)
	pb := b.ForPlugin("forfeit-detector")

	var got oofevents.OOFEvent
	var wg sync.WaitGroup
	wg.Add(1)
	sub := b.SubscribeAll(func(e oofevents.OOFEvent) {
		got = e
		wg.Done()
	})
	defer sub.Cancel()

	pb.PublishSignal(makeEvent("forfeit.signal", oofevents.Signal, "m1"))
	wg.Wait()

	if got.Source().PluginID != "forfeit-detector" {
		t.Fatalf("got PluginID %q, want %q", got.Source().PluginID, "forfeit-detector")
	}
}

func TestRLTranslatorHasEmptyPluginID(t *testing.T) {
	b := startedBus(t)
	rl := b.ForPlugin("") // RL translator convention

	var got oofevents.OOFEvent
	var wg sync.WaitGroup
	wg.Add(1)
	sub := b.SubscribeAll(func(e oofevents.OOFEvent) {
		got = e
		wg.Done()
	})
	defer sub.Cancel()

	rl.PublishAuthoritative(makeEvent("match.started", oofevents.Authoritative, "m1"))
	wg.Wait()

	if got.Source().PluginID != "" {
		t.Fatalf("RL translator event should have empty PluginID, got %q", got.Source().PluginID)
	}
}

func TestSubscribeMinCertaintyFilters(t *testing.T) {
	b := startedBus(t)

	var mu sync.Mutex
	var received []oofevents.OOFEvent
	var wg sync.WaitGroup
	wg.Add(1)

	// Only Authoritative and Inferred (min = Inferred)
	sub := b.SubscribeMinCertainty("goal.scored", oofevents.Inferred, func(e oofevents.OOFEvent) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
		wg.Done()
	})
	defer sub.Cancel()

	b.PublishAuthoritative(makeEvent("goal.scored", oofevents.Authoritative, "m1"))
	wg.Wait()

	mu.Lock()
	n := len(received)
	mu.Unlock()
	if n != 1 {
		t.Fatalf("got %d events, want 1", n)
	}
}

func TestMatchEndedResetsCorroboration(t *testing.T) {
	b := startedBus(t)

	// Just verify ResetMatch doesn't panic and match.ended triggers it.
	var wg sync.WaitGroup
	wg.Add(1)
	sub := b.Subscribe("match.ended", func(e oofevents.OOFEvent) { wg.Done() })
	defer sub.Cancel()

	b.PublishAuthoritative(makeEvent("match.ended", oofevents.Authoritative, "m1"))
	wg.Wait()
}