package oofevents_test

import (
	"sync"
	"testing"

	"OOF_RL/internal/oofevents"
)

func TestStopBeforeStartDoesNotPanic(t *testing.T) {
	b := oofevents.New()
	b.Stop() // Start was never called — must not panic
}

func TestHandlerPanicDoesNotStopBus(t *testing.T) {
	b := startedBus(t)

	sub1 := b.Subscribe("test.panic", func(e oofevents.OOFEvent) { panic("boom") })
	defer sub1.Cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	sub2 := b.Subscribe("test.panic", func(e oofevents.OOFEvent) { wg.Done() })
	defer sub2.Cancel()

	b.PublishAuthoritative(makeEvent("test.panic", oofevents.Authoritative, "x"))
	wg.Wait()
}

func TestPublishInferred(t *testing.T) {
	b := startedBus(t)
	var wg sync.WaitGroup
	wg.Add(1)
	sub := b.Subscribe("match.restarted", func(e oofevents.OOFEvent) {
		if e.Certainty() != oofevents.Inferred {
			panic("expected Inferred certainty")
		}
		wg.Done()
	})
	defer sub.Cancel()
	b.PublishInferred(makeEvent("match.restarted", oofevents.Inferred, "g1"))
	wg.Wait()
}

func TestPublishInferredWrongCertaintyPanics(t *testing.T) {
	b := startedBus(t)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for wrong certainty on PublishInferred")
		}
	}()
	b.PublishInferred(makeEvent("test", oofevents.Authoritative, "x"))
}

func TestPublishSignalWrongCertaintyPanics(t *testing.T) {
	b := startedBus(t)
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for wrong certainty on PublishSignal")
		}
	}()
	b.PublishSignal(makeEvent("test", oofevents.Authoritative, "x"))
}

func TestPluginBusPublishInferred(t *testing.T) {
	b := startedBus(t)
	pb := b.ForPlugin("my-plugin")
	var wg sync.WaitGroup
	wg.Add(1)
	sub := b.Subscribe("match.restarted", func(e oofevents.OOFEvent) { wg.Done() })
	defer sub.Cancel()
	pb.PublishInferred(makeEvent("match.restarted", oofevents.Inferred, "g1"))
	wg.Wait()
}

func TestPluginBusSubscribe(t *testing.T) {
	b := startedBus(t)
	pb := b.ForPlugin("my-plugin")
	var wg sync.WaitGroup
	wg.Add(1)
	sub := pb.Subscribe("goal.scored", func(e oofevents.OOFEvent) { wg.Done() })
	defer sub.Cancel()
	b.PublishAuthoritative(makeEvent("goal.scored", oofevents.Authoritative, "g1"))
	wg.Wait()
}

func TestPluginBusSubscribeAll(t *testing.T) {
	b := startedBus(t)
	pb := b.ForPlugin("my-plugin")
	var wg sync.WaitGroup
	wg.Add(1)
	sub := pb.SubscribeAll(func(e oofevents.OOFEvent) { wg.Done() })
	defer sub.Cancel()
	b.PublishAuthoritative(makeEvent("any.event", oofevents.Authoritative, "g1"))
	wg.Wait()
}

func TestPluginBusSubscribeMinCertainty(t *testing.T) {
	b := startedBus(t)
	pb := b.ForPlugin("my-plugin")
	var wg sync.WaitGroup
	wg.Add(1)
	sub := pb.SubscribeMinCertainty("goal.scored", oofevents.Inferred, func(e oofevents.OOFEvent) { wg.Done() })
	defer sub.Cancel()
	b.PublishAuthoritative(makeEvent("goal.scored", oofevents.Authoritative, "g1"))
	wg.Wait()
}