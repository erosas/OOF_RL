package trackergg

import (
	"os"
	"testing"
)

func TestParseResponseFromFixture(t *testing.T) {
	fixtures := []struct {
		file        string
		wantCount   int    // minimum playlist segments expected
		wantMMR2v2  float64 // expected 2v2 MMR (0 = skip check)
		wantTier2v2 string
	}{
		{
			file:        "../testdata/trackergg/steam_76561198144145654.json",
			wantCount:   9,
			wantMMR2v2:  2314,
			wantTier2v2: "Supersonic Legend",
		},
		{
			file:      "../testdata/trackergg/psn_nssbali_.json",
			wantCount: 3,
		},
		{
			file:      "../testdata/trackergg/epic_lllevaro.json",
			wantCount: 3,
		},
		{
			file:      "../testdata/trackergg/xbl_Squishy4939.json",
			wantCount: 3,
		},
		{
			file:      "../testdata/trackergg/switch_Squishy.json",
			wantCount: 1,
		},
	}

	for _, tc := range fixtures {
		tc := tc
		t.Run(tc.file, func(t *testing.T) {
			data, err := os.ReadFile(tc.file)
			if err != nil {
				t.Skipf("fixture not found (run capture test first): %v", err)
			}

			ranks, err := parseResponse(data)
			if err != nil {
				t.Fatalf("parseResponse: %v", err)
			}

			for _, r := range ranks {
				t.Logf("  playlist=%-25s id=%-3d MMR=%-6.0f tier=%s div=%d",
					r.PlaylistName, r.PlaylistID, r.MMR, r.TierName, r.Division)
			}

			if len(ranks) < tc.wantCount {
				t.Errorf("got %d playlists, want at least %d", len(ranks), tc.wantCount)
			}

			if tc.wantTier2v2 != "" || tc.wantMMR2v2 != 0 {
				for _, r := range ranks {
					if r.PlaylistID == 11 {
						if tc.wantMMR2v2 != 0 && r.MMR != tc.wantMMR2v2 {
							t.Errorf("2v2 MMR: got %.0f, want %.0f", r.MMR, tc.wantMMR2v2)
						}
						if tc.wantTier2v2 != "" && r.TierName != tc.wantTier2v2 {
							t.Errorf("2v2 tier: got %q, want %q", r.TierName, tc.wantTier2v2)
						}
					}
				}
			}
		})
	}
}