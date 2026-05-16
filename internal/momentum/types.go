package momentum

import (
	"time"

	"OOF_RL/internal/oofevents"
)

// Config controls the event-derived signal model. Values are clamped to safe
// ranges by NewEngine so callers can pass a zero-value config.
type Config struct {
	// Decay keeps the signal responsive by fading earlier actions each time a
	// new supported action arrives. Valid range: [0, 1].
	Decay float64
}

// DefaultConfig returns conservative defaults for runtime-only momentum signals.
func DefaultConfig() Config {
	return Config{
		Decay: 0.88,
	}
}

// MomentumState is a runtime-only snapshot derived from typed game actions.
// It is intended for display/enrichment consumers and must not be persisted as
// source-of-truth match data.
type MomentumState struct {
	MatchGUID string
	Sequence  int
	Teams     map[oofevents.Team]TeamSignal
	LastEvent EventSignal
}

// SnapshotProvider is the read-only Momentum access contract for internal
// consumers. It intentionally excludes event handlers, reset methods, and
// wiring controls so consumers cannot own engine logic.
type SnapshotProvider interface {
	Snapshot() MomentumState
	Status() ServiceStatus
}

// TeamSignal describes bounded, event-derived team influence. The fields are
// heuristics, not possession, rotation, win odds, or tactical certainty.
type TeamSignal struct {
	Pressure            float64
	MomentumInfluence   float64
	ContestInvolvement  float64
	EventDerivedControl float64
	Confidence          float64
	Volatility          float64
}

// EventSignal captures the most recent supported action applied to the engine.
type EventSignal struct {
	Action     oofevents.ActionKind
	ActorTeam  oofevents.Team
	ImpactTeam oofevents.Team
	PlayerID   string
	PlayerName string
	VictimID   string
	IsOwnGoal  bool
	IsEpicSave bool
	OccurredAt time.Time
	MatchGUID  string
}
