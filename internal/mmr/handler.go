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

const trackerLookupTimeout = 15 * time.Second

// metaLookuper is an optional Provider extension that also returns when the ranks
// were actually fetched upstream. CachedProvider implements it.
type metaLookuper interface {
	LookupMeta(ctx context.Context, id PlayerIdentity) ([]PlaylistRank, time.Time, error)
}

// Handler returns an http.HandlerFunc that looks up MMR ranks for a player.
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

		// Prefer the real upstream fetch time when the provider can report it (the
		// cache does), so "fetched_at" reflects true data age instead of resetting
		// to now on every cache-served poll.
		var (
			ranks     []PlaylistRank
			fetchedAt time.Time
			err       error
		)
		if mp, ok := p.(metaLookuper); ok {
			ranks, fetchedAt, err = mp.LookupMeta(ctx, identity)
		} else {
			ranks, err = p.Lookup(ctx, identity)
			fetchedAt = time.Now()
		}
		if err != nil {
			log.Printf("[tracker] lookup failed for %s: %v", id, err)
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				httputil.JSONError(w, http.StatusGatewayTimeout, "tracker lookup timed out")
				return
			}
			httputil.JSONError(w, 502, err.Error())
			return
		}
		if fetchedAt.IsZero() {
			fetchedAt = time.Now()
		}

		httputil.WriteJSON(w, map[string]any{
			"fetched_at": fetchedAt.UTC().Format(time.RFC3339),
			"source":     p.Name(),
			"ranks":      ranks,
		})
	}
}
