//go:build !wasip1

package main

import (
	"encoding/json"
	"testing"
)

func reset() {
	mu.Lock()
	state = nil
	mu.Unlock()
}

func TestOnEventStateUpdatedStoresState(t *testing.T) {
	reset()

	payload, _ := json.Marshal(stateUpdatedPayload{
		GUID:    "match-1",
		Players: json.RawMessage(`[{"Name":"Alice","TeamNum":0}]`),
		Game:    json.RawMessage(`{"TimeSeconds":120}`),
	})
	onEvent("state.updated", payload)

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