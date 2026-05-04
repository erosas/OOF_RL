package mmr

import (
	"encoding/json"
	"time"
)

// CacheStore is the persistence contract for MMR result caching.
// db.DB satisfies this interface via its GetTrackerCache / UpsertTrackerCache methods.
type CacheStore interface {
	GetTrackerCache(key string) (data string, fetchedAt time.Time, found bool, err error)
	UpsertTrackerCache(key string, data string) error
}

// CachedProvider wraps a Provider and caches results in a CacheStore.
// Cached entries store []PlaylistRank as JSON under a "ranks:" prefixed key so
// they don't collide with the raw TRN blobs stored by the legacy server handler.
type CachedProvider struct {
	inner Provider
	store CacheStore
	ttl   time.Duration
}

// NewCachedProvider wraps p with DB-backed caching. ttl=0 disables caching.
func NewCachedProvider(p Provider, store CacheStore, ttl time.Duration) *CachedProvider {
	return &CachedProvider{inner: p, store: store, ttl: ttl}
}

func (c *CachedProvider) Name() string                    { return c.inner.Name() }
func (c *CachedProvider) Supports(p Platform) bool        { return c.inner.Supports(p) }

func (c *CachedProvider) Lookup(id PlayerIdentity) ([]PlaylistRank, error) {
	key := "ranks:" + string(id.Platform) + "|" + id.PrimaryID

	if c.ttl > 0 && c.store != nil {
		data, fetchedAt, found, err := c.store.GetTrackerCache(key)
		if err == nil && found && time.Since(fetchedAt) < c.ttl {
			var ranks []PlaylistRank
			if json.Unmarshal([]byte(data), &ranks) == nil {
				return ranks, nil
			}
		}
	}

	ranks, err := c.inner.Lookup(id)
	if err != nil {
		return nil, err
	}

	if c.ttl > 0 && c.store != nil {
		if b, merr := json.Marshal(ranks); merr == nil {
			_ = c.store.UpsertTrackerCache(key, string(b))
		}
	}

	return ranks, nil
}