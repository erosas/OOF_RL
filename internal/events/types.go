package events

import "encoding/json"

type Envelope struct {
	Event string          `json:"Event"`
	Data  json.RawMessage `json:"Data"`
}

type PlayerRef struct {
	Name      string `json:"Name"`
	PrimaryId string `json:"PrimaryId,omitempty"`
	Shortcut  int    `json:"Shortcut"`
	TeamNum   int    `json:"TeamNum"`
}

type AttackerRef struct {
	Name     string `json:"Name"`
	Shortcut int    `json:"Shortcut"`
	TeamNum  int    `json:"TeamNum"`
}

type Player struct {
	Name          string       `json:"Name"`
	PrimaryId     string       `json:"PrimaryId"`
	Shortcut      int          `json:"Shortcut"`
	TeamNum       int          `json:"TeamNum"`
	Score         int          `json:"Score"`
	Goals         int          `json:"Goals"`
	Shots         int          `json:"Shots"`
	Assists       int          `json:"Assists"`
	Saves         int          `json:"Saves"`
	Touches       int          `json:"Touches"`
	CarTouches    int          `json:"CarTouches"`
	Demos         int          `json:"Demos"`
	BHasCar       *bool        `json:"bHasCar,omitempty"`
	Speed         *float64     `json:"Speed,omitempty"`
	Boost         *int         `json:"Boost,omitempty"`
	BBoosting     *bool        `json:"bBoosting,omitempty"`
	BOnGround     *bool        `json:"bOnGround,omitempty"`
	BOnWall       *bool        `json:"bOnWall,omitempty"`
	BPowersliding *bool        `json:"bPowersliding,omitempty"`
	BDemolished   *bool        `json:"bDemolished,omitempty"`
	BSupersonic   *bool        `json:"bSupersonic,omitempty"`
	Attacker      *AttackerRef `json:"Attacker,omitempty"`
}

type Team struct {
	Name           string `json:"Name"`
	TeamNum        int    `json:"TeamNum"`
	Score          int    `json:"Score"`
	ColorPrimary   string `json:"ColorPrimary"`
	ColorSecondary string `json:"ColorSecondary"`
}

type Ball struct {
	Speed   float64 `json:"Speed"`
	TeamNum int     `json:"TeamNum"`
}

type Target struct {
	Name     string `json:"Name"`
	Shortcut int    `json:"Shortcut"`
	TeamNum  int    `json:"TeamNum"`
}

type GameState struct {
	Teams       []Team   `json:"Teams"`
	TimeSeconds int      `json:"TimeSeconds"`
	BOvertime   bool     `json:"bOvertime"`
	Frame       *int     `json:"Frame,omitempty"`
	Elapsed     *float64 `json:"Elapsed,omitempty"`
	Ball        Ball     `json:"Ball"`
	BReplay     bool     `json:"bReplay"`
	BHasWinner  bool     `json:"bHasWinner"`
	Winner      string   `json:"Winner"`
	Arena       string   `json:"Arena"`
	BHasTarget  bool     `json:"bHasTarget"`
	Target      *Target  `json:"Target,omitempty"`
	Playlist    *int     `json:"Playlist,omitempty"`
}

type UpdateStateData struct {
	MatchGuid string    `json:"MatchGuid"`
	Players   []Player  `json:"Players"`
	Game      GameState `json:"Game"`
}

type Vec3 struct {
	X float64 `json:"X"`
	Y float64 `json:"Y"`
	Z float64 `json:"Z"`
}

type BallHitBall struct {
	PreHitSpeed  float64 `json:"PreHitSpeed"`
	PostHitSpeed float64 `json:"PostHitSpeed"`
	Location     Vec3    `json:"Location"`
}

type BallHitData struct {
	MatchGuid string      `json:"MatchGuid"`
	Players   []PlayerRef `json:"Players"`
	Ball      BallHitBall `json:"Ball"`
}

type LastTouch struct {
	Player PlayerRef `json:"Player"`
	Speed  float64   `json:"Speed"`
}

type GoalScoredData struct {
	MatchGuid      string     `json:"MatchGuid"`
	GoalSpeed      float64    `json:"GoalSpeed"`
	GoalTime       float64    `json:"GoalTime"`
	ImpactLocation Vec3       `json:"ImpactLocation"`
	Scorer         PlayerRef  `json:"Scorer"`
	Assister       *PlayerRef `json:"Assister,omitempty"`
	BallLastTouch  LastTouch  `json:"BallLastTouch"`
}

type MatchGuidData struct {
	MatchGuid string `json:"MatchGuid"`
}

type MatchEndedData struct {
	MatchGuid     string `json:"MatchGuid"`
	WinnerTeamNum int    `json:"WinnerTeamNum"`
}

type ClockData struct {
	MatchGuid   string `json:"MatchGuid"`
	TimeSeconds int    `json:"TimeSeconds"`
	BOvertime   bool   `json:"bOvertime"`
}

type CrossbarHitData struct {
	MatchGuid     string    `json:"MatchGuid"`
	BallLocation  Vec3      `json:"BallLocation"`
	BallSpeed     float64   `json:"BallSpeed"`
	ImpactForce   float64   `json:"ImpactForce"`
	BallLastTouch LastTouch `json:"BallLastTouch"`
}

// StatfeedEvent fires for each in-game stat notification.
// EventName values: "Goal", "OwnGoal", "Save", "EpicSave", "Assist", "Demolish", "Shot".
// MainTarget is the actor; SecondaryTarget is the victim (only present for Demolish).
type StatfeedEventData struct {
	MatchGuid       string     `json:"MatchGuid"`
	EventName       string     `json:"EventName"`
	MainTarget      PlayerRef  `json:"MainTarget"`
	SecondaryTarget *PlayerRef `json:"SecondaryTarget,omitempty"`
}
