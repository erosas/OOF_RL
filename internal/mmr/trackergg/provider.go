package trackergg

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"OOF_RL/internal/mmr"
)

// Provider fetches ranked MMR data from tracker.gg.
type Provider struct {
	limiter *rateLimiter
	client  *http.Client
}

func New() *Provider {
	return &Provider{
		limiter: newRateLimiter(),
		client:  &http.Client{Timeout: 12 * time.Second},
	}
}

func (p *Provider) Name() string { return "tracker.gg" }

func (p *Provider) Supports(platform mmr.Platform) bool { return true }

func (p *Provider) Lookup(id mmr.PlayerIdentity) ([]mmr.PlaylistRank, error) {
	platform, lookup := platformAndLookup(id)
	if platform == "" || lookup == "" {
		return nil, fmt.Errorf("trackergg: cannot build URL for %+v", id)
	}

	trnURL := fmt.Sprintf("https://api.tracker.gg/api/v2/rocket-league/standard/profile/%s/%s",
		url.PathEscape(platform), url.PathEscape(lookup))

	p.limiter.Wait()

	req, err := http.NewRequest(http.MethodGet, trnURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	log.Printf("[trackergg] %s → %d (%d bytes)", trnURL, resp.StatusCode, len(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("trackergg: HTTP %d for %s", resp.StatusCode, trnURL)
	}
	return parseResponse(body)
}

// platformAndLookup maps a PlayerIdentity to the tracker.gg platform slug and
// the lookup key (SteamID64 for Steam, display name for all other platforms).
func platformAndLookup(id mmr.PlayerIdentity) (platform, lookup string) {
	switch id.Platform {
	case mmr.PlatformSteam:
		return "steam", id.PrimaryID
	case mmr.PlatformEpic:
		return "epic", mmr.OrFallback(id.DisplayName, id.PrimaryID)
	case mmr.PlatformPSN:
		return "psn", mmr.OrFallback(id.DisplayName, id.PrimaryID)
	case mmr.PlatformXbox:
		return "xbl", mmr.OrFallback(id.DisplayName, id.PrimaryID)
	case mmr.PlatformSwitch:
		name := mmr.OrFallback(id.DisplayName, id.PrimaryID)
		if mmr.IsAllAsterisks(name) {
			return "", "" // masked Switch identity — no profile to look up
		}
		return "switch", name
	default:
		return string(id.Platform), id.PrimaryID
	}
}


// -- response parsing --

type trnResponse struct {
	Data struct {
		Segments []trnSegment `json:"segments"`
	} `json:"data"`
}

type trnSegment struct {
	Type       string `json:"type"`
	Attributes struct {
		PlaylistID int `json:"playlistId"`
	} `json:"attributes"`
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Stats struct {
		Rating struct {
			Value float64 `json:"value"`
		} `json:"rating"`
		Tier struct {
			Value    int `json:"value"`
			Metadata struct {
				Name    string `json:"name"`
				IconURL string `json:"iconUrl"`
			} `json:"metadata"`
		} `json:"tier"`
		Division struct {
			Value int `json:"value"`
		} `json:"division"`
	} `json:"stats"`
}

func parseResponse(body []byte) ([]mmr.PlaylistRank, error) {
	var resp trnResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("trackergg: parse response: %w", err)
	}
	var ranks []mmr.PlaylistRank
	for _, seg := range resp.Data.Segments {
		if seg.Type != "playlist" || seg.Attributes.PlaylistID == 0 {
			continue
		}
		ranks = append(ranks, mmr.PlaylistRank{
			PlaylistID:   seg.Attributes.PlaylistID,
			PlaylistName: seg.Metadata.Name,
			MMR:          seg.Stats.Rating.Value,
			Tier:         seg.Stats.Tier.Value,
			TierName:     seg.Stats.Tier.Metadata.Name,
			Division:     seg.Stats.Division.Value,
			IconURL:      seg.Stats.Tier.Metadata.IconURL,
		})
	}
	return ranks, nil
}

// -- rate limiter --

// rateLimiter spaces out requests to look organic rather than machine-like.
// Each caller reserves the next slot under a mutex, then sleeps outside it,
// so concurrent callers queue cleanly without holding the lock during sleep.
type rateLimiter struct {
	mu        sync.Mutex
	nextSlot  time.Time
	baseDelay time.Duration
	maxJitter time.Duration
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		baseDelay: 3000 * time.Millisecond,
		maxJitter: 1500 * time.Millisecond, // effective gap: 3–4.5 s
	}
}

func (l *rateLimiter) Wait() {
	l.mu.Lock()
	now := time.Now()
	slot := l.nextSlot
	if slot.Before(now) {
		slot = now
	}
	jitter := time.Duration(rand.Int63n(int64(l.maxJitter) + 1))
	l.nextSlot = slot.Add(l.baseDelay + jitter)
	l.mu.Unlock()
	if wait := slot.Sub(now); wait > 0 {
		time.Sleep(wait)
	}
}
