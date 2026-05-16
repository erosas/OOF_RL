package momentum

import (
	"sync"

	"OOF_RL/internal/oofevents"
)

// Service owns a Momentum Engine instance and provides synchronized access to
// runtime-only momentum state. It does not subscribe to or publish on the event
// bus; wiring belongs to a focused integration layer.
type Service struct {
	mu       sync.RWMutex
	engine   *Engine
	inactive bool
	reason   string
}

// ServiceStatus describes runtime service lifecycle state.
type ServiceStatus struct {
	Active bool
	Reason string
}

// NewService creates a thread-safe runtime service around the Momentum Engine.
func NewService(config Config) *Service {
	return &Service{
		engine: NewEngine(config),
	}
}

// HandleGameAction applies one typed game action unless the service is inactive.
func (s *Service) HandleGameAction(event oofevents.GameActionEvent) MomentumState {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.inactive {
		return s.engine.Snapshot()
	}
	return s.engine.ApplyGameAction(event)
}

// Snapshot returns a copy of the current runtime-only momentum state.
func (s *Service) Snapshot() MomentumState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.engine.Snapshot()
}

// Status returns a copy of the service lifecycle state.
func (s *Service) Status() ServiceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return ServiceStatus{
		Active: !s.inactive,
		Reason: s.reason,
	}
}

// Reset clears runtime state and marks the service active again.
func (s *Service) Reset(reason string) MomentumState {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.engine.Reset()
	s.inactive = false
	s.reason = reason
	return s.engine.Snapshot()
}

// HandleMatchStarted resets state for a new match lifecycle boundary.
func (s *Service) HandleMatchStarted(event oofevents.MatchStartedEvent) MomentumState {
	return s.Reset("match.started:" + event.MatchGUID())
}

// HandleMatchRestarted resets state for a new match GUID observed mid-session.
func (s *Service) HandleMatchRestarted(event oofevents.MatchRestartedEvent) MomentumState {
	return s.Reset("match.restarted:" + event.MatchGUID())
}

// HandleMatchDestroyed clears state when RL tears down the match session.
func (s *Service) HandleMatchDestroyed(_ oofevents.MatchDestroyedEvent) MomentumState {
	return s.Reset("match.destroyed")
}

// HandleMatchEnded freezes the current snapshot until a reset or new match occurs.
func (s *Service) HandleMatchEnded(event oofevents.MatchEndedEvent) MomentumState {
	return s.markInactive("match.ended:" + event.MatchGUID())
}

// MarkMatchEnded freezes the current snapshot until a reset or new match occurs.
func (s *Service) MarkMatchEnded() MomentumState {
	return s.markInactive("match.ended")
}

func (s *Service) markInactive(reason string) MomentumState {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.inactive = true
	s.reason = reason
	return s.engine.Snapshot()
}
