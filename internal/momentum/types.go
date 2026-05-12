package momentum

import (
	"encoding/json"
	"strings"
)

type Team string

const (
	TeamBlue   Team = "blue"
	TeamOrange Team = "orange"
)

func TeamFromNum(teamNum int) (Team, bool) {
	switch teamNum {
	case 0:
		return TeamBlue, true
	case 1:
		return TeamOrange, true
	default:
		return "", false
	}
}

func Opponent(team Team) Team {
	if team == TeamBlue {
		return TeamOrange
	}
	if team == TeamOrange {
		return TeamBlue
	}
	return ""
}

type EventType string

const (
	EventBallHit EventType = "ball_hit"
	EventShot    EventType = "shot"
	EventSave    EventType = "save"
	EventGoal    EventType = "goal"
	EventAssist  EventType = "assist"
	EventDemo    EventType = "demo"
)

type FlowState string

const (
	StateNeutral        FlowState = "NEUTRAL"
	StateBlueControl    FlowState = "BLUE_CONTROL"
	StateOrangeControl  FlowState = "ORANGE_CONTROL"
	StateBluePressure   FlowState = "BLUE_PRESSURE"
	StateOrangePressure FlowState = "ORANGE_PRESSURE"
	StateVolatile       FlowState = "VOLATILE"
)

type OverlayPulse string

const (
	PulseShot            OverlayPulse = "SHOT"
	PulseSaveForced      OverlayPulse = "SAVE_FORCED"
	PulseDemoPressure    OverlayPulse = "DEMO_PRESSURE"
	PulseGoalBurst       OverlayPulse = "GOAL_BURST"
	PulseVolatileContest OverlayPulse = "VOLATILE_CONTEST"
)

type NormalizedGameEvent struct {
	Type        EventType `json:"type"`
	Team        Team      `json:"team"`
	PlayerID    string    `json:"playerId,omitempty"`
	PlayerName  string    `json:"playerName,omitempty"`
	AssisterID  string    `json:"assisterId,omitempty"`
	VictimID    string    `json:"victimId,omitempty"`
	Time        int64     `json:"time"`
	MatchClock  *int      `json:"matchClock,omitempty"`
	MatchGUID   string    `json:"matchGuid,omitempty"`
	SourceEvent string    `json:"sourceEvent,omitempty"`
}

type TeamMomentumState struct {
	Control             float64 `json:"control"`
	Pressure            float64 `json:"pressure"`
	TouchChain          int     `json:"touchChain"`
	LastTouchTime       *int64  `json:"lastTouchTime,omitempty"`
	LastStrongEventTime *int64  `json:"lastStrongEventTime,omitempty"`
}

type MomentumState struct {
	Blue          TeamMomentumState `json:"blue"`
	Orange        TeamMomentumState `json:"orange"`
	Volatility    float64           `json:"volatility"`
	LastEventTime *int64            `json:"lastEventTime,omitempty"`
	CurrentState  FlowState         `json:"currentState"`
	Confidence    float64           `json:"confidence"`
	LastTouchTeam Team              `json:"lastTouchTeam,omitempty"`
	LastPulse     OverlayPulse      `json:"lastPulse,omitempty"`
	LastPulseTeam Team              `json:"lastPulseTeam,omitempty"`
	LastPulseTime *int64            `json:"lastPulseTime,omitempty"`
}

type MomentumFlowOutput struct {
	State        FlowState     `json:"state"`
	Blue         TeamOutput    `json:"blue"`
	Orange       TeamOutput    `json:"orange"`
	Confidence   float64       `json:"confidence"`
	Volatility   float64       `json:"volatility"`
	DominantTeam *Team         `json:"dominantTeam,omitempty"`
	Overlay      OverlayOutput `json:"overlay"`
	Debug        *DebugOutput  `json:"debug,omitempty"`
}

type TeamOutput struct {
	Control       float64 `json:"control"`
	Pressure      float64 `json:"pressure"`
	ControlShare  float64 `json:"controlShare"`
	PressureShare float64 `json:"pressureShare"`
}

type OverlayOutput struct {
	ScoreboardGlowTeam       *Team        `json:"scoreboardGlowTeam,omitempty"`
	ScoreboardGlowIntensity  float64      `json:"scoreboardGlowIntensity"`
	ClockRingTeam            *Team        `json:"clockRingTeam,omitempty"`
	ClockRingIntensity       float64      `json:"clockRingIntensity"`
	MomentumBarBluePercent   float64      `json:"momentumBarBluePercent"`
	MomentumBarOrangePercent float64      `json:"momentumBarOrangePercent"`
	Pulse                    OverlayPulse `json:"pulse,omitempty"`
	PulseTeam                Team         `json:"pulseTeam,omitempty"`
}

type DebugOutput struct {
	LastEvents      []NormalizedGameEvent `json:"lastEvents"`
	LastStrongEvent *NormalizedGameEvent  `json:"lastStrongEvent,omitempty"`
	EventCounts     map[EventType]int     `json:"eventCounts,omitempty"`
	SourceCounts    map[string]int        `json:"sourceCounts,omitempty"`
	Reasons         []string              `json:"reasons"`
	WeightsApplied  []WeightApplied       `json:"weightsApplied"`
}

type WeightApplied struct {
	EventType       EventType `json:"eventType"`
	Team            Team      `json:"team"`
	ControlDelta    *float64  `json:"controlDelta,omitempty"`
	PressureDelta   *float64  `json:"pressureDelta,omitempty"`
	VolatilityDelta *float64  `json:"volatilityDelta,omitempty"`
	Reason          string    `json:"reason"`
}

func (e *NormalizedGameEvent) UnmarshalJSON(data []byte) error {
	type alias NormalizedGameEvent
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	a.Type = EventType(strings.TrimSpace(string(a.Type)))
	a.Team = Team(strings.ToLower(strings.TrimSpace(string(a.Team))))
	*e = NormalizedGameEvent(a)
	return nil
}
