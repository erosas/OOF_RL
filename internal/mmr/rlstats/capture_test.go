//go:build manual

// Captures live rlstats.net HTML responses to testdata/ fixtures.
// Run with: go test -v -tags manual -timeout 120s ./internal/mmr/rlstats/

package rlstats_test

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var captureProfiles = []struct {
	name     string
	platform string
	lookup   string
	skip     string
}{
	{"steam_76561198144145654", "steam", "76561198144145654", ""},
	{"psn_nssbali_", "PS4", "nssbali_", ""},
	{"epic_lllevaro", "epic", "lllevaro", ""},
	{"xbox_Squishy4939", "xbox", "Squishy4939", ""},
	{"switch_Squishy", "switch", "Squishy", "rlstats.net dropped Switch support"},
}

func TestCaptureRLStats(t *testing.T) {
	outDir := filepath.Join("..", "testdata", "rlstats")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	client := &http.Client{Timeout: 20 * time.Second}

	for _, tc := range captureProfiles {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip != "" {
				t.Skip(tc.skip)
			}

			profileURL := fmt.Sprintf("https://rlstats.net/profile/%s/%s",
				url.PathEscape(tc.platform), url.PathEscape(tc.lookup))

			req, _ := http.NewRequest(http.MethodGet, profileURL, nil)
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
			req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
			req.Header.Set("Accept-Language", "en-US,en;q=0.9")

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			t.Logf("GET %s → %d (%d bytes)", profileURL, resp.StatusCode, len(body))

			outPath := filepath.Join(outDir, tc.name+".html")
			if err := os.WriteFile(outPath, body, 0o644); err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			t.Logf("saved → %s", outPath)

			time.Sleep(2 * time.Second)
		})
	}
}