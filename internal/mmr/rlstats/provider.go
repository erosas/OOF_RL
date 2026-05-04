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
// Used as fallback when table parsing yields nothing.
var chartDataRe = regexp.MustCompile(`new Date\(\d+\*1000\),\s*([\d,\s]+)`)

// HTML table parsing regexes.
var (
	blockBodyVisibleRe = regexp.MustCompile(`<div class="block-body" data-season="\d+" style="display: block;">`)
	skillTableRe       = regexp.MustCompile(`(?s)<table>(.*?)</table>`)
	tableRowRe         = regexp.MustCompile(`(?s)<tr>(.*?)</tr>`)
	thCellRe           = regexp.MustCompile(`(?s)<th[^>]*>(.*?)</th>`)
	tdCellRe           = regexp.MustCompile(`(?s)<td[^>]*>(.*?)</td>`)
	mmrTagRe           = regexp.MustCompile(`(?s)<mmr[^>]*>.*?</mmr>`)
	htmlTagRe          = regexp.MustCompile(`<[^>]+>`)
	multiSpaceRe       = regexp.MustCompile(`\s+`)
)

// playlistInfo maps rlstats.net playlist header names to playlist IDs and
// canonical names (matching the tracker.gg convention for consistency).
var playlistInfo = map[string]struct {
	id   int
	name string
}{
	"1v1 Duel":      {10, "Ranked Duel 1v1"},
	"2v2 Doubles":   {11, "Ranked Doubles 2v2"},
	"3v3 Standard":  {13, "Ranked Standard 3v3"},
	"2v2 Heatseeker": {63, "Heatseeker"},
	"2v2 Hoops":     {27, "Hoops"},
	"3v3 Rumble":    {28, "Rumble"},
	"3v3 Dropshot":  {29, "Dropshot"},
	"3v3 Snow Day":  {30, "Snowday"},
	"Tournaments":   {34, "Tournament Matches"},
}

// Provider fetches MMR by scraping the rlstats.net profile page.
type Provider struct {
	client *http.Client
}

func New() *Provider {
	return &Provider{client: &http.Client{Timeout: 12 * time.Second}}
}

func (p *Provider) Name() string { return "rlstats.net" }

func (p *Provider) Supports(platform mmr.Platform) bool {
	return platform != mmr.PlatformSwitch
}

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
	bodyStr := string(body)
	if bodyStr == "UnknownPlatform" {
		return nil, fmt.Errorf("rlstats: platform %q is not supported", id.Platform)
	}
	if strings.Contains(bodyStr, "we can no longer support") {
		return nil, fmt.Errorf("rlstats: platform %q is no longer supported by rlstats.net", id.Platform)
	}
	return parseRanks(bodyStr)
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
		return "PS4", orFallback(id.DisplayName, id.PrimaryID)
	case mmr.PlatformXbox:
		return "xbox", orFallback(id.DisplayName, id.PrimaryID)
	case mmr.PlatformSwitch:
		return "", "" // rlstats.net dropped Switch support
	default:
		return string(id.Platform), id.PrimaryID
	}
}

// parseRanks extracts full rank data (tier, division, MMR, all playlists) from the
// rlstats.net profile HTML. Falls back to chart-data MMR extraction if table
// parsing yields nothing.
func parseRanks(html string) ([]mmr.PlaylistRank, error) {
	skillsHTML := currentSeasonSkillsHTML(html)
	if skillsHTML == "" {
		return parseMMRFallback(html)
	}

	tables := skillTableRe.FindAllStringSubmatch(skillsHTML, -1)
	if len(tables) == 0 {
		return parseMMRFallback(html)
	}

	var ranks []mmr.PlaylistRank
	for _, tableMatch := range tables {
		rows := tableRowRe.FindAllStringSubmatch(tableMatch[1], -1)
		if len(rows) < 4 {
			continue
		}

		headers := thCellRe.FindAllStringSubmatch(rows[0][1], -1)
		tierCells := tdCellRe.FindAllStringSubmatch(rows[1][1], -1)
		divCells := tdCellRe.FindAllStringSubmatch(rows[2][1], -1)
		mmrCells := tdCellRe.FindAllStringSubmatch(rows[3][1], -1)

		for i, hMatch := range headers {
			name := cleanText(hMatch[1])
			pl, ok := playlistInfo[name]
			if !ok {
				continue
			}

			tierName := ""
			if i < len(tierCells) {
				tierName = cleanText(tierCells[i][1])
			}
			division := 0
			if i < len(divCells) {
				division = parseDivision(cleanText(divCells[i][1]))
			}
			mmrVal := 0.0
			if i < len(mmrCells) {
				mmrVal = extractMMR(mmrCells[i][1])
			}

			if mmrVal <= 0 && tierName == "Unranked" {
				continue
			}

			ranks = append(ranks, mmr.PlaylistRank{
				PlaylistID:   pl.id,
				PlaylistName: pl.name,
				MMR:          mmrVal,
				TierName:     tierName,
				Division:     division,
			})
		}
	}

	if len(ranks) == 0 {
		return parseMMRFallback(html)
	}
	return ranks, nil
}

// currentSeasonSkillsHTML finds the block-skills content for the visible (current) season.
func currentSeasonSkillsHTML(html string) string {
	loc := blockBodyVisibleRe.FindStringIndex(html)
	if loc == nil {
		return ""
	}
	tail := html[loc[0]:]

	skillsStart := strings.Index(tail, `<div class="block-skills">`)
	if skillsStart < 0 {
		return ""
	}
	skillsTail := tail[skillsStart:]

	// End at the next season block or end of string.
	nextSeason := strings.Index(skillsTail[1:], `data-season="`)
	if nextSeason > 0 {
		return skillsTail[:nextSeason+1]
	}
	return skillsTail
}

// extractMMR strips <mmr> gain/loss tags and returns the central MMR float.
func extractMMR(cellHTML string) float64 {
	clean := mmrTagRe.ReplaceAllString(cellHTML, "")
	clean = strings.TrimSpace(cleanText(clean))
	v, err := strconv.ParseFloat(clean, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

// parseDivision converts "Division I"/"Division II"/… to 0-indexed int (0–3).
func parseDivision(s string) int {
	switch s {
	case "Division I":
		return 0
	case "Division II":
		return 1
	case "Division III":
		return 2
	case "Division IV":
		return 3
	}
	return 0
}

// cleanText strips HTML tags and normalises whitespace.
func cleanText(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return strings.TrimSpace(multiSpaceRe.ReplaceAllString(s, " "))
}

// parseMMRFallback extracts MMR from the embedded Google-Charts history data.
// Only returns 1v1/2v2/3v3 MMR; no tier or division.
func parseMMRFallback(html string) ([]mmr.PlaylistRank, error) {
	type chartPlaylist struct {
		id   int
		name string
	}
	chartPlaylists := []chartPlaylist{
		{10, "Ranked Duel 1v1"},
		{11, "Ranked Doubles 2v2"},
		{13, "Ranked Standard 3v3"},
	}

	matches := chartDataRe.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("rlstats: no rank data found in page")
	}

	parts := strings.Split(matches[len(matches)-1][1], ",")
	var ranks []mmr.PlaylistRank
	for i, pl := range chartPlaylists {
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
		return nil, fmt.Errorf("rlstats: no valid MMR values found")
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