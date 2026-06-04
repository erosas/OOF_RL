//go:build !wasip1

package main

import (
	"encoding/json"
	"testing"
)

func reset() {
	mu.Lock()
	resetStateLocked()
	mu.Unlock()
}

func statePayload(guid, players, game string) []byte {
	payload, _ := json.Marshal(stateUpdatedPayload{
		GUID:    guid,
		Players: json.RawMessage(players),
		Game:    json.RawMessage(game),
	})
	return payload
}

func currentPlayers(t *testing.T) []map[string]any {
	t.Helper()
	mu.RLock()
	s := state
	mu.RUnlock()
	if s == nil {
		t.Fatal("state should be active")
	}
	var players []map[string]any
	if err := json.Unmarshal(s.Players, &players); err != nil {
		t.Fatalf("Players json: %v", err)
	}
	return players
}

func currentGame(t *testing.T) map[string]any {
	t.Helper()
	mu.RLock()
	s := state
	mu.RUnlock()
	if s == nil {
		t.Fatal("state should be active")
	}
	var game map[string]any
	if err := json.Unmarshal(s.Game, &game); err != nil {
		t.Fatalf("Game json: %v", err)
	}
	return game
}

func numberField(t *testing.T, row map[string]any, key string) float64 {
	t.Helper()
	v, ok := row[key].(float64)
	if !ok {
		t.Fatalf("%s: got %T, want number", key, row[key])
	}
	return v
}

func TestOnEventStateUpdatedStoresState(t *testing.T) {
	reset()

	onEvent("state.updated", statePayload("match-1", `[{"Name":"Alice","TeamNum":0}]`, `{"TimeSeconds":120}`))

	mu.RLock()
	s := state
	mu.RUnlock()

	if s == nil {
		t.Fatal("state should be set after state.updated")
	}
	if s.MatchGUID != "match-1" {
		t.Errorf("MatchGUID: got %q, want match-1", s.MatchGUID)
	}
}

func TestReplayStatePreservesCachedLiveTouches(t *testing.T) {
	reset()

	onEvent("state.updated", statePayload("match-1",
		`[{"Name":"Alice","PrimaryId":"steam|alice","Shortcut":1,"TeamNum":0,"Touches":3,"CarTouches":2,"Goals":0,"Score":100}]`,
		`{"TimeSeconds":180,"bReplay":false}`))
	onEvent("state.updated", statePayload("match-1",
		`[{"Name":"Alice","PrimaryId":"steam|alice","Shortcut":1,"TeamNum":0,"Touches":11,"CarTouches":9,"Goals":1,"Score":220}]`,
		`{"TimeSeconds":175,"bReplay":true}`))

	players := currentPlayers(t)
	if got := numberField(t, players[0], "Touches"); got != 3 {
		t.Fatalf("Touches: got %v, want cached live value 3", got)
	}
	if got := numberField(t, players[0], "CarTouches"); got != 2 {
		t.Fatalf("CarTouches: got %v, want cached live value 2", got)
	}
	if got := numberField(t, players[0], "Goals"); got != 1 {
		t.Fatalf("Goals should still update from replay state, got %v", got)
	}
	if got := numberField(t, players[0], "Score"); got != 220 {
		t.Fatalf("Score should still update from replay state, got %v", got)
	}

	game := currentGame(t)
	if game["bReplay"] != true {
		t.Fatalf("replay flag should stay true, got %v", game["bReplay"])
	}
}

func TestReplayStateLeavesUnknownPlayerTouchesUnchanged(t *testing.T) {
	reset()

	onEvent("state.updated", statePayload("match-1",
		`[{"Name":"Bob","PrimaryId":"steam|bob","Shortcut":2,"TeamNum":1,"Touches":7,"CarTouches":6}]`,
		`{"TimeSeconds":175,"bReplay":true}`))

	players := currentPlayers(t)
	if got := numberField(t, players[0], "Touches"); got != 7 {
		t.Fatalf("unknown replay player touches should stay unchanged, got %v", got)
	}
	if got := numberField(t, players[0], "CarTouches"); got != 6 {
		t.Fatalf("unknown replay player car touches should stay unchanged, got %v", got)
	}
}

func TestMatchDestroyedClearsTouchCache(t *testing.T) {
	reset()

	onEvent("state.updated", statePayload("match-1",
		`[{"Name":"Alice","PrimaryId":"steam|alice","Shortcut":1,"TeamNum":0,"Touches":3,"CarTouches":2}]`,
		`{"TimeSeconds":180,"bReplay":false}`))
	onEvent("match.destroyed", nil)
	onEvent("state.updated", statePayload("match-1",
		`[{"Name":"Alice","PrimaryId":"steam|alice","Shortcut":1,"TeamNum":0,"Touches":12,"CarTouches":10}]`,
		`{"TimeSeconds":175,"bReplay":true}`))

	players := currentPlayers(t)
	if got := numberField(t, players[0], "Touches"); got != 12 {
		t.Fatalf("replay touches after destroy should not use old cache, got %v", got)
	}
}

func TestMatchGUIDChangeResetsTouchCache(t *testing.T) {
	reset()

	onEvent("state.updated", statePayload("match-1",
		`[{"Name":"Alice","PrimaryId":"steam|alice","Shortcut":1,"TeamNum":0,"Touches":3,"CarTouches":2}]`,
		`{"TimeSeconds":180,"bReplay":false}`))
	onEvent("state.updated", statePayload("match-2",
		`[{"Name":"Alice","PrimaryId":"steam|alice","Shortcut":1,"TeamNum":0,"Touches":12,"CarTouches":10}]`,
		`{"TimeSeconds":175,"bReplay":true}`))

	players := currentPlayers(t)
	if got := numberField(t, players[0], "Touches"); got != 12 {
		t.Fatalf("new match replay touches should not use previous match cache, got %v", got)
	}
}

func TestOnEventUnknownTypeIsIgnored(t *testing.T) {
	reset()
	onEvent("unknown.event", []byte(`{}`))

	mu.RLock()
	s := state
	mu.RUnlock()
	if s != nil {
		t.Fatal("unknown event should not affect state")
	}
}

func TestOnEventMatchDestroyedClearsState(t *testing.T) {
	mu.Lock()
	state = &liveState{MatchGUID: "match-1"}
	mu.Unlock()

	onEvent("match.destroyed", nil)

	mu.RLock()
	s := state
	mu.RUnlock()
	if s != nil {
		t.Fatal("state should be nil after match.destroyed")
	}
}

func TestHandleHTTPNoActiveMatch(t *testing.T) {
	reset()
	resp := handleHTTP()

	if resp.Status != 200 {
		t.Errorf("status: got %d, want 200", resp.Status)
	}
	var body map[string]any
	json.Unmarshal([]byte(resp.Body), &body)
	if body["active"] != false {
		t.Errorf("expected active:false, got %v", body)
	}
}

func TestHandleHTTPWithActiveMatch(t *testing.T) {
	mu.Lock()
	state = &liveState{
		MatchGUID: "match-2",
		Players:   json.RawMessage(`[]`),
		Game:      json.RawMessage(`{}`),
	}
	mu.Unlock()

	resp := handleHTTP()

	if resp.Status != 200 {
		t.Errorf("status: got %d, want 200", resp.Status)
	}
	var body map[string]any
	json.Unmarshal([]byte(resp.Body), &body)
	if body["active"] != true {
		t.Errorf("expected active:true, got %v", body)
	}
	if body["state"] == nil {
		t.Error("expected state field in response")
	}
}
