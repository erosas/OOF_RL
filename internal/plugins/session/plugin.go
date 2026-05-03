package session

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"OOF_RL/internal/db"
	"OOF_RL/internal/events"
	"OOF_RL/internal/httputil"
	"OOF_RL/internal/plugin"
)

//go:embed view.html view.js
var viewFS embed.FS

const sessionSchema = `
CREATE TABLE IF NOT EXISTS sessions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id  TEXT NOT NULL,
    started_at DATETIME NOT NULL,
    ended_at   DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_player ON sessions(player_id, started_at);
`

type Plugin struct {
	db    *db.DB
	mu    sync.Mutex
	since time.Time
}

func New(database *db.DB) *Plugin {
	if err := database.RunMigration(sessionSchema); err != nil {
		log.Printf("[session] migrate: %v", err)
	}
	return &Plugin{db: database, since: time.Now()}
}

func (p *Plugin) ID() string         { return "session" }
func (p *Plugin) DBPrefix() string   { return "" }
func (p *Plugin) Requires() []string { return []string{"history"} }

func (p *Plugin) NavTab() plugin.NavTab {
	return plugin.NavTab{ID: "session", Label: "Session", Order: 25}
}

func (p *Plugin) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/session/stats", p.handleStats)
	mux.HandleFunc("/api/session/start", p.handleStart)
	mux.HandleFunc("/api/session/new", p.handleNew)
	mux.HandleFunc("/api/session/suggest-player", p.handleSuggestPlayer)
	mux.HandleFunc("/api/session/history/", p.handleHistoryItem)
	mux.HandleFunc("/api/session/history", p.handleHistory)
}

func (p *Plugin) SettingsSchema() []plugin.Setting        { return nil }
func (p *Plugin) ApplySettings(_ map[string]string) error { return nil }
func (p *Plugin) HandleEvent(_ events.Envelope)           {}
func (p *Plugin) Assets() fs.FS                           { return viewFS }

// handleStart: GET returns the current session start time; POST updates it via datetime picker.
func (p *Plugin) handleStart(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		p.mu.Lock()
		since := p.since
		p.mu.Unlock()
		httputil.WriteJSON(w, map[string]string{"since": since.UTC().Format(time.RFC3339)})

	case http.MethodPost:
		var req struct {
			Since string `json:"since"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		p.mu.Lock()
		if req.Since != "" {
			t, err := time.Parse(time.RFC3339Nano, req.Since)
			if err != nil {
				p.mu.Unlock()
				httputil.JSONError(w, 400, "invalid since, use RFC3339")
				return
			}
			p.since = t
		} else {
			p.since = time.Now()
		}
		since := p.since
		p.mu.Unlock()
		httputil.WriteJSON(w, map[string]string{"since": since.UTC().Format(time.RFC3339)})

	default:
		http.Error(w, "method not allowed", 405)
	}
}

// handleNew saves the current session to history (if a player is provided) then resets since to now.
func (p *Plugin) handleNew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req struct {
		PlayerID string `json:"player_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	p.mu.Lock()
	oldSince := p.since
	p.since = time.Now()
	newSince := p.since
	p.mu.Unlock()

	if req.PlayerID != "" {
		if _, err := p.db.CreateSession(req.PlayerID, oldSince, time.Now()); err != nil {
			log.Printf("[session] save history: %v", err)
		}
	}
	httputil.WriteJSON(w, map[string]string{"since": newSince.UTC().Format(time.RFC3339)})
}

// handleHistory: GET returns all saved sessions with aggregate stats for a player.
func (p *Plugin) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	playerID := r.URL.Query().Get("player")
	if playerID == "" {
		httputil.JSONError(w, 400, "player parameter required")
		return
	}
	sessions, err := p.db.ListSessionsWithStats(playerID)
	if err != nil {
		httputil.JSONError(w, 500, err.Error())
		return
	}
	if sessions == nil {
		sessions = []db.SavedSession{}
	}
	httputil.WriteJSON(w, sessions)
}

// handleHistoryItem: DELETE removes a session; PUT updates its start/end times.
func (p *Plugin) handleHistoryItem(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/session/history/")
	id, err := strconv.ParseInt(strings.TrimSuffix(idStr, "/"), 10, 64)
	if err != nil {
		httputil.JSONError(w, 400, "invalid id")
		return
	}

	switch r.Method {
	case http.MethodDelete:
		if err := p.db.DeleteSession(id); err != nil {
			httputil.JSONError(w, 500, err.Error())
			return
		}
		httputil.WriteJSON(w, map[string]string{"status": "ok"})

	case http.MethodPut:
		var req struct {
			StartedAt string `json:"started_at"`
			EndedAt   string `json:"ended_at"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.JSONError(w, 400, err.Error())
			return
		}
		startedAt, err1 := time.Parse(time.RFC3339Nano, req.StartedAt)
		endedAt, err2 := time.Parse(time.RFC3339Nano, req.EndedAt)
		if err1 != nil || err2 != nil {
			httputil.JSONError(w, 400, "invalid time format, use RFC3339")
			return
		}
		if !endedAt.After(startedAt) {
			httputil.JSONError(w, 400, "ended_at must be after started_at")
			return
		}
		if err := p.db.UpdateSession(id, startedAt, endedAt); err != nil {
			httputil.JSONError(w, 500, err.Error())
			return
		}
		httputil.WriteJSON(w, map[string]string{"status": "ok"})

	default:
		http.Error(w, "method not allowed", 405)
	}
}

// handleSuggestPlayer returns the player who appears in the most matches.
func (p *Plugin) handleSuggestPlayer(w http.ResponseWriter, r *http.Request) {
	player, err := p.db.MostFrequentPlayer()
	if err != nil {
		httputil.JSONError(w, 500, err.Error())
		return
	}
	if player == nil {
		httputil.WriteJSON(w, map[string]string{"primary_id": "", "name": ""})
		return
	}
	httputil.WriteJSON(w, map[string]string{"primary_id": player.PrimaryID, "name": player.Name})
}

func (p *Plugin) handleStats(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("player")
	if playerID == "" {
		httputil.JSONError(w, 400, "player parameter required")
		return
	}

	p.mu.Lock()
	since := p.since
	p.mu.Unlock()

	matches, err := p.db.SessionMatchesByPlayer(since, playerID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	type summary struct {
		Games   int `json:"games"`
		Wins    int `json:"wins"`
		Losses  int `json:"losses"`
		Goals   int `json:"goals"`
		Assists int `json:"assists"`
		Saves   int `json:"saves"`
		Shots   int `json:"shots"`
		Demos   int `json:"demos"`
	}
	s := summary{}
	for _, m := range matches {
		s.Games++
		s.Goals += m.Goals
		s.Assists += m.Assists
		s.Saves += m.Saves
		s.Shots += m.Shots
		s.Demos += m.Demos
		if m.WinnerTeamNum >= 0 {
			if m.PlayerTeam == m.WinnerTeamNum {
				s.Wins++
			} else {
				s.Losses++
			}
		}
	}

	if matches == nil {
		matches = []db.SessionMatch{}
	}
	httputil.WriteJSON(w, map[string]any{
		"matches": matches,
		"summary": s,
	})
}