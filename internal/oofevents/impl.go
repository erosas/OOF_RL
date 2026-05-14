package oofevents

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
)

const dispatchBuffer = 256

// New returns a ready-to-use Bus. Call Start before publishing events.
func New() Bus {
	return &busImpl{
		ch:   make(chan OOFEvent, dispatchBuffer),
		subs: make(map[string][]*sub),
	}
}

type busImpl struct {
	mu        sync.RWMutex
	ch        chan OOFEvent
	subs      map[string][]*sub
	allSubs   []*sub
	stopped   atomic.Bool
	done      chan struct{}
	startOnce sync.Once
}

func (b *busImpl) Start() error {
	b.startOnce.Do(func() {
		b.done = make(chan struct{})
		go b.dispatch()
	})
	return nil
}

func (b *busImpl) Stop() {
	if b.stopped.CompareAndSwap(false, true) {
		close(b.ch)
		if b.done != nil {
			<-b.done
		}
	}
}

func (b *busImpl) ForPlugin(pluginID string) PluginBus {
	return &pluginBus{inner: b, pluginID: pluginID}
}

// ResetMatch clears per-match corroboration state. No-op until #39 is implemented.
func (b *busImpl) ResetMatch(_ string) {}

func (b *busImpl) publish(e OOFEvent) {
	if b.stopped.Load() {
		return
	}
	select {
	case b.ch <- e:
	default:
		log.Printf("oofevents: dispatch buffer full, dropping %s", e.Type())
	}
}

func (b *busImpl) PublishAuthoritative(e OOFEvent) {
	if e.Certainty() != Authoritative {
		panic(fmt.Sprintf("oofevents: PublishAuthoritative called with certainty %s", e.Certainty()))
	}
	b.publish(e)
}

func (b *busImpl) PublishInferred(e OOFEvent) {
	if e.Certainty() != Inferred {
		panic(fmt.Sprintf("oofevents: PublishInferred called with certainty %s", e.Certainty()))
	}
	b.publish(e)
}

func (b *busImpl) PublishSignal(e OOFEvent) {
	if e.Certainty() != Signal {
		panic(fmt.Sprintf("oofevents: PublishSignal called with certainty %s", e.Certainty()))
	}
	b.publish(e)
}

func (b *busImpl) Subscribe(eventType string, fn func(OOFEvent)) Subscription {
	s := &sub{fn: fn, eventType: eventType}
	b.mu.Lock()
	b.subs[eventType] = append(b.subs[eventType], s)
	b.mu.Unlock()
	return &canceller{s: s}
}

func (b *busImpl) SubscribeAll(fn func(OOFEvent)) Subscription {
	s := &sub{fn: fn}
	b.mu.Lock()
	b.allSubs = append(b.allSubs, s)
	b.mu.Unlock()
	return &canceller{s: s}
}

// SubscribeMinCertainty registers fn for eventType events that are at least as
// certain as min. Certainty ranks Authoritative(0) < Inferred(1) < Signal(2),
// so the filter is e.Certainty() <= min — e.g. min=Inferred passes Authoritative
// and Inferred but drops Signal.
func (b *busImpl) SubscribeMinCertainty(eventType string, min Certainty, fn func(OOFEvent)) Subscription {
	return b.Subscribe(eventType, func(e OOFEvent) {
		if e.Certainty() <= min {
			fn(e)
		}
	})
}

func (b *busImpl) dispatch() {
	defer close(b.done)
	for e := range b.ch {
		b.mu.RLock()
		typed := append([]*sub(nil), b.subs[e.Type()]...)
		all := append([]*sub(nil), b.allSubs...)
		b.mu.RUnlock()

		for _, s := range typed {
			b.call(s, e)
		}
		for _, s := range all {
			b.call(s, e)
		}

		if e.Type() == TypeMatchEnded {
			b.ResetMatch(e.MatchGUID())
		}
	}
}

// call invokes a handler directly on the dispatch goroutine.
// Panics are recovered so one bad handler cannot stop the bus.
// Handlers must not block — slow work should be offloaded to an internal channel.
func (b *busImpl) call(s *sub, e OOFEvent) {
	if s.cancelled.Load() {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("oofevents: handler for %s panicked: %v", e.Type(), r)
		}
	}()
	s.fn(e)
}

type sub struct {
	fn        func(OOFEvent)
	eventType string
	cancelled atomic.Bool
}

type canceller struct {
	s *sub
}

func (c *canceller) Cancel() {
	c.s.cancelled.Store(true)
	// Lazy removal: the dispatcher skips cancelled subs; compaction not yet implemented.
}

type pluginBus struct {
	inner    *busImpl
	pluginID string
}

// stampedEvent wraps any OOFEvent and overrides Source() with the plugin's ID.
// This avoids requiring concrete event types to expose a mutable source field.
type stampedEvent struct {
	OOFEvent
	src Source
}

func (s stampedEvent) Source() Source   { return s.src }
func (s stampedEvent) Unwrap() OOFEvent { return s.OOFEvent }

// Unwrap returns the concrete event beneath any bus-stamped wrapper.
// Handlers that need to type-assert to a specific event struct must call
// this first: oofevents.Unwrap(e).(oofevents.StateUpdatedEvent)
func Unwrap(e OOFEvent) OOFEvent {
	if u, ok := e.(interface{ Unwrap() OOFEvent }); ok {
		return u.Unwrap()
	}
	return e
}

func (pb *pluginBus) stamp(e OOFEvent) OOFEvent {
	return stampedEvent{OOFEvent: e, src: Source{PluginID: pb.pluginID}}
}

func (pb *pluginBus) PublishAuthoritative(e OOFEvent) { pb.inner.PublishAuthoritative(pb.stamp(e)) }
func (pb *pluginBus) PublishInferred(e OOFEvent)      { pb.inner.PublishInferred(pb.stamp(e)) }
func (pb *pluginBus) PublishSignal(e OOFEvent)        { pb.inner.PublishSignal(pb.stamp(e)) }

func (pb *pluginBus) Subscribe(eventType string, fn func(OOFEvent)) Subscription {
	return pb.inner.Subscribe(eventType, fn)
}
func (pb *pluginBus) SubscribeAll(fn func(OOFEvent)) Subscription {
	return pb.inner.SubscribeAll(fn)
}
func (pb *pluginBus) SubscribeMinCertainty(eventType string, min Certainty, fn func(OOFEvent)) Subscription {
	return pb.inner.SubscribeMinCertainty(eventType, min, fn)
}
