package ballchasing

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"OOF_RL/internal/config"
	"OOF_RL/internal/db"
	"OOF_RL/internal/hub"
	"OOF_RL/internal/httputil"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/plugin"
)

//go:embed view.html view.js
var viewFS embed.FS

const (
	bcBase = "https://ballchasing.com"

	// WSEventSaveReplayReminder is broadcast to the browser when a match ends,
	// prompting the user to save the replay before leaving the post-match screen.
	WSEventSaveReplayReminder = "bc:save-replay-reminder"

	// WSEventUploaded is broadcast to the browser after a successful auto-upload.
	WSEventUploaded = "bc:uploaded"
)

type Plugin struct {
	plugin.BasePlugin
	cfg           *config.Config
	store         *store
	hub           *hub.Hub
	startupTime   time.Time
	mu            sync.Mutex
	uploadPending bool
	subs          []oofevents.Subscription
}

func New(cfg *config.Config, database *db.DB, h *hub.Hub) *Plugin {
	if err := database.RunMigration(`
	CREATE TABLE IF NOT EXISTS bc_uploads (
		replay_name    TEXT PRIMARY KEY,
		ballchasing_id TEXT NOT NULL
	);
`); err != nil {
		log.Printf("[ballchasing] migrate: %v", err)
	}
	return &Plugin{cfg: cfg, store: &store{conn: database.Conn()}, hub: h, startupTime: time.Now()}
}

func (p *Plugin) ID() string         { return "ballchasing" }
func (p *Plugin) DBPrefix() string   { return "bc" }
func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) NavTab() plugin.NavTab {
	return plugin.NavTab{ID: "bc", Label: "Ballchasing", Order: 40}
}

func (p *Plugin) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/ballchasing/ping", p.handlePing)
	mux.HandleFunc("/api/ballchasing/local-replays/purge", p.handlePurgeReplays)
	mux.HandleFunc("/api/ballchasing/matches", p.handleBCMatches)
	mux.HandleFunc("/api/ballchasing/sync", p.handleSync)
	mux.HandleFunc("/api/ballchasing/replays", p.handleReplays)
	mux.HandleFunc("/api/ballchasing/groups", p.handleGroups)
	mux.HandleFunc("/api/ballchasing/upload", p.handleUpload)
}

func (p *Plugin) SettingsSchema() []plugin.Setting {
	return []plugin.Setting{
		{
			Key:         "ballchasing_api_key",
			Label:       "API Key",
			Type:        plugin.SettingTypePassword,
			Description: "Your Ballchasing.com API key. Get one at ballchasing.com/upload — required to upload replays and browse groups.",
			Default:     "",
			Placeholder: "your-api-key",
		},
		{
			Key:         "ballchasing_delete_after_upload",
			Label:       "Delete replay after upload",
			Type:        plugin.SettingTypeCheckbox,
			Description: "Automatically delete the local .replay file from disk after it has been successfully uploaded to Ballchasing.",
			Default:     "false",
		},
	}
}

func (p *Plugin) ApplySettings(values map[string]string) error {
	if v, ok := values["ballchasing_api_key"]; ok {
		p.cfg.BallchasingAPIKey = v
	}
	if v, ok := values["ballchasing_delete_after_upload"]; ok {
		p.cfg.BallchasingDeleteAfterUpload = v == "true" || v == "1" || v == "on"
	}
	return nil
}

func (p *Plugin) Assets() fs.FS { return viewFS }

func (p *Plugin) Init(bus oofevents.PluginBus, _ plugin.Registry, _ *db.DB) error {
	p.subs = []oofevents.Subscription{
		bus.Subscribe(oofevents.TypeMatchEnded, p.onMatchEnded),
		bus.Subscribe(oofevents.TypeMatchDestroyed, p.onMatchDestroyed),
	}
	return nil
}

func (p *Plugin) Shutdown() error {
	for _, s := range p.subs {
		s.Cancel()
	}
	return nil
}

// onMatchEnded fires the save-replay reminder while the user is still on the post-match screen.
func (p *Plugin) onMatchEnded(_ oofevents.OOFEvent) {
	evt, _ := json.Marshal(map[string]any{"Event": WSEventSaveReplayReminder})
	p.hub.Broadcast(evt)
}

// onMatchDestroyed triggers auto-upload after the match session is torn down.
func (p *Plugin) onMatchDestroyed(_ oofevents.OOFEvent) {
	if p.cfg.BallchasingAPIKey == "" {
		return
	}
	p.mu.Lock()
	if p.uploadPending {
		p.mu.Unlock()
		return
	}
	p.uploadPending = true
	p.mu.Unlock()

	go func() {
		defer func() {
			p.mu.Lock()
			p.uploadPending = false
			p.mu.Unlock()
		}()
		// Give RL time to finish writing the replay file to disk.
		time.Sleep(12 * time.Second)
		p.autoUpload()
	}()
}

// -- BC API helpers --

func (p *Plugin) bcDo(ctx context.Context, method, path string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, bcBase+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", p.cfg.BallchasingAPIKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	return (&http.Client{Timeout: 30 * time.Second}).Do(req)
}

// -- Auto-upload after match end --

type autoUploadResult struct {
	Name string `json:"name"`
	BcID string `json:"bc_id"`
}

func (p *Plugin) autoUpload() {
	dir := detectReplayDir()
	if dir == "" {
		log.Printf("[bc] auto-upload: replay directory not found")
		return
	}
	log.Printf("[bc] auto-upload scanning: %s", dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("[bc] auto-upload read dir: %v", err)
		return
	}
	existing, err := p.store.allBCUploads()
	if err != nil {
		log.Printf("[bc] auto-upload db: %v", err)
		return
	}

	ctx := context.Background()
	var uploaded []autoUploadResult

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".replay") {
			continue
		}
		name := e.Name()
		if _, ok := existing[name]; ok {
			continue
		}
		// Only auto-upload recent replays (written since startup).
		info, err2 := e.Info()
		if err2 != nil || info.ModTime().Before(p.startupTime) {
			continue
		}
		fullPath := filepath.Join(dir, name)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("[bc] auto-upload read %s: %v", name, err)
			continue
		}

		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, _ := mw.CreateFormFile("file", name)
		_, _ = fw.Write(data)
		mw.Close()

		resp, err := p.bcDo(ctx, http.MethodPost, "/api/v2/upload?visibility=unlisted", &buf, mw.FormDataContentType())
		if err != nil {
			log.Printf("[bc] auto-upload %s: %v", name, err)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 200 || resp.StatusCode == 201 {
			var res struct {
				ID string `json:"id"`
			}
			if json.Unmarshal(body, &res) == nil && res.ID != "" {
				_ = p.store.upsertBCUpload(name, res.ID)
				uploaded = append(uploaded, autoUploadResult{Name: name, BcID: res.ID})
				log.Printf("[bc] auto-uploaded %s → %s", name, res.ID)
				if p.cfg.BallchasingDeleteAfterUpload {
					if rmErr := os.Remove(fullPath); rmErr != nil {
						log.Printf("[bc] auto-upload delete %s: %v", name, rmErr)
					}
				}
			}
		} else {
			log.Printf("[bc] auto-upload %s: BC returned %d", name, resp.StatusCode)
		}
	}

	if len(uploaded) == 0 {
		return
	}
	evt, _ := json.Marshal(map[string]any{
		"Event": WSEventUploaded,
		"Data":  map[string]any{"replays": uploaded},
	})
	p.hub.Broadcast(evt)
}

// -- HTTP handlers --

func (p *Plugin) handlePing(w http.ResponseWriter, r *http.Request) {
	if p.cfg.BallchasingAPIKey == "" {
		httputil.JSONError(w, 400, "ballchasing API key not configured")
		return
	}
	resp, err := p.bcDo(r.Context(), http.MethodGet, "/api/", nil, "")
	if err != nil {
		httputil.JSONError(w, 502, err.Error())
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
		return
	}
	var result struct {
		Name    string `json:"name"`
		SteamID string `json:"steam_id"`
		Avatar  string `json:"avatar"`
	}
	if err := json.Unmarshal(body, &result); err == nil && result.Name != "" {
		httputil.WriteJSON(w, map[string]string{"name": result.Name, "steam_id": result.SteamID, "avatar": result.Avatar})
		return
	}
	httputil.WriteJSON(w, map[string]string{"name": "(connected)"})
}

func (p *Plugin) handleReplays(w http.ResponseWriter, r *http.Request) {
	if p.cfg.BallchasingAPIKey == "" {
		httputil.JSONError(w, 400, "ballchasing API key not configured")
		return
	}
	path := "/api/replays"
	if q := r.URL.RawQuery; q != "" {
		path += "?" + q
	} else {
		path += "?uploader=me&count=50&sort-by=replay-date&sort-dir=desc"
	}
	resp, err := p.bcDo(r.Context(), http.MethodGet, path, nil, "")
	if err != nil {
		httputil.JSONError(w, 502, err.Error())
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func (p *Plugin) handleGroups(w http.ResponseWriter, r *http.Request) {
	if p.cfg.BallchasingAPIKey == "" {
		httputil.JSONError(w, 400, "ballchasing API key not configured")
		return
	}
	path := "/api/groups"
	if q := r.URL.RawQuery; q != "" {
		path += "?" + q
	} else {
		path += "?creator=me&count=50&sort-by=created&sort-dir=desc"
	}
	resp, err := p.bcDo(r.Context(), http.MethodGet, path, nil, "")
	if err != nil {
		httputil.JSONError(w, 502, err.Error())
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func (p *Plugin) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	if p.cfg.BallchasingAPIKey == "" {
		httputil.JSONError(w, 400, "ballchasing API key not configured")
		return
	}

	var req struct {
		ReplayName string `json:"replay_name"`
		Visibility string `json:"visibility"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.JSONError(w, 400, err.Error())
		return
	}
	if req.ReplayName == "" {
		httputil.JSONError(w, 400, "replay_name required")
		return
	}
	if req.Visibility == "" {
		req.Visibility = "unlisted"
	}
	switch req.Visibility {
	case "public", "unlisted", "private":
	default:
		httputil.JSONError(w, 400, "invalid visibility")
		return
	}
	if filepath.Base(req.ReplayName) != req.ReplayName {
		httputil.JSONError(w, 400, "invalid replay name")
		return
	}

	dir := detectReplayDir()
	if dir == "" {
		httputil.JSONError(w, 500, "replay directory not found")
		return
	}
	fullPath := filepath.Join(dir, req.ReplayName)
	fileData, err := os.ReadFile(fullPath)
	if os.IsNotExist(err) {
		httputil.JSONError(w, 404, fmt.Sprintf("replay file not found: %s", req.ReplayName))
		return
	}
	if err != nil {
		httputil.JSONError(w, 500, fmt.Sprintf("could not read replay: %s", err))
		return
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", req.ReplayName)
	if err != nil {
		httputil.JSONError(w, 500, err.Error())
		return
	}
	if _, err = fw.Write(fileData); err != nil {
		httputil.JSONError(w, 500, err.Error())
		return
	}
	mw.Close()

	bcResp, err := p.bcDo(r.Context(), http.MethodPost,
		"/api/v2/upload?visibility="+req.Visibility,
		&buf, mw.FormDataContentType())
	if err != nil {
		httputil.JSONError(w, 502, err.Error())
		return
	}
	defer bcResp.Body.Close()
	body, _ := io.ReadAll(bcResp.Body)

	if bcResp.StatusCode == 200 || bcResp.StatusCode == 201 {
		var uploadResp struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(body, &uploadResp) == nil && uploadResp.ID != "" {
			_ = p.store.upsertBCUpload(req.ReplayName, uploadResp.ID)
			if p.cfg.BallchasingDeleteAfterUpload {
				if rmErr := os.Remove(fullPath); rmErr != nil {
					log.Printf("[bc] upload delete %s: %v", req.ReplayName, rmErr)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(bcResp.StatusCode)
	_, _ = w.Write(body)
}

// handlePurgeReplays deletes local .replay files that have already been uploaded to BC.
func (p *Plugin) handlePurgeReplays(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	dir := detectReplayDir()
	if dir == "" {
		httputil.JSONError(w, 500, "replay directory not found")
		return
	}
	uploaded, err := p.store.allBCUploads()
	if err != nil {
		httputil.JSONError(w, 500, err.Error())
		return
	}
	deleted := 0
	for name := range uploaded {
		if err := os.Remove(filepath.Join(dir, name)); err == nil {
			deleted++
		}
	}
	httputil.WriteJSON(w, map[string]int{"deleted": deleted})
}

// handleBCMatches returns matches from this app session with their replay/upload status.
func (p *Plugin) handleBCMatches(w http.ResponseWriter, r *http.Request) {
	matches, err := p.store.recentMatches(p.startupTime)
	if err != nil {
		httputil.JSONError(w, 500, err.Error())
		return
	}
	uploads, err := p.store.allBCUploads()
	if err != nil {
		httputil.JSONError(w, 500, err.Error())
		return
	}

	// Only consider replay files written since startup — keeps the assignment
	// pool small and prevents matching files from old sessions.
	dir := detectReplayDir()
	allFiles := scanReplayFiles(dir)
	var sessionFiles []replayFileEntry
	for _, f := range allFiles {
		if !f.modTime.Before(p.startupTime) {
			sessionFiles = append(sessionFiles, f)
		}
	}

	// One-to-one greedy assignment: each file goes to the latest eligible match.
	fileForMatch := matchReplayFiles(sessionFiles, matches)

	type matchRow struct {
		MatchGUID    string    `json:"match_guid"`
		Arena        string    `json:"arena"`
		StartedAt    time.Time `json:"started_at"`
		ReplayExists bool      `json:"replay_exists"`
		ReplayName   string    `json:"replay_name,omitempty"`
		Uploaded     bool      `json:"uploaded"`
		BcID         string    `json:"bc_id,omitempty"`
		BcURL        string    `json:"bc_url,omitempty"`
	}
	out := make([]matchRow, 0, len(matches))
	for i, m := range matches {
		normGUID := normalizeGUID(m.MatchGUID)
		replayName := fileForMatch[i]

		// Check bc_uploads by actual filename first, then by normalised GUID
		// (GUID key is written by the sync-from-BC handler).
		var bcID string
		if replayName != "" {
			if u, ok := uploads[replayName]; ok {
				bcID = u.BallchasingID
			}
		}
		if bcID == "" {
			if u, ok := uploads[normGUID+".replay"]; ok {
				bcID = u.BallchasingID
			}
		}

		var bcURL string
		if bcID != "" {
			bcURL = "https://ballchasing.com/replay/" + bcID
		}

		out = append(out, matchRow{
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
	httputil.WriteJSON(w, out)
}

// handleSync fetches the caller's replays from Ballchasing and backfills
// bc_uploads for any that match a known hist_matches entry.
func (p *Plugin) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	if p.cfg.BallchasingAPIKey == "" {
		httputil.JSONError(w, 400, "ballchasing API key not configured")
		return
	}

	// BC paginates; fetch up to 200 most-recent replays for now.
	resp, err := p.bcDo(r.Context(), http.MethodGet,
		"/api/replays?uploader=me&count=200&sort-by=replay-date&sort-dir=desc", nil, "")
	if err != nil {
		httputil.JSONError(w, 502, err.Error())
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		List []struct {
			ID             string `json:"id"`
			RocketLeagueID string `json:"rocket_league_id"`
		} `json:"list"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[bc] sync: failed to parse response body: %s", string(body))
		httputil.JSONError(w, 502, "failed to parse BC response")
		return
	}

	log.Printf("[bc] sync: BC returned %d replays", len(result.List))
	if len(result.List) > 0 {
		first, _ := json.Marshal(result.List[0])
		log.Printf("[bc] sync: first entry fields: %s", string(first))
	}

	synced := 0
	for _, rp := range result.List {
		guid := normalizeGUID(rp.RocketLeagueID)
		if guid == "" || rp.ID == "" {
			continue
		}
		if err := p.store.upsertBCUpload(guid+".replay", rp.ID); err != nil {
			log.Printf("[bc] sync upsert %s: %v", guid, err)
			continue
		}
		synced++
	}
	log.Printf("[bc] sync: %d replays backfilled from Ballchasing", synced)
	httputil.WriteJSON(w, map[string]int{"synced": synced})
}

// normalizeGUID uppercases and strips dashes so BC's rocket_league_id format
// matches the RL replay filenames (e.g. "024690394AE0B6BB20BBD1A3EFB2DA1E").
func normalizeGUID(s string) string {
	return strings.ToUpper(strings.ReplaceAll(s, "-", ""))
}

// --- Replay file helpers ---

type replayFileEntry struct {
	name    string
	modTime time.Time
}

func scanReplayFiles(dir string) []replayFileEntry {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []replayFileEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".replay") {
			continue
		}
		if info, err := e.Info(); err == nil {
			files = append(files, replayFileEntry{name: e.Name(), modTime: info.ModTime()})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].modTime.Before(files[j].modTime) })
	return files
}

// matchReplayFiles does a one-to-one greedy assignment of replay files to matches.
// Each file is assigned to the most-recently-started eligible match (started before
// file modTime, within a 30-min window) that hasn't already claimed a file.
// This prevents a single file matching multiple back-to-back matches.
func matchReplayFiles(files []replayFileEntry, matches []MatchUploadStatus) map[int]string {
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

func detectReplayDir() string {
	const rlRelPath = `My Games\Rocket League\TAGame\Demos`
	seen := map[string]struct{}{}
	var candidates []string

	add := func(base string) {
		if base == "" {
			return
		}
		p := filepath.Join(base, rlRelPath)
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		candidates = append(candidates, p)
	}

	home := os.Getenv("USERPROFILE")
	if home != "" {
		add(filepath.Join(home, "OneDrive", "Documents"))
		add(filepath.Join(home, "Documents"))
	}
	if od := os.Getenv("OneDriveConsumer"); od != "" {
		add(filepath.Join(od, "Documents"))
	}
	if od := os.Getenv("OneDrive"); od != "" {
		add(filepath.Join(od, "Documents"))
	}

	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	if len(candidates) > 0 {
		log.Printf("[bc] replay dir not found; checked: %v", candidates)
	}
	return ""
}
