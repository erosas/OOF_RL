package histstore

import "time"

type Player struct {
	PrimaryID string    `json:"PrimaryID"`
	Name      string    `json:"Name"`
	LastSeen  time.Time `json:"LastSeen"`
}

type Match struct {
	ID            int64
	MatchGUID     string
	Arena         string
	StartedAt     time.Time
	EndedAt       string
	WinnerTeamNum int
	Overtime      bool
	Incomplete    bool
	Forfeit       bool
	PlaylistType  *int
	TeamScore0    *int
	TeamScore1    *int
	PlayerTeam    *int `json:"player_team,omitempty"`
}

type PlayerMatchStats struct {
	PrimaryID  string
	Name       string
	TeamNum    int
	Score      int
	Goals      int
	Shots      int
	Assists    int
	Saves      int
	Touches    int
	CarTouches int
	Demos      int
}

type GoalEvent struct {
	ID           int64
	ScorerID     string
	ScorerName   string
	AssisterID   string
	AssisterName string
	GoalSpeed    float64
	GoalTime     float64
	ImpactX      float64
	ImpactY      float64
	ImpactZ      float64
	ScoredAt     time.Time
}

type StatfeedEvent struct {
	ID         int64     `json:"id"`
	PlayerID   string    `json:"player_id"`
	PlayerName string    `json:"player_name"`
	TeamNum    int       `json:"team_num"`
	EventType  string    `json:"event_type"`
	TargetID   string    `json:"target_id"`
	TargetName string    `json:"target_name"`
	OccurredAt time.Time `json:"occurred_at"`
}

type PlayerAggregate struct {
	PrimaryID string
	Name      string
	Matches   int
	Goals     int
	Shots     int
	Assists   int
	Saves     int
	Demos     int
	Touches   int
}
