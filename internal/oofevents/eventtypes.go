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
	Scorer    string
	Assister  string // "" if no assist
	GoalSpeed float64
	GoalTime  float64
	TeamNum   int
}

func NewGoalScored(guid, scorer, assister string, speed, goalTime float64, teamNum int) GoalScoredEvent {
	return GoalScoredEvent{
		Base:      NewBase(TypeGoalScored, Authoritative, guid),
		Scorer:    scorer,
		Assister:  assister,
		GoalSpeed: speed,
		GoalTime:  goalTime,
		TeamNum:   teamNum,
	}
}

// StatFeedEvent carries a single stat notification.
// EventName: "Goal", "OwnGoal", "Save", "EpicSave", "Assist", "Demolish", "Shot".
type StatFeedEvent struct {
	Base
	EventName       string
	MainTarget      string
	SecondaryTarget string // present for Demolish (victim)
}

func NewStatFeed(guid, eventName, mainTarget, secondaryTarget string) StatFeedEvent {
	return StatFeedEvent{
		Base:            NewBase(TypeStatFeed, Authoritative, guid),
		EventName:       eventName,
		MainTarget:      mainTarget,
		SecondaryTarget: secondaryTarget,
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
	PlayerName   string
	PreHitSpeed  float64
	PostHitSpeed float64
}

func NewBallHit(guid, playerName string, preSpeed, postSpeed float64) BallHitEvent {
	return BallHitEvent{
		Base:         NewBase(TypeBallHit, Authoritative, guid),
		PlayerName:   playerName,
		PreHitSpeed:  preSpeed,
		PostHitSpeed: postSpeed,
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
type PlayerSnapshot struct {
	Name      string
	PrimaryID string
	TeamNum   int
	Score     int
	Goals     int
	Shots     int
	Assists   int
	Saves     int
	Demos     int
	Speed     *float64
	Boost     *int
}

// GameSnapshot is a point-in-time game state extracted from a game update.
type GameSnapshot struct {
	Teams       []TeamSnapshot
	TimeSeconds int
	IsOvertime  bool
	BallSpeed   float64
	Arena       string
	Playlist    *int
	HasWinner   bool
	Winner      string
}

// TeamSnapshot is a point-in-time team state.
type TeamSnapshot struct {
	Name    string
	TeamNum int
	Score   int
}