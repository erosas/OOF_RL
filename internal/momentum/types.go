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

	// The remaining weights are explicit display/enrichment weights adapted from
	// the old PR #47 event-pressure model onto the bounded MomentumState fields.
	// They are not possession, tactical certainty, or win-probability values.
	TouchChainWindow       time.Duration
	AlternatingTouchWindow time.Duration
	DemoBeforeShotWindow   time.Duration
	DemoBeforeGoalWindow   time.Duration

	BallHitControl               float64
	BallHitPressure              float64
	SameTeamTouchControlBonus    float64
	SameTeamTouchPressureBonus   float64
	MaxTouchChainBonus           float64
	OpponentTouchNewControl      float64
	OpponentTouchPreviousPenalty float64

	ShotControl  float64
	ShotPressure float64

	SaveDefendingControl        float64
	SaveForcedAttackingPressure float64
	EpicSaveControlBonus        float64
	EpicSaveContestBonus        float64
	EpicSaveConfidenceBonus     float64
	EpicSaveVolatilityBonus     float64

	GoalScoringControl  float64
	GoalScoringPressure float64

	AssistPressure        float64
	AssistConfidenceBonus float64

	DemoPressure                float64
	DemoBeforeShotPressureBonus float64
	DemoBeforeGoalPressureBonus float64

	AlternatingTouchVolatilityBonus float64

	ConfidenceBase       float64
	ConfidencePlayerID   float64
	ConfidencePlayerName float64
	ConfidenceDemoVictim float64
}

// DefaultConfig returns conservative defaults for runtime-only momentum signals.
func DefaultConfig() Config {
	return Config{
		Decay: 0.88,

		TouchChainWindow:       5 * time.Second,
		AlternatingTouchWindow: 2 * time.Second,
		DemoBeforeShotWindow:   5 * time.Second,
		DemoBeforeGoalWindow:   8 * time.Second,

		BallHitControl:               0.06,
		BallHitPressure:              0.02,
		SameTeamTouchControlBonus:    0.025,
		SameTeamTouchPressureBonus:   0.012,
		MaxTouchChainBonus:           0.12,
		OpponentTouchNewControl:      0.06,
		OpponentTouchPreviousPenalty: -0.025,

		ShotControl:  0.08,
		ShotPressure: 0.26,

		SaveDefendingControl:        0.13,
		SaveForcedAttackingPressure: 0.16,
		EpicSaveControlBonus:        0.05,
		EpicSaveContestBonus:        0.05,
		EpicSaveConfidenceBonus:     0.02,
		EpicSaveVolatilityBonus:     0.04,

		GoalScoringControl:  0.16,
		GoalScoringPressure: 0.42,

		AssistPressure:        0.14,
		AssistConfidenceBonus: 0.02,

		DemoPressure:                0.08,
		DemoBeforeShotPressureBonus: 0.16,
		DemoBeforeGoalPressureBonus: 0.22,

		AlternatingTouchVolatilityBonus: 0.12,

		ConfidenceBase:       0.08,
		ConfidencePlayerID:   0.04,
		ConfidencePlayerName: 0.02,
		ConfidenceDemoVictim: 0.02,
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
