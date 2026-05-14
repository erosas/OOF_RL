package oofevents

import "time"

// Event type string constants — stable subscription keys.
const (
	TypeMatchStarted    = "match.started"
	TypeMatchEnded      = "match.ended"
	TypeMatchDestroyed  = "match.destroyed"
	TypeMatchRestarted  = "match.restarted"
	TypeOvertimeStarted = "overtime.started"
	TypeGoalScored      = "goal.scored"
	TypeStatFeed        = "stat.feed"
	TypeClockUpdated    = "clock.updated"
	TypeBallHit         = "ball.hit"
	TypeCrossbarHit     = "crossbar.hit"
	TypeStateUpdated    = "state.updated"
)

// NewBase constructs a Base stamped with the current time.
// Useful in tests and for plugins implementing custom event types.
func NewBase(typ string, cert Certainty, guid string) Base {
	return Base{EventType: typ, At: time.Now(), Cert: cert, GUID: guid}
}

// MatchStartedEvent is published when a match GUID is first assigned.
// Corresponds to RL "MatchCreated" or "MatchInitialized".
type MatchStartedEvent struct{ Base }

func NewMatchStarted(guid string) MatchStartedEvent {
	return MatchStartedEvent{NewBase(TypeMatchStarted, Authoritative, guid)}
}

// MatchEndedEvent is published when RL reports a match winner.
type MatchEndedEvent struct {
	Base
	WinnerTeamNum int
}

func NewMatchEnded(guid string, winnerTeamNum int) MatchEndedEvent {
	return MatchEndedEvent{NewBase(TypeMatchEnded, Authoritative, guid), winnerTeamNum}
}

// MatchDestroyedEvent is published when RL tears down the match session.
type MatchDestroyedEvent struct{ Base }

func NewMatchDestroyed() MatchDestroyedEvent {
	return MatchDestroyedEvent{NewBase(TypeMatchDestroyed, Authoritative, "")}
}

// MatchRestartedEvent (Inferred) is published when a new GUID appears mid-session.
type MatchRestartedEvent struct {
	Base
	PreviousGUID string
}

func NewMatchRestarted(guid, previousGUID string) MatchRestartedEvent {
	return MatchRestartedEvent{NewBase(TypeMatchRestarted, Inferred, guid), previousGUID}
}

// OvertimeStartedEvent (Inferred) is published when bOvertime flips false→true.
type OvertimeStartedEvent struct {
	Base
	ClockSeconds int
}

func NewOvertimeStarted(guid string, clockSeconds int) OvertimeStartedEvent {
	return OvertimeStartedEvent{NewBase(TypeOvertimeStarted, Inferred, guid), clockSeconds}
}

// GoalScoredEvent carries goal details.
type GoalScoredEvent struct {
	Base
	Scorer            string
	ScorerShortcut    int
	Assister          string // "" if no assist
	AssisterShortcut  int
	LastTouchShortcut int
	GoalSpeed         float64
	GoalTime          float64
	ImpactX           float64
	ImpactY           float64
	ImpactZ           float64
	TeamNum           int
}

func NewGoalScored(guid, scorer string, scorerShortcut int, assister string, assisterShortcut, lastTouchShortcut int, speed, goalTime, impactX, impactY, impactZ float64, teamNum int) GoalScoredEvent {
	return GoalScoredEvent{
		Base:              NewBase(TypeGoalScored, Authoritative, guid),
		Scorer:            scorer,
		ScorerShortcut:    scorerShortcut,
		Assister:          assister,
		AssisterShortcut:  assisterShortcut,
		LastTouchShortcut: lastTouchShortcut,
		GoalSpeed:         speed,
		GoalTime:          goalTime,
		ImpactX:           impactX,
		ImpactY:           impactY,
		ImpactZ:           impactZ,
		TeamNum:           teamNum,
	}
}

// StatFeedEvent carries a single stat notification.
// EventName: "Goal", "OwnGoal", "Save", "EpicSave", "Assist", "Demolish", "Shot".
type StatFeedEvent struct {
	Base
	EventName               string
	MainTarget              string
	MainTargetShortcut      int
	MainTargetTeamNum       int
	SecondaryTarget         string // present for Demolish (victim)
	SecondaryTargetShortcut int
}

func NewStatFeed(guid, eventName, mainTarget string, mainTargetShortcut, mainTargetTeamNum int, secondaryTarget string, secondaryTargetShortcut int) StatFeedEvent {
	return StatFeedEvent{
		Base:                    NewBase(TypeStatFeed, Authoritative, guid),
		EventName:               eventName,
		MainTarget:              mainTarget,
		MainTargetShortcut:      mainTargetShortcut,
		MainTargetTeamNum:       mainTargetTeamNum,
		SecondaryTarget:         secondaryTarget,
		SecondaryTargetShortcut: secondaryTargetShortcut,
	}
}

// ClockUpdatedEvent carries a clock tick.
type ClockUpdatedEvent struct {
	Base
	TimeSeconds int
	IsOvertime  bool
}

func NewClockUpdated(guid string, seconds int, overtime bool) ClockUpdatedEvent {
	return ClockUpdatedEvent{
		Base:        NewBase(TypeClockUpdated, Authoritative, guid),
		TimeSeconds: seconds,
		IsOvertime:  overtime,
	}
}

// BallHitEvent carries a ball touch. High-frequency; off by default.
type BallHitEvent struct {
	Base
	PlayerName      string
	PlayerPrimaryID string
	PlayerShortcut  int
	PreHitSpeed     float64
	PostHitSpeed    float64
	LocX            float64
	LocY            float64
	LocZ            float64
}

func NewBallHit(guid, playerName, playerPrimaryID string, playerShortcut int, preSpeed, postSpeed, locX, locY, locZ float64) BallHitEvent {
	return BallHitEvent{
		Base:            NewBase(TypeBallHit, Authoritative, guid),
		PlayerName:      playerName,
		PlayerPrimaryID: playerPrimaryID,
		PlayerShortcut:  playerShortcut,
		PreHitSpeed:     preSpeed,
		PostHitSpeed:    postSpeed,
		LocX:            locX,
		LocY:            locY,
		LocZ:            locZ,
	}
}

// CrossbarHitEvent carries a crossbar/post hit.
type CrossbarHitEvent struct {
	Base
	BallSpeed   float64
	ImpactForce float64
	LastToucher string
}

func NewCrossbarHit(guid, lastToucher string, ballSpeed, impactForce float64) CrossbarHitEvent {
	return CrossbarHitEvent{
		Base:        NewBase(TypeCrossbarHit, Authoritative, guid),
		BallSpeed:   ballSpeed,
		ImpactForce: impactForce,
		LastToucher: lastToucher,
	}
}

// StateUpdatedEvent carries a full game snapshot. Rate-limited; not every tick.
type StateUpdatedEvent struct {
	Base
	Players []PlayerSnapshot
	Game    GameSnapshot
}

func NewStateUpdated(guid string, players []PlayerSnapshot, game GameSnapshot) StateUpdatedEvent {
	return StateUpdatedEvent{
		Base:    NewBase(TypeStateUpdated, Authoritative, guid),
		Players: players,
		Game:    game,
	}
}

// PlayerSnapshot is a point-in-time player state extracted from a game update.
// JSON tags mirror the RL API wire format so snapshots can be served directly to the frontend.
type PlayerSnapshot struct {
	Name           string   `json:"Name"`
	PrimaryID      string   `json:"PrimaryId"`
	Shortcut       int      `json:"Shortcut"`
	TeamNum        int      `json:"TeamNum"`
	Score          int      `json:"Score"`
	Goals          int      `json:"Goals"`
	Shots          int      `json:"Shots"`
	Assists        int      `json:"Assists"`
	Saves          int      `json:"Saves"`
	Touches        int      `json:"Touches"`
	CarTouches     int      `json:"CarTouches"`
	Demos          int      `json:"Demos"`
	Speed          *float64 `json:"Speed,omitempty"`
	Boost          *int     `json:"Boost,omitempty"`
	IsBoosting     *bool    `json:"bBoosting,omitempty"`
	IsOnWall       *bool    `json:"bOnWall,omitempty"`
	IsPowersliding *bool    `json:"bPowersliding,omitempty"`
	IsDemolished   *bool    `json:"bDemolished,omitempty"`
	IsSupersonic   *bool    `json:"bSupersonic,omitempty"`
}

// GameSnapshot is a point-in-time game state extracted from a game update.
type GameSnapshot struct {
	Teams       []TeamSnapshot `json:"Teams"`
	TimeSeconds int            `json:"TimeSeconds"`
	IsOvertime  bool           `json:"bOvertime"`
	IsReplay    bool           `json:"bReplay"`
	Ball        BallSnapshot   `json:"Ball"`
	Arena       string         `json:"Arena"`
	Playlist    *int           `json:"Playlist,omitempty"`
	HasWinner   bool           `json:"bHasWinner"`
	Winner      string         `json:"Winner"`
}

// BallSnapshot carries ball state from a game update.
type BallSnapshot struct {
	Speed float64 `json:"Speed"`
}

// TeamSnapshot is a point-in-time team state.
type TeamSnapshot struct {
	Name           string `json:"Name"`
	TeamNum        int    `json:"TeamNum"`
	Score          int    `json:"Score"`
	ColorPrimary   string `json:"ColorPrimary,omitempty"`
	ColorSecondary string `json:"ColorSecondary,omitempty"`
}
