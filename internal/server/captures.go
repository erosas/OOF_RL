package server

import (
	"bufio"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type captureSummary struct {
	ID          string `json:"id"`
	MatchGuid   string `json:"match_guid"`
	StartedAt   string `json:"started_at_utc"`
	EndedAt     string `json:"ended_at_utc"`
	DurationMs  int64  `json:"duration_ms"`
	PacketCount int    `json:"packet_count"`
	EndReason   string `json:"end_reason"`
}

type captureMetaResp struct {
	NormalizedFiles []string `json:"normalized_files"`
}

func (s *Server) handleCaptures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	base := strings.TrimSpace(s.cfg.Storage.RawPacketsDir)
	if base == "" {
		base = "captures"
	}

	entries, err := os.ReadDir(base)
	if errors.Is(err, os.ErrNotExist) {
		writeJSON(w, []captureSummary{})
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out := make([]captureSummary, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		metaPath := filepath.Join(base, e.Name(), "capture_meta.json")
		b, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var m struct {
			MatchGuid   string `json:"match_guid"`
			StartedAt   string `json:"started_at_utc"`
			EndedAt     string `json:"ended_at_utc"`
			DurationMs  int64  `json:"duration_ms"`
			PacketCount int    `json:"packet_count"`
			EndReason   string `json:"end_reason"`
		}
		if json.Unmarshal(b, &m) != nil {
			continue
		}
		out = append(out, captureSummary{
			ID:          e.Name(),
			MatchGuid:   m.MatchGuid,
			StartedAt:   m.StartedAt,
			EndedAt:     m.EndedAt,
			DurationMs:  m.DurationMs,
			PacketCount: m.PacketCount,
			EndReason:   m.EndReason,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt > out[j].StartedAt })
	writeJSON(w, out)
}

func (s *Server) handleCaptureDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	suffix := strings.TrimPrefix(r.URL.Path, "/api/captures/")
	parts := strings.Split(suffix, "/")
	if len(parts) != 2 || parts[0] == "" {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}
	captureID, kind := parts[0], parts[1]
	if strings.Contains(captureID, "..") || strings.ContainsAny(captureID, `\\/`) {
		http.Error(w, "bad capture id", http.StatusBadRequest)
		return
	}

	base := strings.TrimSpace(s.cfg.Storage.RawPacketsDir)
	if base == "" {
		base = "captures"
	}
	captureDir := filepath.Join(base, captureID)

	switch kind {
	case "meta":
		metaPath := filepath.Join(captureDir, "capture_meta.json")
		b, err := os.ReadFile(metaPath)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	case "index":
		idxPath := filepath.Join(captureDir, "capture_index.json")
		b, err := os.ReadFile(idxPath)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	case "events":
		files, err := captureNormalizedFiles(captureDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		for _, name := range files {
			f, err := os.Open(filepath.Join(captureDir, name))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			sc := bufio.NewScanner(f)
			buf := make([]byte, 0, 1024*1024)
			sc.Buffer(buf, 8*1024*1024)
			for sc.Scan() {
				line := strings.TrimSpace(sc.Text())
				if line == "" {
					continue
				}
				_, _ = w.Write([]byte(line + "\n"))
			}
			_ = f.Close()
			if err := sc.Err(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	default:
		http.Error(w, "bad path", http.StatusBadRequest)
	}
}

func captureNormalizedFiles(captureDir string) ([]string, error) {
	metaPath := filepath.Join(captureDir, "capture_meta.json")
	if b, err := os.ReadFile(metaPath); err == nil {
		var m captureMetaResp
		if json.Unmarshal(b, &m) == nil && len(m.NormalizedFiles) > 0 {
			return m.NormalizedFiles, nil
		}
	}
	entries, err := os.ReadDir(captureDir)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), "packets_normalized_") && strings.HasSuffix(e.Name(), ".ndjson") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, os.ErrNotExist
	}
	return files, nil
}
