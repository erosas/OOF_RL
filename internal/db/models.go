package db

import "time"

type Player struct {
	PrimaryID string    `json:"PrimaryID"`
	Name      string    `json:"Name"`
	LastSeen  time.Time `json:"LastSeen"`
}

type BCUpload struct {
	ReplayName    string    `json:"replay_name"`
	BallchasingID string    `json:"ballchasing_id"`
	BCURL         string    `json:"bc_url"`
	UploadedAt    time.Time `json:"uploaded_at"`
}

type Match struct {
	ID            int64
	MatchGUID     string
	Arena         string
	StartedAt     time.Time
	EndedAt       string
	WinnerTeamNum int
	Overtime      bool
	PlaylistType  *int
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