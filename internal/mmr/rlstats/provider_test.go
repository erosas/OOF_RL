package rlstats

import (
	"os"
	"testing"
)

func TestParseRanksFromFixture(t *testing.T) {
	fixtures := []struct {
		file        string
		wantPlaylists []int // playlist IDs that must be present
		wantTier    string // expected tier for playlist 11 (2v2) — empty = skip
	}{
		{
			file:          "../testdata/rlstats/steam_76561198144145654.html",
			wantPlaylists: []int{10, 11, 13},
			wantTier:      "Supersonic Legend",
		},
		{
			file:          "../testdata/rlstats/psn_nssbali_.html",
			wantPlaylists: []int{11, 13},
		},
		{
			file:          "../testdata/rlstats/xbox_Squishy4939.html",
			wantPlaylists: []int{10, 11, 13},
		},
		{
			file:          "../testdata/rlstats/epic_lllevaro.html",
			wantPlaylists: []int{},
		},
	}

	for _, tc := range fixtures {
		tc := tc
		t.Run(tc.file, func(t *testing.T) {
			data, err := os.ReadFile(tc.file)
			if err != nil {
				t.Skipf("fixture not found (run capture test first): %v", err)
			}

			ranks, err := parseRanks(string(data))
			if err != nil {
				if len(tc.wantPlaylists) == 0 {
					t.Logf("parseRanks returned error (expected for no-data profile): %v", err)
					return
				}
				t.Fatalf("parseRanks: %v", err)
			}

			byID := make(map[int]struct{})
			for _, r := range ranks {
				byID[r.PlaylistID] = struct{}{}
				t.Logf("  playlist=%-25s id=%-3d MMR=%-6.0f tier=%s div=%d",
					r.PlaylistName, r.PlaylistID, r.MMR, r.TierName, r.Division)
			}

			for _, want := range tc.wantPlaylists {
				if _, ok := byID[want]; !ok {
					t.Errorf("missing playlist id=%d", want)
				}
			}

			if tc.wantTier != "" {
				found := false
				for _, r := range ranks {
					if r.PlaylistID == 11 && r.TierName == tc.wantTier {
						found = true
					}
				}
				if !found {
					t.Errorf("expected 2v2 tier=%q, not found in results", tc.wantTier)
				}
			}
		})
	}
}