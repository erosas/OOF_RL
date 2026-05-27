package main

import (
	"encoding/json"
	"sync"

	sdk "github.com/erosas/oof-plugin-sdk"
)

var (
	mu    sync.RWMutex
	state *liveState
)

type liveState struct {
	MatchGUID string          `json:"MatchGuid"`
	Players   json.RawMessage `json:"Players"`
	Game      json.RawMessage `json:"Game"`
}

// stateUpdatedPayload mirrors the JSON shape of oofevents.StateUpdatedEvent
// as serialized by the host (exported Go field names).
type stateUpdatedPayload struct {
	GUID    string          `json:"GUID"`
	Players json.RawMessage `json:"Players"`
	Game    json.RawMessage `json:"Game"`
}

func onEvent(eventType string, payload []byte) {
	switch eventType {
	case "state.updated":
		var ev stateUpdatedPayload
		if err := json.Unmarshal(payload, &ev); err != nil {
			sdk.Log("live: parse state.updated: " + err.Error())
			return
		}
		mu.Lock()
		state = &liveState{MatchGUID: ev.GUID, Players: ev.Players, Game: ev.Game}
		mu.Unlock()

	case "match.destroyed":
		mu.Lock()
		state = nil
		mu.Unlock()
	}
}

func handleHTTP() sdk.HTTPResponse {
	mu.RLock()
	s := state
	mu.RUnlock()

	var body []byte
	if s == nil {
		body, _ = json.Marshal(map[string]any{"active": false})
	} else {
		body, _ = json.Marshal(map[string]any{"active": true, "state": s})
	}
	return sdk.JSONResponse(body)
}