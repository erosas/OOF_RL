package rlstats

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"OOF_RL/internal/mmr"
)

// chartDataRe matches embedded Google-Charts row data in the rlstats.net profile page.
// Each row looks like:  new Date(timestamp*1000), mmr1v1, mmr2v2, mmr3v3
// The capture group holds the comma-separated MMR values (digits/commas/spaces only;
// null values terminate the match, leaving fewer than 3 items).
var chartDataRe = regexp.MustCompile(`new Date\(\d+\*1000\),\s*([\d,\s]+)`)

// playlist maps capture-group index → RL playlist ID and name.
var playlists = []struct {
	id   int
	name string
}{
	{10, "Ranked Duel 1v1"},
	{11, "Ranked Doubles 2v2"},
	{13, "Ranked Standard 3v3"},
}

// Provider fetches MMR by scraping the rlstats.net profile page.
// It returns only MMR values; Tier/TierName are left empty because rlstats.net
// does not expose tier data in the embedded chart rows.
type Provider struct {
	client *http.Client
}

func New() *Provider {
	return &Provider{client: &http.Client{Timeout: 12 * time.Second}}
}

func (p *Provider) Name() string { return "rlstats.net" }

func (p *Provider) Lookup(id mmr.PlayerIdentity) ([]mmr.PlaylistRank, error) {
	platform, lookup := platformAndLookup(id)
	if platform == "" || lookup == "" {
		return nil, fmt.Errorf("rlstats: cannot build URL for %+v", id)
	}

	profileURL := fmt.Sprintf("https://rlstats.net/profile/%s/%s",
		url.PathEscape(platform), url.PathEscape(lookup))

	req, err := http.NewRequest(http.MethodGet, profileURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	log.Printf("[rlstats] %s → %d (%d bytes)", profileURL, resp.StatusCode, len(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rlstats: HTTP %d for %s", resp.StatusCode, profileURL)
	}
	return parseMMR(string(body))
}

// platformAndLookup maps a PlayerIdentity to the rlstats.net platform slug and
// lookup key (SteamID64 for Steam, display name for all other platforms).
func platformAndLookup(id mmr.PlayerIdentity) (platform, lookup string) {
	switch id.Platform {
	case mmr.PlatformSteam:
		return "steam", id.PrimaryID
	case mmr.PlatformEpic:
		return "epic", orFallback(id.DisplayName, id.PrimaryID)
	case mmr.PlatformPSN:
		return "ps", orFallback(id.DisplayName, id.PrimaryID) // rlstats uses "ps", not "psn"
	case mmr.PlatformXbox:
		return "xbl", orFallback(id.DisplayName, id.PrimaryID)
	case mmr.PlatformSwitch:
		name := orFallback(id.DisplayName, id.PrimaryID)
		if isAllAsterisks(name) {
			return "", ""
		}
		return "switch", name
	default:
		return string(id.Platform), id.PrimaryID
	}
}

// parseMMR finds all chart data rows in the page HTML and extracts MMR values
// from the most recent row (last match = most recent history point).
func parseMMR(html string) ([]mmr.PlaylistRank, error) {
	matches := chartDataRe.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("rlstats: no chart data found in page")
	}

	// The last match is the most recent data point.
	parts := strings.Split(matches[len(matches)-1][1], ",")

	var ranks []mmr.PlaylistRank
	for i, pl := range playlists {
		if i >= len(parts) {
			break
		}
		v := strings.TrimSpace(parts[i])
		if v == "" {
			continue
		}
		mmrVal, err := strconv.ParseFloat(v, 64)
		if err != nil || mmrVal <= 0 {
			continue
		}
		ranks = append(ranks, mmr.PlaylistRank{
			PlaylistID:   pl.id,
			PlaylistName: pl.name,
			MMR:          mmrVal,
		})
	}

	if len(ranks) == 0 {
		return nil, fmt.Errorf("rlstats: no valid MMR values parsed from chart data")
	}
	return ranks, nil
}

func orFallback(s, fallback string) string {
	if s != "" {
		return s
	}
	return fallback
}

func isAllAsterisks(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c != '*' {
			return false
		}
	}
	return true
}