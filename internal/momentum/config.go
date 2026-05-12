package momentum

type Config struct {
	ShortWindowMs  int64
	MediumWindowMs int64
	LongWindowMs   int64
	MaxEvents      int

	ControlDecayPerSecond    float64
	PressureDecayPerSecond   float64
	VolatilityDecayPerSecond float64
	ConfidenceDecayPerSecond float64

	TouchChainWindowMs       int64
	AlternatingTouchWindowMs int64
	DemoBeforeShotWindowMs   int64
	DemoBeforeGoalWindowMs   int64
	PulseHoldMs              int64

	ControlThreshold       float64
	PressureThreshold      float64
	ConfidenceThreshold    float64
	VolatilityThreshold    float64
	PressureShareThreshold float64
	ControlShareThreshold  float64

	BallHitControl                  float64
	BallHitPressure                 float64
	SameTeamTouchControlBonus       float64
	SameTeamTouchPressureBonus      float64
	MaxChainBonus                   float64
	OpponentTouchNewControl         float64
	OpponentTouchPreviousPenalty    float64
	ShotControl                     float64
	ShotPressure                    float64
	SaveDefendingControl            float64
	SaveForcedAttackingPressure     float64
	GoalScoringControl              float64
	GoalScoringPressure             float64
	AssistPressure                  float64
	AssistConfidenceBonus           float64
	DemoPressure                    float64
	DemoBeforeShotPressureBonus     float64
	DemoBeforeGoalPressureBonus     float64
	AlternatingTouchVolatilityBonus float64
}

func DefaultConfig() Config {
	return Config{
		ShortWindowMs:  3000,
		MediumWindowMs: 8000,
		LongWindowMs:   15000,
		MaxEvents:      128,

		ControlDecayPerSecond:    0.72,
		PressureDecayPerSecond:   0.84,
		VolatilityDecayPerSecond: 0.78,
		ConfidenceDecayPerSecond: 0.82,

		TouchChainWindowMs:       5000,
		AlternatingTouchWindowMs: 2000,
		DemoBeforeShotWindowMs:   5000,
		DemoBeforeGoalWindowMs:   8000,
		PulseHoldMs:              1200,

		ControlThreshold:       1.8,
		PressureThreshold:      3.2,
		ConfidenceThreshold:    0.35,
		VolatilityThreshold:    0.65,
		PressureShareThreshold: 0.62,
		ControlShareThreshold:  0.60,

		BallHitControl:                  1.0,
		BallHitPressure:                 0.2,
		SameTeamTouchControlBonus:       0.4,
		SameTeamTouchPressureBonus:      0.15,
		MaxChainBonus:                   2.0,
		OpponentTouchNewControl:         1.0,
		OpponentTouchPreviousPenalty:    -0.4,
		ShotControl:                     0.8,
		ShotPressure:                    4.0,
		SaveDefendingControl:            1.5,
		SaveForcedAttackingPressure:     2.5,
		GoalScoringControl:              2.0,
		GoalScoringPressure:             10.0,
		AssistPressure:                  2.0,
		AssistConfidenceBonus:           0.1,
		DemoPressure:                    1.0,
		DemoBeforeShotPressureBonus:     2.0,
		DemoBeforeGoalPressureBonus:     3.5,
		AlternatingTouchVolatilityBonus: 1.0,
	}
}
