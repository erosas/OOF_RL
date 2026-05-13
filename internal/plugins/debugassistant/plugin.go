package debugassistant

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"OOF_RL/internal/config"
	"OOF_RL/internal/events"
	"OOF_RL/internal/httputil"
	"OOF_RL/internal/plugin"
)

//go:embed view.html view.js
var viewFS embed.FS

const maxRecentEvents = 120

type recentEvent struct {
	Event     string    `json:"event"`
	MatchGUID string    `json:"match_guid,omitempty"`
	Summary   string    `json:"summary"`
	At        time.Time `json:"at"`
}

type exportReportRequest struct {
	Plain    string          `json:"plain"`
	HTML     string          `json:"html"`
	State    json.RawMessage `json:"state"`
	ExportID string          `json:"export_id"`
}

// Plugin is a read-only regression helper. It observes RL event flow and serves
// tester-facing state without writing match/session/history data.
type Plugin struct {
	plugin.BasePlugin
	cfg    *config.Config
	mu     sync.RWMutex
	events []recentEvent
}

func New(cfg *config.Config) *Plugin {
	p := &Plugin{cfg: cfg}
	p.append("PluginLoaded", "", "Debug Assistant ready")
	return p
}

func (p *Plugin) ID() string         { return "debug-assistant" }
func (p *Plugin) DBPrefix() string   { return "" }
func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) NavTab() plugin.NavTab {
	return plugin.NavTab{ID: "debug", Label: "Debug", Order: 90}
}

func (p *Plugin) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/debug-assistant/events", p.handleEvents)
	mux.HandleFunc("/api/debug-assistant/context", p.handleContext)
	mux.HandleFunc("/api/debug-assistant/screenshots", p.handleScreenshots)
	mux.HandleFunc("/api/debug-assistant/screenshot/", p.handleScreenshot)
	mux.HandleFunc("/api/debug-assistant/export-report", p.handleExportReport)
	mux.HandleFunc("/api/debug-assistant/reset", p.handleReset)
}

func (p *Plugin) SettingsSchema() []plugin.Setting        { return nil }
func (p *Plugin) ApplySettings(_ map[string]string) error { return nil }
func (p *Plugin) Assets() fs.FS                           { return viewFS }

func (p *Plugin) HandleEvent(env events.Envelope) {
	if !p.enabled() {
		return
	}

	switch env.Event {
	case "MatchCreated", "MatchInitialized":
		var d events.MatchGuidData
		_ = json.Unmarshal(env.Data, &d)
		p.append(env.Event, d.MatchGuid, "match lifecycle started")
	case "UpdateState":
		var d events.UpdateStateData
		if err := json.Unmarshal(env.Data, &d); err != nil {
			p.append(env.Event, "", "received state update")
			return
		}
		p.append(env.Event, d.MatchGuid, updateStateSummary(d))
	case "GoalScored":
		var d events.GoalScoredData
		_ = json.Unmarshal(env.Data, &d)
		p.append(env.Event, d.MatchGuid, playerSummary(d.Scorer.Name, "goal scored"))
	case "StatfeedEvent":
		var d events.StatfeedEventData
		_ = json.Unmarshal(env.Data, &d)
		p.append(env.Event, d.MatchGuid, playerSummary(d.MainTarget.Name, d.EventName))
	case "ClockUpdatedSeconds":
		var d events.ClockData
		_ = json.Unmarshal(env.Data, &d)
		p.append(env.Event, d.MatchGuid, clockSummary(d))
	case "MatchEnded":
		var d events.MatchEndedData
		_ = json.Unmarshal(env.Data, &d)
		p.append(env.Event, d.MatchGuid, "match ended")
	case "MatchDestroyed":
		p.append(env.Event, "", "match destroyed")
	}
}

func (p *Plugin) enabled() bool {
	if p.cfg == nil {
		return true
	}
	for _, id := range p.cfg.DisabledPlugins {
		if id == p.ID() {
			return false
		}
	}
	return true
}

func (p *Plugin) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	p.mu.RLock()
	events := append([]recentEvent(nil), p.events...)
	p.mu.RUnlock()

	httputil.WriteJSON(w, map[string]any{"events": events})
}

func (p *Plugin) handleContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	p.mu.RLock()
	count := len(p.events)
	last := recentEvent{}
	if count > 0 {
		last = p.events[count-1]
	}
	p.mu.RUnlock()

	dataDir := ""
	if p.cfg != nil {
		dataDir = p.cfg.DataDir
	}
	httputil.WriteJSON(w, map[string]any{
		"data_dir":          dataDir,
		"observed_events":   count,
		"last_event":        last,
		"plugin_enabled":    p.enabled(),
		"capture_note":      "Collect oof_rl.log, oof_rl.db when needed, latest captures folder, and screenshots for failed checks.",
		"source_of_truth":   "Debug Assistant is read-only and does not mutate Live, Session, History, or match state.",
		"screenshot_target": "History collapsed row, expanded match details, Session overview, and Live state when relevant.",
	})
}

func (p *Plugin) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	p.mu.Lock()
	p.events = nil
	p.mu.Unlock()

	httputil.WriteJSON(w, map[string]any{"ok": true})
}

func (p *Plugin) handleScreenshots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dir := p.debugScreenshotsDir()
	if dir == "" {
		httputil.WriteJSON(w, map[string]any{"screenshots": []string{}})
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		httputil.WriteJSON(w, map[string]any{"screenshots": []string{}})
		return
	}

	var screenshots []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if isDebugImage(name) {
			screenshots = append(screenshots, name)
		}
	}
	httputil.WriteJSON(w, map[string]any{"screenshots": screenshots})
}

func (p *Plugin) handleScreenshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rawName := strings.TrimPrefix(r.URL.Path, "/api/debug-assistant/screenshot/")
	decodedName, err := url.PathUnescape(rawName)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	name := filepath.Base(decodedName)
	if name == "." || name == "" || !isDebugImage(name) {
		http.NotFound(w, r)
		return
	}
	dir := p.debugScreenshotsDir()
	if dir == "" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(dir, name))
}

func (p *Plugin) handleExportReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if p.cfg == nil || p.cfg.DataDir == "" {
		http.Error(w, "data dir unavailable", http.StatusInternalServerError)
		return
	}

	var req exportReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Plain) == "" && strings.TrimSpace(req.HTML) == "" {
		http.Error(w, "empty report", http.StatusBadRequest)
		return
	}

	dir := filepath.Join(p.cfg.DataDir, "debug_reports")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		http.Error(w, "creating report dir: "+err.Error(), http.StatusInternalServerError)
		return
	}

	exportID := safeExportID(req.ExportID)
	if exportID == "" {
		exportID = time.Now().Format("20060102-150405")
	}
	base := "oof-rl-debug-report-" + exportID
	plainPath := filepath.Join(dir, base+".md")
	htmlPath := filepath.Join(dir, base+".html")
	jsonPath := filepath.Join(dir, base+".json")

	if fileExists(plainPath) || fileExists(htmlPath) || fileExists(jsonPath) {
		httputil.WriteJSON(w, map[string]any{
			"dir":       dir,
			"markdown":  plainPath,
			"html":      htmlPath,
			"json":      jsonPath,
			"duplicate": true,
			"message":   "Report already exported. Duplicate export skipped.",
		})
		return
	}

	if err := os.WriteFile(plainPath, []byte(req.Plain), 0o644); err != nil {
		http.Error(w, "writing markdown report: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(htmlPath, []byte(req.HTML), 0o644); err != nil {
		http.Error(w, "writing html report: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if len(req.State) > 0 {
		if err := os.WriteFile(jsonPath, req.State, 0o644); err != nil {
			http.Error(w, "writing json report: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	httputil.WriteJSON(w, map[string]any{
		"dir":       dir,
		"markdown":  plainPath,
		"html":      htmlPath,
		"json":      jsonPath,
		"duplicate": false,
	})
}

func (p *Plugin) debugScreenshotsDir() string {
	if p.cfg == nil || p.cfg.DataDir == "" {
		return ""
	}
	return filepath.Join(p.cfg.DataDir, "debug_screenshots")
}

func isDebugImage(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif":
		return true
	default:
		return false
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

func (p *Plugin) append(event, matchGUID, summary string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.events = append(p.events, recentEvent{
		Event:     event,
		MatchGUID: matchGUID,
		Summary:   summary,
		At:        time.Now().UTC(),
	})
	if len(p.events) > maxRecentEvents {
		p.events = p.events[len(p.events)-maxRecentEvents:]
	}
}

func updateStateSummary(d events.UpdateStateData) string {
	blueScore, orangeScore := 0, 0
	for _, team := range d.Game.Teams {
		switch team.TeamNum {
		case 0:
			blueScore = team.Score
		case 1:
			orangeScore = team.Score
		}
	}
	return "state update: " +
		strconv.Itoa(len(d.Players)) + " players, score " +
		strconv.Itoa(blueScore) + "-" + strconv.Itoa(orangeScore)
}

func clockSummary(d events.ClockData) string {
	if d.BOvertime {
		return "clock update: overtime"
	}
	return "clock update: " + strconv.Itoa(d.TimeSeconds) + "s"
}

func playerSummary(name, action string) string {
	if name == "" {
		return action
	}
	return name + ": " + action
}
