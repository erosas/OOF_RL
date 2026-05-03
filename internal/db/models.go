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

type SessionMatch struct {
	MatchID       int64     `json:"match_id"`
	Arena         string    `json:"arena"`
	StartedAt     time.Time `json:"started_at"`
	WinnerTeamNum int       `json:"winner_team_num"` // -1 if not finished
	PlayerTeam    int       `json:"player_team"`
	Goals         int       `json:"goals"`
	Assists       int       `json:"assists"`
	Saves         int       `json:"saves"`
	Shots         int       `json:"shots"`
	Demos         int       `json:"demos"`
	Score         int       `json:"score"`
}

type SavedSession struct {
	ID        int64     `json:"id"`
	PlayerID  string    `json:"player_id"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Games     int       `json:"games"`
	Wins      int       `json:"wins"`
	Losses    int       `json:"losses"`
	Goals     int       `json:"goals"`
	Assists   int       `json:"assists"`
	Saves     int       `json:"saves"`
	Shots     int       `json:"shots"`
	Demos     int       `json:"demos"`
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