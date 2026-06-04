package main

import (
	"encoding/json"
	"fmt"
	"sync"

	sdk "github.com/erosas/oof-plugin-sdk"
)

var (
	mu             sync.RWMutex
	state          *liveState
	touchCacheGUID string
	touchCache     map[string]touchSnapshot
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

type gamePayload struct {
	IsReplay bool `json:"bReplay"`
}

type touchSnapshot struct {
	Touches    any
	CarTouches any
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
		state = normalizeStateUpdated(ev)
		mu.Unlock()

	case "match.destroyed":
		mu.Lock()
		resetStateLocked()
		mu.Unlock()
	}
}

func normalizeStateUpdated(ev stateUpdatedPayload) *liveState {
	if ev.GUID != touchCacheGUID {
		touchCacheGUID = ev.GUID
		touchCache = map[string]touchSnapshot{}
	}

	var game gamePayload
	if err := json.Unmarshal(ev.Game, &game); err != nil {
		sdk.Log("live: parse state.updated game: " + err.Error())
		return &liveState{MatchGUID: ev.GUID, Players: ev.Players, Game: ev.Game}
	}

	var players []map[string]any
	if err := json.Unmarshal(ev.Players, &players); err != nil {
		sdk.Log("live: parse state.updated players: " + err.Error())
		return &liveState{MatchGUID: ev.GUID, Players: ev.Players, Game: ev.Game}
	}

	if game.IsReplay {
		players = applyLiveTouchCache(players)
		b, err := json.Marshal(players)
		if err != nil {
			sdk.Log("live: encode replay players: " + err.Error())
			return &liveState{MatchGUID: ev.GUID, Players: ev.Players, Game: ev.Game}
		}
		return &liveState{MatchGUID: ev.GUID, Players: json.RawMessage(b), Game: ev.Game}
	}

	refreshLiveTouchCache(players)
	return &liveState{MatchGUID: ev.GUID, Players: ev.Players, Game: ev.Game}
}

func refreshLiveTouchCache(players []map[string]any) {
	if touchCache == nil {
		touchCache = map[string]touchSnapshot{}
	}
	for _, pl := range players {
		key := playerCacheKey(pl)
		if key == "" {
			continue
		}
		touchCache[key] = touchSnapshot{
			Touches:    pl["Touches"],
			CarTouches: pl["CarTouches"],
		}
	}
}

func applyLiveTouchCache(players []map[string]any) []map[string]any {
	for _, pl := range players {
		key := playerCacheKey(pl)
		if cached, ok := touchCache[key]; ok {
			pl["Touches"] = cached.Touches
			pl["CarTouches"] = cached.CarTouches
		}
	}
	return players
}

func playerCacheKey(pl map[string]any) string {
	if primaryID, ok := pl["PrimaryId"].(string); ok && primaryID != "" {
		return primaryID
	}
	return fmt.Sprintf("%v:%v:%v", pl["TeamNum"], pl["Shortcut"], pl["Name"])
}

func resetStateLocked() {
	state = nil
	touchCacheGUID = ""
	touchCache = nil
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
