package momentum

import (
	"time"

	"OOF_RL/internal/oofevents"
)

// Config controls the event-derived signal model. Values are filled from the
// old PR #47 event-pressure defaults by NewEngine so callers can pass a
// zero-value config.
type Config struct {
	// Decay is retained for older tests/callers. When provided, it is used as a
	// fallback for the per-second decay values below.
	Decay float64

	ControlDecayPerSecond    float64
	PressureDecayPerSecond   float64
	VolatilityDecayPerSecond float64
	ConfidenceDecayPerSecond float64

	ControlThreshold       float64
	PressureThreshold      float64
	ConfidenceThreshold    float64
	VolatilityThreshold    float64
	PressureShareThreshold float64
	ControlShareThreshold  float64

	// The remaining weights are explicit display/enrichment weights restored
	// from the old PR #47 event-pressure model.
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
		Decay: 0,

		ControlDecayPerSecond:    0.72,
		PressureDecayPerSecond:   0.84,
		VolatilityDecayPerSecond: 0.78,
		ConfidenceDecayPerSecond: 0.82,

		ControlThreshold:       1.8,
		PressureThreshold:      3.2,
		ConfidenceThreshold:    0.35,
		VolatilityThreshold:    0.65,
		PressureShareThreshold: 0.62,
		ControlShareThreshold:  0.60,

		TouchChainWindow:       5 * time.Second,
		AlternatingTouchWindow: 2 * time.Second,
		DemoBeforeShotWindow:   5 * time.Second,
		DemoBeforeGoalWindow:   8 * time.Second,

		BallHitControl:               1.0,
		BallHitPressure:              0.2,
		SameTeamTouchControlBonus:    0.4,
		SameTeamTouchPressureBonus:   0.15,
		MaxTouchChainBonus:           2.0,
		OpponentTouchNewControl:      1.0,
		OpponentTouchPreviousPenalty: -0.4,

		ShotControl:  0.8,
		ShotPressure: 4.0,

		SaveDefendingControl:        1.5,
		SaveForcedAttackingPressure: 2.5,
		EpicSaveControlBonus:        0.12,
		EpicSaveContestBonus:        0.10,
		EpicSaveConfidenceBonus:     0.06,
		EpicSaveVolatilityBonus:     0.08,

		GoalScoringControl:  2.0,
		GoalScoringPressure: 10.0,

		AssistPressure:        2.0,
		AssistConfidenceBonus: 0.1,

		DemoPressure:                1.0,
		DemoBeforeShotPressureBonus: 2.0,
		DemoBeforeGoalPressureBonus: 3.5,

		AlternatingTouchVolatilityBonus: 1.0,

		ConfidenceBase:       0,
		ConfidencePlayerID:   0,
		ConfidencePlayerName: 0,
		ConfidenceDemoVictim: 0,
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

// TeamSignal describes event-derived team influence. Pressure/control values
// use the old PR #47 event-pressure scale; confidence remains [0, 1].
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
