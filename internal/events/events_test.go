package events_test

import (
	"encoding/json"
	"testing"

	"OOF_RL/internal/events"
)

func TestEnvelopeMarshalUnmarshal(t *testing.T) {
	raw := `{"Event":"TestEvent","Data":{"foo":"bar"}}`
	var env events.Envelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if env.Event != "TestEvent" {
		t.Errorf("Event: got %q, want TestEvent", env.Event)
	}
	if string(env.Data) != `{"foo":"bar"}` {
		t.Errorf("Data: got %s", env.Data)
	}
}

// TestDoubleEncodedEnvelope verifies the RL wire format where Data is a
// JSON-encoded string wrapping the actual payload object.
func TestDoubleEncodedEnvelope(t *testing.T) {
	inner := `{"MatchGuid":"guid-123","WinnerTeamNum":1}`
	innerEncoded, _ := json.Marshal(inner) // produces `"<escaped inner>"`
	wire := `{"Event":"MatchEnded","Data":` + string(innerEncoded) + `}`

	var env events.Envelope
	if err := json.Unmarshal([]byte(wire), &env); err != nil {
		t.Fatalf("Unmarshal envelope: %v", err)
	}

	var dataStr string
	if err := json.Unmarshal(env.Data, &dataStr); err != nil {
		t.Fatalf("Data should be a JSON string: %v", err)
	}

	var d events.MatchEndedData
	if err := json.Unmarshal([]byte(dataStr), &d); err != nil {
		t.Fatalf("Unmarshal inner: %v", err)
	}
	if d.MatchGuid != "guid-123" {
		t.Errorf("MatchGuid: got %q, want guid-123", d.MatchGuid)
	}
	if d.WinnerTeamNum != 1 {
		t.Errorf("WinnerTeamNum: got %d, want 1", d.WinnerTeamNum)
	}
}

func TestUpdateStateDataUnmarshal(t *testing.T) {
	raw := `{
		"MatchGuid": "guid-123",
		"Players": [
			{
				"Name": "Alice",
				"PrimaryId": "pid1",
				"TeamNum": 0,
				"Score": 500,
				"Goals": 3,
				"Shots": 5,
				"Assists": 1,
				"Saves": 2,
				"Touches": 10,
				"CarTouches": 8,
				"Demos": 1
			}
		],
		"Game": {
			"TimeSeconds": 300,
			"bOvertime": false,
			"Arena": "DFH Stadium",
			"Teams": [
				{"TeamNum": 0, "Score": 2, "Name": "Blue"},
				{"TeamNum": 1, "Score": 1, "Name": "Orange"}
			],
			"Ball": {"Speed": 1200.5}
		}
	}`
	var d events.UpdateStateData
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if d.MatchGuid != "guid-123" {
		t.Errorf("MatchGuid: got %q", d.MatchGuid)
	}
	if len(d.Players) != 1 {
		t.Fatalf("Players: got %d", len(d.Players))
	}
	p := d.Players[0]
	if p.Name != "Alice" || p.PrimaryId != "pid1" {
		t.Errorf("Player identity: %+v", p)
	}
	if p.Goals != 3 || p.Shots != 5 || p.Assists != 1 || p.Saves != 2 {
		t.Errorf("Player stats: %+v", p)
	}
	if d.Game.TimeSeconds != 300 {
		t.Errorf("TimeSeconds: got %d", d.Game.TimeSeconds)
	}
	if d.Game.Arena != "DFH Stadium" {
		t.Errorf("Arena: got %q", d.Game.Arena)
	}
	if d.Game.Ball.Speed != 1200.5 {
		t.Errorf("Ball.Speed: got %f", d.Game.Ball.Speed)
	}
	if len(d.Game.Teams) != 2 {
		t.Errorf("Teams: got %d", len(d.Game.Teams))
	}
}

func TestUpdateStateOvertimeFlag(t *testing.T) {
	raw := `{"MatchGuid":"g","Players":[],"Game":{"bOvertime":true,"TimeSeconds":0,"Ball":{}}}`
	var d events.UpdateStateData
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !d.Game.BOvertime {
		t.Error("expected BOvertime true")
	}
}

func TestGoalScoredDataWithAssister(t *testing.T) {
	raw := `{
		"MatchGuid": "guid-123",
		"GoalSpeed": 120.5,
		"GoalTime": 45.3,
		"ImpactLocation": {"X": 1.0, "Y": 2.0, "Z": 3.0},
		"Scorer":   {"Name": "Alice", "PrimaryId": "pid1", "TeamNum": 0},
		"Assister": {"Name": "Bob",   "PrimaryId": "pid2", "TeamNum": 0},
		"BallLastTouch": {
			"Player": {"Name": "Alice", "PrimaryId": "pid1", "TeamNum": 0},
			"Speed": 115.0
		}
	}`
	var d events.GoalScoredData
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if d.GoalSpeed != 120.5 {
		t.Errorf("GoalSpeed: got %f", d.GoalSpeed)
	}
	if d.GoalTime != 45.3 {
		t.Errorf("GoalTime: got %f", d.GoalTime)
	}
	if d.ImpactLocation.X != 1.0 || d.ImpactLocation.Y != 2.0 || d.ImpactLocation.Z != 3.0 {
		t.Errorf("ImpactLocation: %+v", d.ImpactLocation)
	}
	if d.Scorer.PrimaryId != "pid1" {
		t.Errorf("Scorer: got %q", d.Scorer.PrimaryId)
	}
	if d.Assister == nil || d.Assister.PrimaryId != "pid2" {
		t.Errorf("Assister: %+v", d.Assister)
	}
	if d.BallLastTouch.Player.PrimaryId != "pid1" {
		t.Errorf("BallLastTouch.Player.PrimaryId: got %q", d.BallLastTouch.Player.PrimaryId)
	}
}

func TestGoalScoredDataNoAssister(t *testing.T) {
	raw := `{
		"MatchGuid": "guid-123",
		"GoalSpeed": 100.0,
		"GoalTime": 30.0,
		"ImpactLocation": {"X": 0, "Y": 0, "Z": 0},
		"Scorer": {"Name": "Alice", "PrimaryId": "pid1", "TeamNum": 0},
		"BallLastTouch": {"Player": {"Name": "Alice", "PrimaryId": "pid1"}, "Speed": 100.0}
	}`
	var d events.GoalScoredData
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if d.Assister != nil {
		t.Errorf("expected nil Assister, got %+v", d.Assister)
	}
}

func TestBallHitDataUnmarshal(t *testing.T) {
	raw := `{
		"MatchGuid": "guid-123",
		"Players": [{"Name": "Alice", "PrimaryId": "pid1", "TeamNum": 0}],
		"Ball": {
			"PreHitSpeed": 50.0,
			"PostHitSpeed": 80.0,
			"Location": {"X": 10.0, "Y": 20.0, "Z": 30.0}
		}
	}`
	var d events.BallHitData
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if d.Ball.PreHitSpeed != 50.0 || d.Ball.PostHitSpeed != 80.0 {
		t.Errorf("speeds: %+v", d.Ball)
	}
	if d.Ball.Location.X != 10.0 || d.Ball.Location.Y != 20.0 || d.Ball.Location.Z != 30.0 {
		t.Errorf("Location: %+v", d.Ball.Location)
	}
	if len(d.Players) != 1 || d.Players[0].PrimaryId != "pid1" {
		t.Errorf("Players: %+v", d.Players)
	}
}

func TestMatchEndedDataUnmarshal(t *testing.T) {
	raw := `{"MatchGuid":"guid-123","WinnerTeamNum":1}`
	var d events.MatchEndedData
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if d.MatchGuid != "guid-123" {
		t.Errorf("MatchGuid: got %q", d.MatchGuid)
	}
	if d.WinnerTeamNum != 1 {
		t.Errorf("WinnerTeamNum: got %d", d.WinnerTeamNum)
	}
}

func TestMatchGuidDataUnmarshal(t *testing.T) {
	raw := `{"MatchGuid":"abc-def-123"}`
	var d events.MatchGuidData
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if d.MatchGuid != "abc-def-123" {
		t.Errorf("MatchGuid: got %q", d.MatchGuid)
	}
}

func TestPlayerOptionalFields(t *testing.T) {
	raw := `{
		"Name": "Alice", "PrimaryId": "pid1", "TeamNum": 0,
		"Score": 0, "Goals": 0, "Shots": 0, "Assists": 0, "Saves": 0,
		"Touches": 0, "CarTouches": 0, "Demos": 0,
		"Speed": 1500.5,
		"Boost": 85,
		"bBoosting": true,
		"bSupersonic": true,
		"bDemolished": false,
		"bOnWall": false
	}`
	var p events.Player
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if p.Speed == nil || *p.Speed != 1500.5 {
		t.Errorf("Speed: got %v", p.Speed)
	}
	if p.Boost == nil || *p.Boost != 85 {
		t.Errorf("Boost: got %v", p.Boost)
	}
	if p.BBoosting == nil || !*p.BBoosting {
		t.Errorf("BBoosting: got %v", p.BBoosting)
	}
	if p.BSupersonic == nil || !*p.BSupersonic {
		t.Errorf("BSupersonic: got %v", p.BSupersonic)
	}
}

func TestPlayerOptionalFieldsAbsent(t *testing.T) {
	raw := `{"Name":"Bob","PrimaryId":"pid2","TeamNum":1,"Score":0,"Goals":0,"Shots":0,"Assists":0,"Saves":0,"Touches":0,"CarTouches":0,"Demos":0}`
	var p events.Player
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if p.Speed != nil {
		t.Errorf("Speed should be nil, got %v", p.Speed)
	}
	if p.Boost != nil {
		t.Errorf("Boost should be nil, got %v", p.Boost)
	}
	if p.BBoosting != nil {
		t.Errorf("BBoosting should be nil, got %v", p.BBoosting)
	}
}

func TestTeamUnmarshal(t *testing.T) {
	raw := `{"Name":"Blue","TeamNum":0,"Score":3,"ColorPrimary":"#0000FF","ColorSecondary":"#00FFFF"}`
	var team events.Team
	if err := json.Unmarshal([]byte(raw), &team); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if team.Score != 3 || team.TeamNum != 0 {
		t.Errorf("Team: %+v", team)
	}
	if team.ColorPrimary != "#0000FF" {
		t.Errorf("ColorPrimary: got %q", team.ColorPrimary)
	}
}

func TestCrossbarHitDataUnmarshal(t *testing.T) {
	raw := `{
		"MatchGuid": "guid-x",
		"BallLocation": {"X": 0, "Y": 0, "Z": 200},
		"BallSpeed": 850.0,
		"ImpactForce": 10.0,
		"BallLastTouch": {"Player": {"Name": "Alice", "PrimaryId": "pid1"}, "Speed": 800.0}
	}`
	var d events.CrossbarHitData
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if d.BallSpeed != 850.0 {
		t.Errorf("BallSpeed: got %f", d.BallSpeed)
	}
	if d.BallLastTouch.Player.PrimaryId != "pid1" {
		t.Errorf("BallLastTouch.Player: got %q", d.BallLastTouch.Player.PrimaryId)
	}
}