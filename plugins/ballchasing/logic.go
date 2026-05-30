package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	sdk "github.com/erosas/oof-plugin-sdk"
)

const bcBase = "https://ballchasing.com"

var (
	apiKey            string
	deleteAfterUpload bool
	startupTime       time.Time
)

func initPlugin() uint32 {
	sdk.DBExec(`CREATE TABLE IF NOT EXISTS bc_uploads (
		replay_name    TEXT PRIMARY KEY,
		ballchasing_id TEXT NOT NULL
	)`, nil)
	startupTime = time.Now()
	return 0
}

func applySettings(data []byte) uint32 {
	var settings map[string]string
	if err := json.Unmarshal(data, &settings); err != nil {
		return 1
	}
	if v, ok := settings["ballchasing_api_key"]; ok {
		apiKey = v
	}
	if v, ok := settings["ballchasing_delete_after_upload"]; ok {
		deleteAfterUpload = sdk.ParseBool(v)
	}
	return 0
}

func onEvent(eventType string, _ []byte) {
	switch eventType {
	case "match.ended":
		msg, _ := json.Marshal(map[string]any{"Event": "bc:save-replay-reminder"})
		sdk.BroadcastWS(msg)
	}
}

func handleHTTP(req sdk.HTTPRequest) sdk.HTTPResponse {
	switch req.Path {
	case "/api/ballchasing/ping":
		return handlePing(req)
	case "/api/ballchasing/local-replays/purge":
		return handlePurgeReplays(req)
	case "/api/ballchasing/matches":
		return handleBCMatches(req)
	case "/api/ballchasing/sync":
		return handleSync(req)
	case "/api/ballchasing/replays":
		return handleReplays(req)
	case "/api/ballchasing/groups":
		return handleGroups(req)
	case "/api/ballchasing/upload":
		return handleUpload(req)
	default:
		return sdk.JSONError(404, "not found")
	}
}

// bcFetch performs a Ballchasing API request, injecting auth headers.
// Set req.URL to the API path (e.g. "/api/replays"); bcBase is prepended automatically.
func bcFetch(req sdk.HTTPFetchRequest) sdk.HTTPFetchResult {
	req.URL = bcBase + req.URL
	if req.Headers == nil {
		req.Headers = map[string]string{}
	}
	req.Headers["Authorization"] = apiKey
	if req.Headers["Accept"] == "" {
		req.Headers["Accept"] = "application/json"
	}
	return sdk.HTTPFetch(req)
}

// --- Handlers ---

func handlePing(_ sdk.HTTPRequest) sdk.HTTPResponse {
	if apiKey == "" {
		return sdk.JSONError(400, "ballchasing API key not configured")
	}
	res := bcFetch(sdk.HTTPFetchRequest{Method: "GET", URL: "/api/"})
	if res.Error != "" {
		return sdk.JSONError(502, res.Error)
	}
	if res.Status != 200 {
		return sdk.ProxyResponse(res)
	}
	var result struct {
		Name    string `json:"name"`
		SteamID string `json:"steam_id"`
		Avatar  string `json:"avatar"`
	}
	name := "(connected)"
	if json.Unmarshal([]byte(res.Body), &result) == nil && result.Name != "" {
		name = result.Name
	}
	b, _ := json.Marshal(map[string]string{"name": name, "steam_id": result.SteamID, "avatar": result.Avatar})
	return sdk.JSONResponse(b)
}

func handleReplays(req sdk.HTTPRequest) sdk.HTTPResponse {
	if apiKey == "" {
		return sdk.JSONError(400, "ballchasing API key not configured")
	}
	path := "/api/replays"
	if req.Query != "" {
		path += "?" + req.Query
	} else {
		path += "?uploader=me&count=50&sort-by=replay-date&sort-dir=desc"
	}
	return sdk.ProxyResponse(bcFetch(sdk.HTTPFetchRequest{Method: "GET", URL: path}))
}

func handleGroups(req sdk.HTTPRequest) sdk.HTTPResponse {
	if apiKey == "" {
		return sdk.JSONError(400, "ballchasing API key not configured")
	}
	path := "/api/groups"
	if req.Query != "" {
		path += "?" + req.Query
	} else {
		path += "?creator=me&count=50&sort-by=created&sort-dir=desc"
	}
	return sdk.ProxyResponse(bcFetch(sdk.HTTPFetchRequest{Method: "GET", URL: path}))
}

func handleUpload(req sdk.HTTPRequest) sdk.HTTPResponse {
	if req.Method != "POST" {
		return sdk.JSONError(405, "method not allowed")
	}
	if apiKey == "" {
		return sdk.JSONError(400, "ballchasing API key not configured")
	}

	var body struct {
		ReplayName string `json:"replay_name"`
		Visibility string `json:"visibility"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return sdk.JSONError(400, "invalid request body")
	}
	if body.ReplayName == "" {
		return sdk.JSONError(400, "replay_name required")
	}
	if body.Visibility == "" {
		body.Visibility = "unlisted"
	}
	switch body.Visibility {
	case "public", "unlisted", "private":
	default:
		return sdk.JSONError(400, "invalid visibility")
	}
	if name := filepath.Base(body.ReplayName); name == "." || name == ".." || name != body.ReplayName {
		return sdk.JSONError(400, "invalid replay name")
	}

	if _, err := os.Stat("/replays/" + body.ReplayName); err != nil {
		return sdk.JSONError(404, "replay file not found: "+body.ReplayName)
	}

	res := sdk.UploadFile(
		"/replays/"+body.ReplayName,
		bcBase+"/api/v2/upload?visibility="+body.Visibility,
		"file",
		map[string]string{
			"Authorization": apiKey,
			"Accept":        "application/json",
		},
	)
	if res.Error != "" {
		return sdk.JSONError(502, res.Error)
	}

	if res.Status == 200 || res.Status == 201 {
		var uploadResp struct {
			ID string `json:"id"`
		}
		if json.Unmarshal([]byte(res.Body), &uploadResp) == nil && uploadResp.ID != "" {
			sdk.DBExec(`INSERT INTO bc_uploads(replay_name, ballchasing_id) VALUES(?,?)
				ON CONFLICT(replay_name) DO UPDATE SET ballchasing_id=excluded.ballchasing_id`,
				[]string{body.ReplayName, uploadResp.ID})
			if deleteAfterUpload {
				os.Remove("/replays/" + body.ReplayName)
			}
		}
	}

	return sdk.ProxyResponse(res)
}

func handlePurgeReplays(req sdk.HTTPRequest) sdk.HTTPResponse {
	if req.Method != "POST" {
		return sdk.JSONError(405, "method not allowed")
	}
	rows := sdk.DBQuery(`SELECT replay_name FROM bc_uploads`, nil)
	deleted := 0
	for _, row := range rows {
		name, _ := row["replay_name"].(string)
		if name != "" {
			if err := os.Remove("/replays/" + name); err == nil {
				deleted++
			}
		}
	}
	b, _ := json.Marshal(map[string]int{"deleted": deleted})
	return sdk.JSONResponse(b)
}

func handleBCMatches(_ sdk.HTTPRequest) sdk.HTTPResponse {
	startupRFC := startupTime.UTC().Format(time.RFC3339)
	matchRows := sdk.DBQuery(`
		SELECT match_guid, COALESCE(arena,'') AS arena, started_at
		FROM hist_matches
		WHERE match_guid IS NOT NULL AND match_guid != '' AND started_at >= ?
		ORDER BY started_at DESC
		LIMIT 200`, []string{startupRFC})

	var matches []matchInfo
	for _, row := range matchRows {
		guid, _ := row["match_guid"].(string)
		arena, _ := row["arena"].(string)
		startedAtStr, _ := row["started_at"].(string)
		matches = append(matches, matchInfo{MatchGUID: guid, Arena: arena, StartedAt: sdk.ParseTime(startedAtStr)})
	}

	uploadRows := sdk.DBQuery(`SELECT replay_name, ballchasing_id FROM bc_uploads`, nil)
	uploads := make(map[string]string)
	for _, row := range uploadRows {
		name, _ := row["replay_name"].(string)
		id, _ := row["ballchasing_id"].(string)
		if name != "" {
			uploads[name] = id
		}
	}

	dirEntries, _ := os.ReadDir("/replays")
	var sessionFiles []replayFileEntry
	for _, e := range dirEntries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".replay") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if modTime := info.ModTime(); !modTime.Before(startupTime) {
			sessionFiles = append(sessionFiles, replayFileEntry{name: e.Name(), modTime: modTime})
		}
	}
	sort.Slice(sessionFiles, func(i, j int) bool { return sessionFiles[i].modTime.Before(sessionFiles[j].modTime) })

	fileForMatch := matchReplayFiles(sessionFiles, matches)

	type outRow struct {
		MatchGUID    string    `json:"match_guid"`
		Arena        string    `json:"arena"`
		StartedAt    time.Time `json:"started_at"`
		ReplayExists bool      `json:"replay_exists"`
		ReplayName   string    `json:"replay_name,omitempty"`
		Uploaded     bool      `json:"uploaded"`
		BcID         string    `json:"bc_id,omitempty"`
		BcURL        string    `json:"bc_url,omitempty"`
	}
	out := make([]outRow, 0, len(matches))
	for i, m := range matches {
		normGUID := normalizeGUID(m.MatchGUID)
		replayName := fileForMatch[i]

		bcID := uploads[replayName]
		if bcID == "" {
			bcID = uploads[normGUID+".replay"]
		}

		var bcURL string
		if bcID != "" {
			bcURL = "https://ballchasing.com/replay/" + bcID
		}
		out = append(out, outRow{
			MatchGUID:    normGUID,
			Arena:        m.Arena,
			StartedAt:    m.StartedAt,
			ReplayExists: replayName != "",
			ReplayName:   replayName,
			Uploaded:     bcID != "",
			BcID:         bcID,
			BcURL:        bcURL,
		})
	}
	b, _ := json.Marshal(out)
	return sdk.JSONResponse(b)
}

func handleSync(req sdk.HTTPRequest) sdk.HTTPResponse {
	if req.Method != "POST" {
		return sdk.JSONError(405, "method not allowed")
	}
	if apiKey == "" {
		return sdk.JSONError(400, "ballchasing API key not configured")
	}

	res := bcFetch(sdk.HTTPFetchRequest{Method: "GET", URL: "/api/replays?uploader=me&count=200&sort-by=replay-date&sort-dir=desc"})
	if res.Error != "" {
		return sdk.JSONError(502, res.Error)
	}

	var result struct {
		List []struct {
			ID             string `json:"id"`
			RocketLeagueID string `json:"rocket_league_id"`
		} `json:"list"`
	}
	if err := json.Unmarshal([]byte(res.Body), &result); err != nil {
		return sdk.JSONError(502, "failed to parse BC response")
	}

	synced := 0
	for _, rp := range result.List {
		guid := normalizeGUID(rp.RocketLeagueID)
		if guid == "" || rp.ID == "" {
			continue
		}
		if n := sdk.DBExec(`INSERT INTO bc_uploads(replay_name, ballchasing_id) VALUES(?,?)
			ON CONFLICT(replay_name) DO UPDATE SET ballchasing_id=excluded.ballchasing_id`,
			[]string{guid + ".replay", rp.ID}); n >= 0 {
			synced++
		}
	}
	b, _ := json.Marshal(map[string]int{"synced": synced})
	return sdk.JSONResponse(b)
}

// --- Helpers ---

type matchInfo struct {
	MatchGUID string
	Arena     string
	StartedAt time.Time
}

type replayFileEntry struct {
	name    string
	modTime time.Time
}

func matchReplayFiles(files []replayFileEntry, matches []matchInfo) map[int]string {
	result := make(map[int]string)
	const window = 30 * time.Minute
	for _, f := range files {
		bestIdx := -1
		for i, m := range matches {
			if _, taken := result[i]; taken {
				continue
			}
			if m.StartedAt.Before(f.modTime) && f.modTime.Before(m.StartedAt.Add(window)) {
				if bestIdx == -1 || m.StartedAt.After(matches[bestIdx].StartedAt) {
					bestIdx = i
				}
			}
		}
		if bestIdx >= 0 {
			result[bestIdx] = f.name
		}
	}
	return result
}

func normalizeGUID(s string) string {
	return strings.ToUpper(strings.ReplaceAll(s, "-", ""))
}
