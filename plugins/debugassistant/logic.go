//go:build wasip1

package main

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	sdk "github.com/erosas/oof-plugin-sdk"
)

const maxRecentEvents = 120

type recentEvent struct {
	Event     string    `json:"event"`
	MatchGUID string    `json:"match_guid,omitempty"`
	Summary   string    `json:"summary"`
	At        time.Time `json:"at"`
}

// events holds the in-memory event log. No mutex needed: the host serializes
// all calls into a WASM module instance.
var events []recentEvent

func initPlugin() uint32 {
	// Pre-create the screenshots directory so the user knows where to drop files.
	os.MkdirAll("/data/screenshots", 0o755)
	appendEvent("PluginLoaded", "", "Debug Assistant ready")
	return 0
}

func appendEvent(eventType, matchGUID, summary string) {
	events = append(events, recentEvent{
		Event:     eventType,
		MatchGUID: matchGUID,
		Summary:   summary,
		At:        time.Now().UTC(),
	})
	if len(events) > maxRecentEvents {
		events = events[len(events)-maxRecentEvents:]
	}
}

// onEvent is called for every subscribed event type. Payloads are JSON from
// oofevents.Unwrap — structs without json tags marshal as PascalCase; GUID is
// the match GUID from the embedded Base.
func onEvent(eventType string, payload []byte) {
	switch eventType {
	case "match.started":
		var e struct {
			GUID string `json:"GUID"`
		}
		json.Unmarshal(payload, &e)
		appendEvent(eventType, e.GUID, "match lifecycle started")

	case "state.updated":
		var e struct {
			GUID    string            `json:"GUID"`
			Players []json.RawMessage `json:"Players"`
			Game    struct {
				Teams []struct {
					TeamNum int `json:"TeamNum"`
					Score   int `json:"Score"`
				} `json:"Teams"`
			} `json:"Game"`
		}
		json.Unmarshal(payload, &e)
		blue, orange := 0, 0
		for _, t := range e.Game.Teams {
			switch t.TeamNum {
			case 0:
				blue = t.Score
			case 1:
				orange = t.Score
			}
		}
		summary := "state update: " + strconv.Itoa(len(e.Players)) + " players, score " +
			strconv.Itoa(blue) + "-" + strconv.Itoa(orange)
		appendEvent(eventType, e.GUID, summary)

	case "goal.scored":
		var e struct {
			GUID   string `json:"GUID"`
			Scorer string `json:"Scorer"`
		}
		json.Unmarshal(payload, &e)
		appendEvent(eventType, e.GUID, playerSummary(e.Scorer, "goal scored"))

	case "stat.feed":
		var e struct {
			GUID       string `json:"GUID"`
			EventName  string `json:"EventName"`
			MainTarget string `json:"MainTarget"`
		}
		json.Unmarshal(payload, &e)
		appendEvent(eventType, e.GUID, playerSummary(e.MainTarget, e.EventName))

	case "clock.updated":
		var e struct {
			GUID        string `json:"GUID"`
			TimeSeconds int    `json:"TimeSeconds"`
			IsOvertime  bool   `json:"IsOvertime"`
		}
		json.Unmarshal(payload, &e)
		summary := "clock update: " + strconv.Itoa(e.TimeSeconds) + "s"
		if e.IsOvertime {
			summary = "clock update: overtime"
		}
		appendEvent(eventType, e.GUID, summary)

	case "match.ended":
		var e struct {
			GUID string `json:"GUID"`
		}
		json.Unmarshal(payload, &e)
		appendEvent(eventType, e.GUID, "match ended")

	case "match.destroyed":
		appendEvent(eventType, "", "match destroyed")
	}
}

func handleHTTP(req sdk.HTTPRequest) sdk.HTTPResponse {
	switch req.Path {
	case "/api/debug-assistant/events":
		if req.Method != "GET" {
			return sdk.JSONError(405, "method not allowed")
		}
		return handleEvents()

	case "/api/debug-assistant/context":
		if req.Method != "GET" {
			return sdk.JSONError(405, "method not allowed")
		}
		return handleContext()

	case "/api/debug-assistant/screenshots":
		if req.Method != "GET" {
			return sdk.JSONError(405, "method not allowed")
		}
		return handleListScreenshots()

	case "/api/debug-assistant/export-report":
		if req.Method != "POST" {
			return sdk.JSONError(405, "method not allowed")
		}
		return handleExportReport(req)

	case "/api/debug-assistant/reset":
		if req.Method != "POST" {
			return sdk.JSONError(405, "method not allowed")
		}
		return handleReset()
	}

	if strings.HasPrefix(req.Path, "/api/debug-assistant/screenshot/") {
		if req.Method != "GET" {
			return sdk.JSONError(405, "method not allowed")
		}
		return handleScreenshot(req.Path)
	}

	return sdk.JSONError(404, "not found")
}

func handleEvents() sdk.HTTPResponse {
	b, _ := json.Marshal(map[string]any{"events": events})
	return sdk.JSONResponse(b)
}

func handleContext() sdk.HTTPResponse {
	count := len(events)
	last := recentEvent{}
	if count > 0 {
		last = events[count-1]
	}
	dataDir := sdk.GetConfig("data_dir")
	b, _ := json.Marshal(map[string]any{
		"data_dir":          dataDir,
		"observed_events":   count,
		"last_event":        last,
		"plugin_enabled":    true,
		"capture_note":      "Collect oof_rl.log, oof_rl.db when needed, and screenshots for failed checks.",
		"source_of_truth":   "Debug Assistant is read-only and does not mutate Live, Session, History, or match state.",
		"screenshot_target": "History collapsed row, expanded match details, Session overview, and Live state when relevant.",
	})
	return sdk.JSONResponse(b)
}

func handleListScreenshots() sdk.HTTPResponse {
	entries, err := os.ReadDir("/data/screenshots")
	if err != nil {
		b, _ := json.Marshal(map[string]any{"screenshots": []string{}})
		return sdk.JSONResponse(b)
	}
	names := []string{}
	for _, e := range entries {
		if !e.IsDir() && isImage(e.Name()) {
			names = append(names, e.Name())
		}
	}
	b, _ := json.Marshal(map[string]any{"screenshots": names})
	return sdk.JSONResponse(b)
}

func handleScreenshot(path string) sdk.HTTPResponse {
	rawName := strings.TrimPrefix(path, "/api/debug-assistant/screenshot/")
	decoded, err := url.PathUnescape(rawName)
	if err != nil {
		return sdk.JSONError(404, "not found")
	}
	name := filepath.Base(decoded)
	if name == "." || name == "" || !isImage(name) {
		return sdk.JSONError(404, "not found")
	}
	data, err := os.ReadFile("/data/screenshots/" + name)
	if err != nil {
		return sdk.JSONError(404, "not found")
	}
	return sdk.HTTPResponse{
		Status:    200,
		Headers:   map[string]string{"Content-Type": imageContentType(name)},
		BodyBytes: data,
	}
}

func handleExportReport(req sdk.HTTPRequest) sdk.HTTPResponse {
	var body struct {
		Plain    string          `json:"plain"`
		HTML     string          `json:"html"`
		State    json.RawMessage `json:"state"`
		ExportID string          `json:"export_id"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return sdk.JSONError(400, "invalid request")
	}
	if strings.TrimSpace(body.Plain) == "" && strings.TrimSpace(body.HTML) == "" {
		return sdk.JSONError(400, "empty report")
	}

	if err := os.MkdirAll("/data/reports", 0o755); err != nil {
		return sdk.JSONError(500, "creating report dir: "+err.Error())
	}

	exportID := safeExportID(body.ExportID)
	if exportID == "" {
		exportID = time.Now().Format("20060102-150405")
	}
	base := "oof-rl-debug-report-" + exportID
	plainName := base + ".md"
	htmlName := base + ".html"
	jsonName := base + ".json"

	duplicate := fileExists("/data/reports/"+plainName) || fileExists("/data/reports/"+htmlName)
	if duplicate {
		return sdk.JSONResponse(reportPaths(plainName, htmlName, jsonName, true, "Report already exported. Duplicate export skipped."))
	}

	if err := os.WriteFile("/data/reports/"+plainName, []byte(body.Plain), 0o644); err != nil {
		return sdk.JSONError(500, "writing markdown: "+err.Error())
	}
	if err := os.WriteFile("/data/reports/"+htmlName, []byte(body.HTML), 0o644); err != nil {
		return sdk.JSONError(500, "writing html: "+err.Error())
	}
	if len(body.State) > 0 {
		if err := os.WriteFile("/data/reports/"+jsonName, body.State, 0o644); err != nil {
			return sdk.JSONError(500, "writing json: "+err.Error())
		}
	}

	return sdk.JSONResponse(reportPaths(plainName, htmlName, jsonName, false, ""))
}

// reportPaths builds the export-report response body. Absolute paths are
// constructed from data_dir config so the user can open files directly.
func reportPaths(plainName, htmlName, jsonName string, duplicate bool, msg string) []byte {
	dataDir := sdk.GetConfig("data_dir")
	reportsDir := filepath.Join(dataDir, "plugin_data", "debugassistant", "reports")
	m := map[string]any{
		"dir":       reportsDir,
		"markdown":  filepath.Join(reportsDir, plainName),
		"html":      filepath.Join(reportsDir, htmlName),
		"json":      filepath.Join(reportsDir, jsonName),
		"duplicate": duplicate,
	}
	if msg != "" {
		m["message"] = msg
	}
	b, _ := json.Marshal(m)
	return b
}

func handleReset() sdk.HTTPResponse {
	events = nil
	b, _ := json.Marshal(map[string]any{"ok": true})
	return sdk.JSONResponse(b)
}

// --- helpers ---

func isImage(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif":
		return true
	}
	return false
}

func imageContentType(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}

func safeExportID(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() > 64 {
		return b.String()[:64]
	}
	return b.String()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func playerSummary(name, action string) string {
	if name == "" {
		return action
	}
	return name + ": " + action
}