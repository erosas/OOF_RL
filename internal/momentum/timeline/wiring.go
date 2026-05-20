package timeline

import (
	"sync"

	"OOF_RL/internal/momentum"
	"OOF_RL/internal/oofevents"
)

// Wiring coordinates typed events for Momentum and the Timeline collector.
// Game actions update the Momentum service before the collector samples it.
type Wiring struct {
	service   *momentum.Service
	collector *Collector

	mu      sync.Mutex
	subs    []oofevents.Subscription
	stopped bool
}

// NewWiring subscribes to typed game action and match lifecycle events.
func NewWiring(bus oofevents.PluginBus, service *momentum.Service, collector *Collector) *Wiring {
	w := &Wiring{
		service:   service,
		collector: collector,
	}
	w.subs = []oofevents.Subscription{
		bus.Subscribe(oofevents.TypeGameAction, w.handleGameAction),
		bus.Subscribe(oofevents.TypeMatchStarted, w.handleMatchStarted),
		bus.Subscribe(oofevents.TypeMatchRestarted, w.handleMatchRestarted),
		bus.Subscribe(oofevents.TypeMatchEnded, w.handleMatchEnded),
		bus.Subscribe(oofevents.TypeMatchDestroyed, w.handleMatchDestroyed),
	}
	return w
}

// Stop cancels all subscriptions. It is safe to call more than once.
func (w *Wiring) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return
	}
	for _, sub := range w.subs {
		sub.Cancel()
	}
	w.stopped = true
}

func (w *Wiring) handleGameAction(event oofevents.OOFEvent) {
	gameAction, ok := oofevents.Unwrap(event).(oofevents.GameActionEvent)
	if !ok {
		return
	}
	if w.service != nil {
		w.service.HandleGameAction(gameAction)
	}
	if w.collector != nil {
		w.collector.HandleGameAction(gameAction)
	}
}

func (w *Wiring) handleMatchStarted(event oofevents.OOFEvent) {
	matchStarted, ok := oofevents.Unwrap(event).(oofevents.MatchStartedEvent)
	if !ok {
		return
	}
	if w.service != nil {
		w.service.HandleMatchStarted(matchStarted)
	}
	if w.collector != nil {
		w.collector.Reset(matchStarted.MatchGUID())
	}
}

func (w *Wiring) handleMatchRestarted(event oofevents.OOFEvent) {
	matchRestarted, ok := oofevents.Unwrap(event).(oofevents.MatchRestartedEvent)
	if !ok {
		return
	}
	if w.service != nil {
		w.service.HandleMatchRestarted(matchRestarted)
	}
	if w.collector != nil {
		w.collector.Reset(matchRestarted.MatchGUID())
	}
}

func (w *Wiring) handleMatchEnded(event oofevents.OOFEvent) {
	matchEnded, ok := oofevents.Unwrap(event).(oofevents.MatchEndedEvent)
	if !ok {
		return
	}
	if w.service != nil {
		w.service.HandleMatchEnded(matchEnded)
	}
	if w.collector != nil {
		w.collector.MarkMatchEnded(matchEnded.MatchGUID())
	}
}

func (w *Wiring) handleMatchDestroyed(event oofevents.OOFEvent) {
	matchDestroyed, ok := oofevents.Unwrap(event).(oofevents.MatchDestroyedEvent)
	if !ok {
		return
	}
	if w.service != nil {
		w.service.HandleMatchDestroyed(matchDestroyed)
	}
	if w.collector != nil {
		w.collector.Clear()
	}
}
