package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const bcBase = "https://ballchasing.com"

func (s *Server) bcDo(r *http.Request, method, path string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(r.Context(), method, bcBase+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", s.cfg.BallchasingAPIKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	return client.Do(req)
}

func (s *Server) handleBCPing(w http.ResponseWriter, r *http.Request) {
	if s.cfg.BallchasingAPIKey == "" {
		jsonError(w, 400, "ballchasing API key not configured")
		return
	}
	// BC has no public /api/ping; validate auth by fetching one replay and
	// extracting the uploader identity from it.
	resp, err := s.bcDo(r, http.MethodGet, "/api/replays?count=1", nil, "")
	if err != nil {
		jsonError(w, 502, err.Error())
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
		writeJSON(w, map[string]string{"name": u.Name, "steam_id": u.SteamID, "avatar": u.Avatar})
		return
	}
	writeJSON(w, map[string]string{"name": "(connected)"})
}

func (s *Server) handleBCReplays(w http.ResponseWriter, r *http.Request) {
	if s.cfg.BallchasingAPIKey == "" {
		jsonError(w, 400, "ballchasing API key not configured")
		return
	}
	path := "/api/replays"
	if q := r.URL.RawQuery; q != "" {
		path += "?" + q
	} else {
		path += "?count=50&sort-by=replay-date&sort-dir=desc"
	}
	resp, err := s.bcDo(r, http.MethodGet, path, nil, "")
	if err != nil {
		jsonError(w, 502, err.Error())
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func (s *Server) handleBCGroups(w http.ResponseWriter, r *http.Request) {
	if s.cfg.BallchasingAPIKey == "" {
		jsonError(w, 400, "ballchasing API key not configured")
		return
	}
	path := "/api/groups"
	if q := r.URL.RawQuery; q != "" {
		path += "?" + q
	} else {
		path += "?creator=me&count=50&sort-by=created&sort-dir=desc"
	}
	resp, err := s.bcDo(r, http.MethodGet, path, nil, "")
	if err != nil {
		jsonError(w, 502, err.Error())
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

func (s *Server) handleBCUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	if s.cfg.BallchasingAPIKey == "" {
		jsonError(w, 400, "ballchasing API key not configured")
		return
	}

	var req struct {
		ReplayName string `json:"replay_name"`
		Visibility string `json:"visibility"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, 400, err.Error())
		return
	}
	if req.ReplayName == "" {
		jsonError(w, 400, "replay_name required")
		return
	}
	if req.Visibility == "" {
		req.Visibility = "public-team"
	}
	// Sanitise path — no directory traversal.
	if filepath.Base(req.ReplayName) != req.ReplayName {
		jsonError(w, 400, "invalid replay name")
		return
	}

	dir := detectReplayDir()
	if dir == "" {
		jsonError(w, 500, "replay directory not found")
		return
	}
	replayPath := filepath.Join(dir, req.ReplayName)
	fileData, err := os.ReadFile(replayPath)
	if err != nil {
		jsonError(w, 404, fmt.Sprintf("replay file not found: %s", err))
		return
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", req.ReplayName)
	if err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	if _, err = fw.Write(fileData); err != nil {
		jsonError(w, 500, err.Error())
		return
	}
	mw.Close()

	bcResp, err := s.bcDo(r, http.MethodPost,
		"/api/v2/uploads?visibility="+req.Visibility,
		&buf, mw.FormDataContentType())
	if err != nil {
		jsonError(w, 502, err.Error())
		return
	}
	defer bcResp.Body.Close()
	body, _ := io.ReadAll(bcResp.Body)

	// On success store the upload record in the local DB.
	if bcResp.StatusCode == 200 || bcResp.StatusCode == 201 {
		var uploadResp struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(body, &uploadResp) == nil && uploadResp.ID != "" {
			bcURL := "https://ballchasing.com/replay/" + uploadResp.ID
			_ = s.db.UpsertBCUpload(req.ReplayName, uploadResp.ID, bcURL)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(bcResp.StatusCode)
	_, _ = w.Write(body)
}

func (s *Server) handleBCUploads(w http.ResponseWriter, r *http.Request) {
	uploads, err := s.db.AllBCUploads()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, uploads)
}
