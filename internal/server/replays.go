package server

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type replayFile struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	SizeBytes  int64  `json:"size_bytes"`
	ModifiedAt string `json:"modified_at"`
}

func (s *Server) handleReplays(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dir := detectReplayDir()
	if dir == "" {
		writeJSON(w, []replayFile{})
		return
	}

	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		writeJSON(w, []replayFile{})
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out := make([]replayFile, 0)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".replay") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, replayFile{
			Name:       e.Name(),
			Path:       filepath.Join(dir, e.Name()),
			SizeBytes:  info.Size(),
			ModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ModifiedAt > out[j].ModifiedAt })
	writeJSON(w, out)
}

func detectReplayDir() string {
	home := os.Getenv("USERPROFILE")
	if strings.TrimSpace(home) == "" {
		return ""
	}
	candidates := []string{
		filepath.Join(home, `OneDrive\Documents\My Games\Rocket League\TAGame\Demos`),
		filepath.Join(home, `Documents\My Games\Rocket League\TAGame\Demos`),
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
	}
	return candidates[0]
}
