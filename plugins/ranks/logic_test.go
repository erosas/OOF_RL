//go:build !wasip1

package main

import (
	"encoding/json"
	"testing"
)

func reset() {
	mu.Lock()
	players = nil
	mu.Unlock()
}

func TestOnEventStateUpdatedStorePlayers(t *testing.T) {
	reset()

	payload, _ := json.Marshal(stateUpdatedPayload{
		Players: []struct {
			PrimaryID string `json:"PrimaryId"`
			Name      string `json:"Name"`
			TeamNum   int    `json:"TeamNum"`
		}{
			{PrimaryID: "abc", Name: "Alice", TeamNum: 0},
			{PrimaryID: "def", Name: "Bob", TeamNum: 1},
		},
	})
	onEvent("state.updated", payload)

	mu.RLock()
	ps := players
	mu.RUnlock()

	if len(ps) != 2 {
		t.Fatalf("expected 2 players, got %d", len(ps))
	}
	if ps[0].PrimaryID != "abc" || ps[0].Name != "Alice" || ps[0].TeamNum != 0 {
		t.Errorf("player[0]: got %+v", ps[0])
	}
	if ps[1].PrimaryID != "def" || ps[1].Name != "Bob" || ps[1].TeamNum != 1 {
		t.Errorf("player[1]: got %+v", ps[1])
	}
}

func TestOnEventMatchDestroyedClearsPlayers(t *testing.T) {
	mu.Lock()
	players = []rankPlayer{{PrimaryID: "abc", Name: "Alice", TeamNum: 0}}
	mu.Unlock()

	onEvent("match.destroyed", nil)

	mu.RLock()
	ps := players
	mu.RUnlock()
	if ps != nil {
		t.Fatal("players should be nil after match.destroyed")
	}
}

func TestOnEventUnknownTypeIsIgnored(t *testing.T) {
	reset()
	onEvent("unknown.event", []byte(`{}`))

	mu.RLock()
	ps := players
	mu.RUnlock()
	if ps != nil {
		t.Fatal("unknown event should not affect state")
	}
}

func TestHandleHTTPNoActivePlayers(t *testing.T) {
	reset()
	resp := handleHTTP()

	if resp.Status != 200 {
		t.Errorf("status: got %d, want 200", resp.Status)
	}
	var body []rankPlayer
	if err := json.Unmarshal([]byte(resp.Body), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body) != 0 {
		t.Errorf("expected empty array, got %v", body)
	}
}

func TestHandleHTTPWithPlayers(t *testing.T) {
	mu.Lock()
	players = []rankPlayer{
		{PrimaryID: "abc", Name: "Alice", TeamNum: 0},
		{PrimaryID: "def", Name: "Bob", TeamNum: 1},
	}
	mu.Unlock()

	resp := handleHTTP()

	if resp.Status != 200 {
		t.Errorf("status: got %d, want 200", resp.Status)
	}
	var body []rankPlayer
	if err := json.Unmarshal([]byte(resp.Body), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body) != 2 {
		t.Fatalf("expected 2 players, got %d", len(body))
	}
}