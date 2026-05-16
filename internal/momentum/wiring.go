package momentum

import (
	"sync"

	"OOF_RL/internal/oofevents"
)

// Wiring connects OOF typed events to a Momentum Service. It owns only event
// subscriptions; service ownership stays with the caller.
type Wiring struct {
	service *Service

	mu      sync.Mutex
	subs    []oofevents.Subscription
	stopped bool
}

// NewWiring subscribes to typed game action and match lifecycle events.
func NewWiring(bus oofevents.PluginBus, service *Service) *Wiring {
	w := &Wiring{service: service}
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
	if !ok || w.service == nil {
		return
	}
	w.service.HandleGameAction(gameAction)
}

func (w *Wiring) handleMatchStarted(event oofevents.OOFEvent) {
	matchStarted, ok := oofevents.Unwrap(event).(oofevents.MatchStartedEvent)
	if !ok || w.service == nil {
		return
	}
	w.service.HandleMatchStarted(matchStarted)
}

func (w *Wiring) handleMatchRestarted(event oofevents.OOFEvent) {
	matchRestarted, ok := oofevents.Unwrap(event).(oofevents.MatchRestartedEvent)
	if !ok || w.service == nil {
		return
	}
	w.service.HandleMatchRestarted(matchRestarted)
}

func (w *Wiring) handleMatchEnded(event oofevents.OOFEvent) {
	matchEnded, ok := oofevents.Unwrap(event).(oofevents.MatchEndedEvent)
	if !ok || w.service == nil {
		return
	}
	w.service.HandleMatchEnded(matchEnded)
}

func (w *Wiring) handleMatchDestroyed(event oofevents.OOFEvent) {
	matchDestroyed, ok := oofevents.Unwrap(event).(oofevents.MatchDestroyedEvent)
	if !ok || w.service == nil {
		return
	}
	w.service.HandleMatchDestroyed(matchDestroyed)
}
