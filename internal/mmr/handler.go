package mmr

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"OOF_RL/internal/httputil"
)

const trackerLookupTimeout = 8 * time.Second

// Handler returns an http.HandlerFunc that looks up MMR ranks for a player.
// p should be a CachedProvider wrapping the real provider in production.
//
// Query params:
//
//	id   — composite player ID in the form "platform|primaryID" (required)
//	name — display name (optional, used for non-Steam masked identity checks)
func Handler(p Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if p == nil {
			httputil.JSONError(w, 503, "tracker service unavailable")
			return
		}

		id := r.URL.Query().Get("id")
		playerName := r.URL.Query().Get("name")
		if id == "" {
			httputil.JSONError(w, 400, "missing id")
			return
		}

		sep := strings.IndexAny(id, "|:_")
		if sep < 1 {
			httputil.JSONError(w, 400, "invalid id format, expected platform|id")
			return
		}
		rawPlatform := id[:sep]
		primaryID := id[sep+1:]
		if end := strings.IndexAny(primaryID, "|:_"); end >= 0 {
			primaryID = primaryID[:end]
		}

		if IsAllAsterisks(primaryID) || (strings.ToLower(rawPlatform) != "steam" && IsAllAsterisks(playerName)) {
			httputil.JSONError(w, 400, "masked player name")
			return
		}

		identity := NewPlayerIdentity(rawPlatform, primaryID, playerName)

		ctx, cancel := context.WithTimeout(r.Context(), trackerLookupTimeout)
		defer cancel()

		ranks, err := p.Lookup(ctx, identity)
		if err != nil {
			log.Printf("[tracker] lookup failed for %s: %v", id, err)
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				httputil.JSONError(w, http.StatusGatewayTimeout, "tracker lookup timed out")
				return
			}
			httputil.JSONError(w, 502, err.Error())
			return
		}

		httputil.WriteJSON(w, map[string]any{
			"fetched_at": time.Now().UTC().Format(time.RFC3339),
			"source":     p.Name(),
			"ranks":      ranks,
		})
	}
}
