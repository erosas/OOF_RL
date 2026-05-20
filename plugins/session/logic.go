package main

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	sdk "github.com/erosas/oof-plugin-sdk"
)

var (
	mu    sync.RWMutex
	since time.Time
)

func initPlugin() uint32 {
	sdk.DBExec(`CREATE TABLE IF NOT EXISTS sessions (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		player_id  TEXT NOT NULL,
		started_at DATETIME NOT NULL,
		ended_at   DATETIME NOT NULL
	)`, nil)
	sdk.DBExec(`CREATE INDEX IF NOT EXISTS idx_sessions_player ON sessions(player_id, started_at)`, nil)
	return 0
}

func onEvent(eventType string) {
	if eventType == "match.started" {
		mu.Lock()
		if since.IsZero() {
			since = time.Now()
		}
		mu.Unlock()
	}
}

func handleHTTP(req sdk.HTTPRequest) sdk.HTTPResponse {
	switch req.Path {
	case "/api/session/stats":
		return handleStats(req)
	case "/api/session/start":
		return handleStart(req)
	case "/api/session/new":
		return handleNew(req)
	case "/api/session/suggest-player":
		return handleSuggestPlayer(req)
	case "/api/session/history":
		return handleHistory(req)
	default:
		if strings.HasPrefix(req.Path, "/api/session/history/") {
			return handleHistoryItem(req)
		}
		return jsonError(404, "not found")
	}
}

func handleStats(req sdk.HTTPRequest) sdk.HTTPResponse {
	playerID := queryParam(req.Query, "player")
	if playerID == "" {
		return jsonError(400, "player parameter required")
	}

	mu.RLock()
	s := since
	mu.RUnlock()

	if s.IsZero() {
		b, _ := json.Marshal(map[string]any{"matches": []any{}, "summary": struct{}{}})
		return jsonOK(b)
	}

	matches, _ := sessionMatchesByPlayer(s, playerID)

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
	sum := summary{}
	for _, m := range matches {
		sum.Games++
		sum.Goals += m.Goals
		sum.Assists += m.Assists
		sum.Saves += m.Saves
		sum.Shots += m.Shots
		sum.Demos += m.Demos
		if m.WinnerTeamNum >= 0 {
			if m.PlayerTeam == m.WinnerTeamNum {
				sum.Wins++
			} else {
				sum.Losses++
			}
		}
	}
	if matches == nil {
		matches = []SessionMatch{}
	}
	b, _ := json.Marshal(map[string]any{"matches": matches, "summary": sum})
	return jsonOK(b)
}

func handleStart(req sdk.HTTPRequest) sdk.HTTPResponse {
	switch req.Method {
	case "GET":
		mu.RLock()
		s := since
		mu.RUnlock()
		b, _ := json.Marshal(map[string]any{
			"since":  s.UTC().Format(time.RFC3339),
			"active": !s.IsZero(),
		})
		return jsonOK(b)

	case "POST":
		var body struct {
			Since string `json:"since"`
		}
		json.Unmarshal([]byte(req.Body), &body) //nolint

		mu.Lock()
		if body.Since != "" {
			t, err := time.Parse(time.RFC3339Nano, body.Since)
			if err != nil {
				mu.Unlock()
				return jsonError(400, "invalid since, use RFC3339")
			}
			since = t
		} else {
			since = time.Now()
		}
		s := since
		mu.Unlock()

		b, _ := json.Marshal(map[string]any{
			"since":  s.UTC().Format(time.RFC3339),
			"active": true,
		})
		return jsonOK(b)

	default:
		return jsonError(405, "method not allowed")
	}
}

func handleNew(req sdk.HTTPRequest) sdk.HTTPResponse {
	if req.Method != "POST" {
		return jsonError(405, "method not allowed")
	}
	var body struct {
		PlayerID string `json:"player_id"`
	}
	json.Unmarshal([]byte(req.Body), &body) //nolint

	mu.Lock()
	oldSince := since
	since = time.Now()
	newSince := since
	mu.Unlock()

	if body.PlayerID != "" {
		now := time.Now()
		sdk.DBExec(
			`INSERT INTO sessions(player_id, started_at, ended_at) VALUES(?,?,?)`,
			[]string{body.PlayerID, oldSince.UTC().Format(time.RFC3339), now.UTC().Format(time.RFC3339)},
		)
	}

	b, _ := json.Marshal(map[string]any{
		"since":  newSince.UTC().Format(time.RFC3339),
		"active": true,
	})
	return jsonOK(b)
}

func handleHistory(req sdk.HTTPRequest) sdk.HTTPResponse {
	if req.Method != "GET" {
		return jsonError(405, "method not allowed")
	}
	playerID := queryParam(req.Query, "player")
	if playerID == "" {
		return jsonError(400, "player parameter required")
	}
	sessions := listSessionsWithStats(playerID)
	if sessions == nil {
		sessions = []SavedSession{}
	}
	b, _ := json.Marshal(sessions)
	return jsonOK(b)
}

func handleHistoryItem(req sdk.HTTPRequest) sdk.HTTPResponse {
	idStr := strings.TrimSuffix(strings.TrimPrefix(req.Path, "/api/session/history/"), "/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return jsonError(400, "invalid id")
	}
	idArg := strconv.FormatInt(id, 10)

	switch req.Method {
	case "DELETE":
		if sdk.DBExec(`DELETE FROM sessions WHERE id=?`, []string{idArg}) < 0 {
			return jsonError(500, "delete failed")
		}
		b, _ := json.Marshal(map[string]string{"status": "ok"})
		return jsonOK(b)

	case "PUT":
		var body struct {
			StartedAt string `json:"started_at"`
			EndedAt   string `json:"ended_at"`
		}
		if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
			return jsonError(400, err.Error())
		}
		startedAt, err1 := time.Parse(time.RFC3339Nano, body.StartedAt)
		endedAt, err2 := time.Parse(time.RFC3339Nano, body.EndedAt)
		if err1 != nil || err2 != nil {
			return jsonError(400, "invalid time format, use RFC3339")
		}
		if !endedAt.After(startedAt) {
			return jsonError(400, "ended_at must be after started_at")
		}
		if sdk.DBExec(
			`UPDATE sessions SET started_at=?, ended_at=? WHERE id=?`,
			[]string{startedAt.UTC().Format(time.RFC3339), endedAt.UTC().Format(time.RFC3339), idArg},
		) < 0 {
			return jsonError(500, "update failed")
		}
		b, _ := json.Marshal(map[string]string{"status": "ok"})
		return jsonOK(b)

	default:
		return jsonError(405, "method not allowed")
	}
}

func handleSuggestPlayer(_ sdk.HTTPRequest) sdk.HTTPResponse {
	rows := sdk.DBQuery(`
		SELECT s.primary_id, pl.name
		FROM hist_player_match_stats s
		JOIN hist_players pl ON pl.primary_id = s.primary_id
		GROUP BY s.primary_id
		ORDER BY COUNT(*) DESC
		LIMIT 1`, nil)

	if len(rows) == 0 {
		b, _ := json.Marshal(map[string]string{"primary_id": "", "name": ""})
		return jsonOK(b)
	}
	b, _ := json.Marshal(map[string]string{
		"primary_id": rowStr(rows[0], "primary_id"),
		"name":       rowStr(rows[0], "name"),
	})
	return jsonOK(b)
}

// --- Models ---

type SavedSession struct {
	ID        int64  `json:"id"`
	PlayerID  string `json:"player_id"`
	StartedAt string `json:"started_at"`
	EndedAt   string `json:"ended_at"`
	Games     int    `json:"games"`
	Wins      int    `json:"wins"`
	Losses    int    `json:"losses"`
	Goals     int    `json:"goals"`
	Assists   int    `json:"assists"`
	Saves     int    `json:"saves"`
	Shots     int    `json:"shots"`
	Demos     int    `json:"demos"`
}

type SessionMatch struct {
	MatchID       int64  `json:"match_id"`
	Arena         string `json:"arena"`
	StartedAt     string `json:"started_at"`
	WinnerTeamNum int    `json:"winner_team_num"`
	Incomplete    bool   `json:"incomplete"`
	Forfeit       bool   `json:"forfeit"`
	PlayerTeam    int    `json:"player_team"`
	Goals         int    `json:"goals"`
	Assists       int    `json:"assists"`
	Saves         int    `json:"saves"`
	Shots         int    `json:"shots"`
	Demos         int    `json:"demos"`
	Score         int    `json:"score"`
	PlaylistType  *int   `json:"playlist_type"`
	PlayerCount   int    `json:"player_count"`
}

// --- DB helpers ---

func listSessionsWithStats(playerID string) []SavedSession {
	rows := sdk.DBQuery(
		`SELECT id, player_id, started_at, ended_at FROM sessions WHERE player_id=? ORDER BY started_at DESC LIMIT 50`,
		[]string{playerID})

	var sessions []SavedSession
	for _, row := range rows {
		sess := SavedSession{
			ID:        rowInt(row, "id"),
			PlayerID:  rowStr(row, "player_id"),
			StartedAt: rowStr(row, "started_at"),
			EndedAt:   rowStr(row, "ended_at"),
		}
		start := parseTime(sess.StartedAt)
		end := parseTime(sess.EndedAt)
		for _, m := range sessionMatchesBetween(start, end, playerID) {
			sess.Games++
			sess.Goals += m.Goals
			sess.Assists += m.Assists
			sess.Saves += m.Saves
			sess.Shots += m.Shots
			sess.Demos += m.Demos
			if !m.Incomplete && m.WinnerTeamNum >= 0 {
				if m.PlayerTeam == m.WinnerTeamNum {
					sess.Wins++
				} else {
					sess.Losses++
				}
			}
		}
		sessions = append(sessions, sess)
	}
	return sessions
}

func sessionMatchesByPlayer(start time.Time, playerID string) ([]SessionMatch, error) {
	return sessionMatchesBetween(start, time.Time{}, playerID), nil
}

func sessionMatchesBetween(start, end time.Time, playerID string) []SessionMatch {
	rows := sdk.DBQuery(`
		SELECT m.id AS match_id,
		       COALESCE(m.arena,'') AS arena,
		       m.started_at AS started_at,
		       COALESCE(m.winner_team_num,-1) AS winner_team_num,
		       COALESCE(m.incomplete,0) AS incomplete,
		       COALESCE(m.forfeit,0) AS forfeit,
		       s.team_num AS team_num,
		       s.goals AS goals,
		       s.assists AS assists,
		       s.saves AS saves,
		       s.shots AS shots,
		       s.demos AS demos,
		       s.score AS score,
		       m.playlist_type AS playlist_type,
		       (SELECT COUNT(*) FROM hist_player_match_stats WHERE match_id = m.id) AS player_count
		FROM hist_matches m
		JOIN hist_player_match_stats s ON s.match_id = m.id
		WHERE s.primary_id = ?
		ORDER BY m.started_at ASC`,
		[]string{playerID})

	var out []SessionMatch
	for _, row := range rows {
		startedAtStr := rowStr(row, "started_at")
		startedAt := parseTime(startedAtStr)
		if startedAt.Before(start) {
			continue
		}
		if !end.IsZero() && !startedAt.Before(end) {
			continue
		}
		m := SessionMatch{
			MatchID:       rowInt(row, "match_id"),
			Arena:         rowStr(row, "arena"),
			StartedAt:     startedAtStr,
			WinnerTeamNum: int(rowInt(row, "winner_team_num")),
			Incomplete:    rowBool(row, "incomplete"),
			Forfeit:       rowBool(row, "forfeit"),
			PlayerTeam:    int(rowInt(row, "team_num")),
			Goals:         int(rowInt(row, "goals")),
			Assists:       int(rowInt(row, "assists")),
			Saves:         int(rowInt(row, "saves")),
			Shots:         int(rowInt(row, "shots")),
			Demos:         int(rowInt(row, "demos")),
			Score:         int(rowInt(row, "score")),
			PlayerCount:   int(rowInt(row, "player_count")),
		}
		if pt, ok := row["playlist_type"].(float64); ok {
			v := int(pt)
			m.PlaylistType = &v
		}
		out = append(out, m)
	}
	return out
}

// --- Utilities ---

func rowInt(row map[string]any, key string) int64 {
	v, _ := row[key].(float64)
	return int64(v)
}

func rowStr(row map[string]any, key string) string {
	v, _ := row[key].(string)
	return v
}

func rowBool(row map[string]any, key string) bool {
	v, _ := row[key].(float64)
	return v != 0
}

func parseTime(s string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func queryParam(query, key string) string {
	vals, err := url.ParseQuery(query)
	if err != nil {
		return ""
	}
	return vals.Get(key)
}

func jsonOK(body []byte) sdk.HTTPResponse {
	return sdk.HTTPResponse{
		Status:  200,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    string(body),
	}
}

func jsonError(status int, msg string) sdk.HTTPResponse {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return sdk.HTTPResponse{
		Status:  status,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    string(b),
	}
}