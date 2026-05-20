package main

import (
	"encoding/json"
	"sync"

	sdk "github.com/erosas/oof-plugin-sdk"
)

var (
	mu      sync.RWMutex
	players []rankPlayer
)

type rankPlayer struct {
	PrimaryID string `json:"primary_id"`
	Name      string `json:"name"`
	TeamNum   int    `json:"team_num"`
}

type stateUpdatedPayload struct {
	Players []struct {
		PrimaryID string `json:"PrimaryId"`
		Name      string `json:"Name"`
		TeamNum   int    `json:"TeamNum"`
	} `json:"Players"`
}

func onEvent(eventType string, payload []byte) {
	switch eventType {
	case "state.updated":
		var ev stateUpdatedPayload
		if err := json.Unmarshal(payload, &ev); err != nil {
			sdk.Log("ranks: parse state.updated: " + err.Error())
			return
		}
		out := make([]rankPlayer, 0, len(ev.Players))
		for _, p := range ev.Players {
			out = append(out, rankPlayer{
				PrimaryID: p.PrimaryID,
				Name:      p.Name,
				TeamNum:   p.TeamNum,
			})
		}
		mu.Lock()
		players = out
		mu.Unlock()

	case "match.destroyed":
		mu.Lock()
		players = nil
		mu.Unlock()
	}
}

func handleHTTP() sdk.HTTPResponse {
	mu.RLock()
	ps := players
	mu.RUnlock()

	var body []byte
	if ps == nil {
		body, _ = json.Marshal([]rankPlayer{})
	} else {
		body, _ = json.Marshal(ps)
	}

	return sdk.HTTPResponse{
		Status:  200,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    string(body),
	}
}