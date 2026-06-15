package mmr

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"
)

// CacheStore is the persistence contract for MMR result caching.
// db.DB satisfies this interface via its GetTrackerCache / UpsertTrackerCache methods.
type CacheStore interface {
	GetTrackerCache(key string) (data string, fetchedAt time.Time, found bool, err error)
	UpsertTrackerCache(key string, data string) error
}

// cacheEntry is the JSON payload persisted per player: the ranks plus the cache
// generation they were fetched under. A generation bump (Invalidate, called at
// match boundaries) makes existing entries stale without deleting them, so they
// remain available to serve stale if a re-fetch fails.
type cacheEntry struct {
	Gen   int64          `json:"gen"`
	Ranks []PlaylistRank `json:"ranks"`
}

// CachedProvider wraps a Provider and caches results in a CacheStore.
//
// It is the layer that makes "fetch every time" cheap and safe regardless of how
// chatty callers are:
//   - within a generation, a fresh entry short-circuits the upstream call;
//   - Invalidate bumps the generation so each new match re-fetches once;
//   - an upstream failure serves the last good value instead of propagating an
//     error, so a transient rate-limit never flips a populated rank to an error;
//   - concurrent identical lookups collapse into a single upstream call.
//
// Cached entries are stored as cacheEntry JSON under a "ranks:" prefixed key so
// they don't collide with the raw TRN blobs stored by the legacy server handler.
type CachedProvider struct {
	inner Provider
	store CacheStore
	ttl   time.Duration

	gen atomic.Int64

	mu       sync.Mutex
	inflight map[string]*inflightCall
}

// inflightCall is a single shared upstream lookup that concurrent callers wait on.
type inflightCall struct {
	done  chan struct{}
	ranks []PlaylistRank
	err   error
}

// NewCachedProvider wraps p with DB-backed caching. ttl=0 disables caching.
func NewCachedProvider(p Provider, store CacheStore, ttl time.Duration) *CachedProvider {
	return &CachedProvider{inner: p, store: store, ttl: ttl, inflight: make(map[string]*inflightCall)}
}

func (c *CachedProvider) Name() string             { return c.inner.Name() }
func (c *CachedProvider) Supports(p Platform) bool { return c.inner.Supports(p) }

// Invalidate marks every cached entry stale: the next lookup per player re-fetches
// once, while the stored values remain available to serve stale if a re-fetch
// fails. Call it at match boundaries so each match shows fresh MMR without
// re-scraping on every in-match update. Safe for concurrent use.
func (c *CachedProvider) Invalidate() { c.gen.Add(1) }

func (c *CachedProvider) Lookup(ctx context.Context, id PlayerIdentity) ([]PlaylistRank, error) {
	key := "ranks:" + string(id.Platform) + "|" + id.PrimaryID
	gen := c.gen.Load()
	cacheable := c.ttl > 0 && c.store != nil

	// Read whatever we have once; it backs both the fresh-hit path and the
	// serve-stale-on-error fallback below.
	var (
		cached     cacheEntry
		haveCached bool
		fetchedAt  time.Time
	)
	if cacheable {
		if data, fa, found, err := c.store.GetTrackerCache(key); err == nil && found {
			if json.Unmarshal([]byte(data), &cached) == nil {
				haveCached = true
				fetchedAt = fa
			}
		}
		// Fresh hit: same generation and within TTL.
		if haveCached && cached.Gen == gen && time.Since(fetchedAt) < c.ttl {
			return cached.Ranks, nil
		}
	}

	ranks, err := c.lookupSingle(ctx, key, id, gen, cacheable)
	if err != nil {
		// Serve stale rather than propagate a transient upstream failure.
		if haveCached && cached.Ranks != nil {
			return cached.Ranks, nil
		}
		return nil, err
	}
	return ranks, nil
}

// lookupSingle ensures only one in-flight upstream call exists per key; concurrent
// callers wait for and share its result instead of each hitting the provider. The
// leader (the goroutine that performs the call) is also the only one that writes
// the result to the cache, so followers don't issue redundant store writes.
func (c *CachedProvider) lookupSingle(ctx context.Context, key string, id PlayerIdentity, gen int64, cacheable bool) ([]PlaylistRank, error) {
	c.mu.Lock()
	if call, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		select {
		case <-call.done:
			return call.ranks, call.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	call := &inflightCall{done: make(chan struct{})}
	c.inflight[key] = call
	c.mu.Unlock()

	call.ranks, call.err = c.inner.Lookup(ctx, id)
	if call.err == nil && cacheable {
		if b, merr := json.Marshal(cacheEntry{Gen: gen, Ranks: call.ranks}); merr == nil {
			_ = c.store.UpsertTrackerCache(key, string(b))
		}
	}
	close(call.done)

	c.mu.Lock()
	delete(c.inflight, key)
	c.mu.Unlock()

	return call.ranks, call.err
}