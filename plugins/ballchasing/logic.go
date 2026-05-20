package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
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
		deleteAfterUpload = v == "true" || v == "1" || v == "on"
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
		return jsonError(404, "not found")
	}
}

// --- BC API helpers ---

func bcFetch(method, path, body string, headers map[string]string) sdk.HTTPFetchResult {
	h := map[string]string{
		"Authorization": apiKey,
		"Accept":        "application/json",
	}
	for k, v := range headers {
		h[k] = v
	}
	return sdk.HTTPFetch(sdk.HTTPFetchRequest{
		Method:  method,
		URL:     bcBase + path,
		Headers: h,
		Body:    body,
	})
}

func bcFetchBinary(method, path string, bodyBytes []byte, headers map[string]string) sdk.HTTPFetchResult {
	h := map[string]string{
		"Authorization": apiKey,
		"Accept":        "application/json",
	}
	for k, v := range headers {
		h[k] = v
	}
	return sdk.HTTPFetch(sdk.HTTPFetchRequest{
		Method:    method,
		URL:       bcBase + path,
		Headers:   h,
		BodyBytes: bodyBytes,
	})
}

// --- Handlers ---

func handlePing(_ sdk.HTTPRequest) sdk.HTTPResponse {
	if apiKey == "" {
		return jsonError(400, "ballchasing API key not configured")
	}
	res := bcFetch("GET", "/api/", "", nil)
	if res.Error != "" {
		return jsonError(502, res.Error)
	}
	if res.Status != 200 {
		return sdk.HTTPResponse{
			Status:  res.Status,
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    res.Body,
		}
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
	return jsonOK(b)
}

func handleReplays(req sdk.HTTPRequest) sdk.HTTPResponse {
	if apiKey == "" {
		return jsonError(400, "ballchasing API key not configured")
	}
	path := "/api/replays"
	if req.Query != "" {
		path += "?" + req.Query
	} else {
		path += "?uploader=me&count=50&sort-by=replay-date&sort-dir=desc"
	}
	res := bcFetch("GET", path, "", nil)
	if res.Error != "" {
		return jsonError(502, res.Error)
	}
	return sdk.HTTPResponse{
		Status:  res.Status,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    res.Body,
	}
}

func handleGroups(req sdk.HTTPRequest) sdk.HTTPResponse {
	if apiKey == "" {
		return jsonError(400, "ballchasing API key not configured")
	}
	path := "/api/groups"
	if req.Query != "" {
		path += "?" + req.Query
	} else {
		path += "?creator=me&count=50&sort-by=created&sort-dir=desc"
	}
	res := bcFetch("GET", path, "", nil)
	if res.Error != "" {
		return jsonError(502, res.Error)
	}
	return sdk.HTTPResponse{
		Status:  res.Status,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    res.Body,
	}
}

func handleUpload(req sdk.HTTPRequest) sdk.HTTPResponse {
	if req.Method != "POST" {
		return jsonError(405, "method not allowed")
	}
	if apiKey == "" {
		return jsonError(400, "ballchasing API key not configured")
	}

	var body struct {
		ReplayName string `json:"replay_name"`
		Visibility string `json:"visibility"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return jsonError(400, "invalid request body")
	}
	if body.ReplayName == "" {
		return jsonError(400, "replay_name required")
	}
	if body.Visibility == "" {
		body.Visibility = "unlisted"
	}
	switch body.Visibility {
	case "public", "unlisted", "private":
	default:
		return jsonError(400, "invalid visibility")
	}
	if strings.ContainsAny(body.ReplayName, `/\`) || body.ReplayName != stripDir(body.ReplayName) {
		return jsonError(400, "invalid replay name")
	}

	fileData := sdk.ReadFile(body.ReplayName)
	if fileData == nil {
		return jsonError(404, "replay file not found: "+body.ReplayName)
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", body.ReplayName)
	if err != nil {
		return jsonError(500, "multipart: "+err.Error())
	}
	if _, err = fw.Write(fileData); err != nil {
		return jsonError(500, "multipart write: "+err.Error())
	}
	mw.Close()

	res := bcFetchBinary("POST", "/api/v2/upload?visibility="+body.Visibility, buf.Bytes(),
		map[string]string{"Content-Type": mw.FormDataContentType()})
	if res.Error != "" {
		return jsonError(502, res.Error)
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
				sdk.DeleteFile(body.ReplayName)
			}
		}
	}

	return sdk.HTTPResponse{
		Status:  res.Status,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    res.Body,
	}
}

func handlePurgeReplays(req sdk.HTTPRequest) sdk.HTTPResponse {
	if req.Method != "POST" {
		return jsonError(405, "method not allowed")
	}
	rows := sdk.DBQuery(`SELECT replay_name, ballchasing_id FROM bc_uploads`, nil)
	deleted := 0
	for _, row := range rows {
		name, _ := row["replay_name"].(string)
		if name != "" && sdk.DeleteFile(name) {
			deleted++
		}
	}
	b, _ := json.Marshal(map[string]int{"deleted": deleted})
	return jsonOK(b)
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
		t := parseTime(startedAtStr)
		matches = append(matches, matchInfo{MatchGUID: guid, Arena: arena, StartedAt: t})
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

	dirEntries := sdk.ScanDir()
	var sessionFiles []replayFileEntry
	for _, e := range dirEntries {
		if e.IsDir || !strings.HasSuffix(strings.ToLower(e.Name), ".replay") {
			continue
		}
		modTime := parseTime(e.ModTime)
		if !modTime.Before(startupTime) {
			sessionFiles = append(sessionFiles, replayFileEntry{name: e.Name, modTime: modTime})
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

		var bcID string
		if replayName != "" {
			bcID = uploads[replayName]
		}
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
	return jsonOK(b)
}

func handleSync(req sdk.HTTPRequest) sdk.HTTPResponse {
	if req.Method != "POST" {
		return jsonError(405, "method not allowed")
	}
	if apiKey == "" {
		return jsonError(400, "ballchasing API key not configured")
	}

	res := bcFetch("GET", "/api/replays?uploader=me&count=200&sort-by=replay-date&sort-dir=desc", "", nil)
	if res.Error != "" {
		return jsonError(502, res.Error)
	}

	var result struct {
		List []struct {
			ID             string `json:"id"`
			RocketLeagueID string `json:"rocket_league_id"`
		} `json:"list"`
	}
	if err := json.Unmarshal([]byte(res.Body), &result); err != nil {
		return jsonError(502, "failed to parse BC response")
	}

	synced := 0
	for _, rp := range result.List {
		guid := normalizeGUID(rp.RocketLeagueID)
		if guid == "" || rp.ID == "" {
			continue
		}
		n := sdk.DBExec(`INSERT INTO bc_uploads(replay_name, ballchasing_id) VALUES(?,?)
			ON CONFLICT(replay_name) DO UPDATE SET ballchasing_id=excluded.ballchasing_id`,
			[]string{guid + ".replay", rp.ID})
		if n >= 0 {
			synced++
		}
	}
	b, _ := json.Marshal(map[string]int{"synced": synced})
	return jsonOK(b)
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

func parseTime(s string) time.Time {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func stripDir(name string) string {
	for _, sep := range []string{"/", `\`} {
		if i := strings.LastIndex(name, sep); i >= 0 {
			return name[i+1:]
		}
	}
	return name
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
