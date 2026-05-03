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
	"strings"
	"sync"
	"syscall"
	"time"

	"OOF_RL/internal/config"
	"OOF_RL/internal/db"
	"OOF_RL/internal/events"
	"OOF_RL/internal/hub"
	"OOF_RL/internal/httputil"
	"OOF_RL/internal/plugin"
)

//go:embed view.html view.js
var viewFS embed.FS

const bcBase = "https://ballchasing.com"

type Plugin struct {
	cfg           *config.Config
	db            *db.DB
	hub           *hub.Hub
	mu            sync.Mutex
	uploadPending bool
}

func New(cfg *config.Config, database *db.DB, h *hub.Hub) *Plugin {
	if err := database.RunMigration(`
	CREATE TABLE IF NOT EXISTS bc_uploads (
		replay_name    TEXT PRIMARY KEY,
		ballchasing_id TEXT NOT NULL,
		bc_url         TEXT NOT NULL,
		uploaded_at    DATETIME NOT NULL
	);
`); err != nil {
		log.Printf("[ballchasing] migrate: %v", err)
	}
	return &Plugin{cfg: cfg, db: database, hub: h}
}

func (p *Plugin) ID() string         { return "ballchasing" }
func (p *Plugin) DBPrefix() string   { return "bc" }
func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) NavTab() plugin.NavTab {
	return plugin.NavTab{ID: "bc", Label: "Ballchasing", Order: 40}
}

func (p *Plugin) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/ballchasing/ping", p.handlePing)
	mux.HandleFunc("/api/ballchasing/replays", p.handleReplays)
	mux.HandleFunc("/api/ballchasing/groups", p.handleGroups)
	mux.HandleFunc("/api/ballchasing/upload", p.handleUpload)
	mux.HandleFunc("/api/ballchasing/uploads", p.handleUploads)
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
	}
}

func (p *Plugin) ApplySettings(values map[string]string) error {
	if v, ok := values["ballchasing_api_key"]; ok {
		p.cfg.BallchasingAPIKey = v
	}
	return nil
}

// HandleEvent triggers auto-upload 5 seconds after each match ends.
func (p *Plugin) HandleEvent(env events.Envelope) {
	if env.Event != "MatchDestroyed" || p.cfg.BallchasingAPIKey == "" {
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

func (p *Plugin) Assets() fs.FS { return viewFS }

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
	Name  string `json:"name"`
	BcID  string `json:"bc_id"`
	BcURL string `json:"bc_url"`
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
	existing, err := p.db.AllBCUploads()
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
		// Only auto-upload recent replays — older ones can be uploaded manually.
		if info, err2 := e.Info(); err2 == nil && time.Since(replayCreatedAt(info)) > 30*time.Minute {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			log.Printf("[bc] auto-upload read %s: %v", name, err)
			continue
		}

		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, _ := mw.CreateFormFile("file", name)
		_, _ = fw.Write(data)
		mw.Close()

		resp, err := p.bcDo(ctx, http.MethodPost, "/api/v2/upload?visibility=public-team", &buf, mw.FormDataContentType())
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
				bcURL := "https://ballchasing.com/replay/" + res.ID
				_ = p.db.UpsertBCUpload(name, res.ID, bcURL)
				uploaded = append(uploaded, autoUploadResult{Name: name, BcID: res.ID, BcURL: bcURL})
				log.Printf("[bc] auto-uploaded %s → %s", name, res.ID)
			}
		} else {
			log.Printf("[bc] auto-upload %s: BC returned %d", name, resp.StatusCode)
		}
	}

	if len(uploaded) == 0 {
		return
	}
	evt, _ := json.Marshal(map[string]any{
		"Event": "bc:uploaded",
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
	resp, err := p.bcDo(r.Context(), http.MethodGet, "/api/replays?count=1", nil, "")
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
		List []struct {
			Uploader struct {
				Name    string `json:"name"`
				SteamID string `json:"steam_id"`
				Avatar  string `json:"avatar"`
			} `json:"uploader"`
		} `json:"list"`
	}
	if err := json.Unmarshal(body, &result); err == nil && len(result.List) > 0 {
		u := result.List[0].Uploader
		httputil.WriteJSON(w, map[string]string{"name": u.Name, "steam_id": u.SteamID, "avatar": u.Avatar})
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
		path += "?count=50&sort-by=replay-date&sort-dir=desc"
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
		req.Visibility = "public-team"
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
			bcURL := "https://ballchasing.com/replay/" + uploadResp.ID
			_ = p.db.UpsertBCUpload(req.ReplayName, uploadResp.ID, bcURL)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(bcResp.StatusCode)
	_, _ = w.Write(body)
}

func (p *Plugin) handleUploads(w http.ResponseWriter, r *http.Request) {
	uploads, err := p.db.AllBCUploads()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	httputil.WriteJSON(w, uploads)
}

// replayCreatedAt returns the Windows file creation time, falling back to ModTime.
func replayCreatedAt(info os.FileInfo) time.Time {
	if stat, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, stat.CreationTime.Nanoseconds())
	}
	return info.ModTime()
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
		// OneDrive-synced Documents (most common on modern Windows)
		add(filepath.Join(home, "OneDrive", "Documents"))
		// Standard Documents
		add(filepath.Join(home, "Documents"))
	}
	// OneDrive environment variables (personal and work accounts)
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
