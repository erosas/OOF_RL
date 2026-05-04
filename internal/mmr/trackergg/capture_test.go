//go:build manual

// Captures live tracker.gg API responses to testdata/ fixtures.
// Run with: go test -v -tags manual -timeout 120s ./internal/mmr/trackergg/

package trackergg_test

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
}{
	{"steam_76561198144145654", "steam", "76561198144145654"},
	{"psn_nssbali_", "psn", "nssbali_"},
	{"epic_lllevaro", "epic", "lllevaro"},
	{"xbl_Squishy4939", "xbl", "Squishy4939"},
	{"switch_Squishy", "switch", "Squishy"},
}

func TestCaptureTrackerGG(t *testing.T) {
	outDir := filepath.Join("..", "testdata", "trackergg")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	client := &http.Client{Timeout: 20 * time.Second}

	for _, tc := range captureProfiles {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			apiURL := fmt.Sprintf("https://api.tracker.gg/api/v2/rocket-league/standard/profile/%s/%s",
				url.PathEscape(tc.platform), url.PathEscape(tc.lookup))

			req, _ := http.NewRequest(http.MethodGet, apiURL, nil)
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Accept-Language", "en-US,en;q=0.9")
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			t.Logf("GET %s → %d (%d bytes)", apiURL, resp.StatusCode, len(body))

			outPath := filepath.Join(outDir, tc.name+".json")
			if err := os.WriteFile(outPath, body, 0o644); err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			t.Logf("saved → %s", outPath)

			// Rate-limit friendly pause between requests.
			time.Sleep(4 * time.Second)
		})
	}
}