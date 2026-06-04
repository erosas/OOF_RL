package mmr

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"
)

const (
	maxAttempts    = 3
	providerDelay  = 2 * time.Second  // between consecutive providers in a round
	retryBaseDelay = 10 * time.Second // before cycling back to the same provider
)

// FallbackProvider tries providers sequentially in round-robin order.
// A cursor advances on each call so no single provider is always hit first.
// Providers that don't support the requested platform are skipped.
// A short delay separates each provider within a round; a longer delay
// is applied before cycling back to re-hit the same provider.
type FallbackProvider struct {
	providers []Provider
	cursor    atomic.Uint64
}

func NewFallbackProvider(providers ...Provider) *FallbackProvider {
	return &FallbackProvider{providers: providers}
}

func (f *FallbackProvider) Name() string {
	names := make([]string, len(f.providers))
	for i, p := range f.providers {
		names[i] = p.Name()
	}
	return strings.Join(names, "/")
}

func (f *FallbackProvider) Supports(platform Platform) bool {
	for _, p := range f.providers {
		if p.Supports(platform) {
			return true
		}
	}
	return false
}

func (f *FallbackProvider) Lookup(ctx context.Context, id PlayerIdentity) ([]PlaylistRank, error) {
	n := len(f.providers)
	if n == 0 {
		return nil, fmt.Errorf("mmr: no providers configured")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Advance cursor so each call starts on a different provider.
	start := int(f.cursor.Add(1)-1) % n

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt) * retryBaseDelay
			log.Printf("[mmr] all providers failed for %s|%s — retry %d/%d in %s",
				id.Platform, id.PrimaryID, attempt, maxAttempts-1, delay)
			if err := waitContext(ctx, delay); err != nil {
				return nil, err
			}
		}

		tried := 0
		permanentCount := 0
		for i := 0; i < n; i++ {
			p := f.providers[(start+i)%n]
			if !p.Supports(id.Platform) {
				continue
			}
			if tried > 0 {
				if err := waitContext(ctx, providerDelay); err != nil {
					return nil, err
				}
			}
			tried++
			ranks, err := p.Lookup(ctx, id)
			if err == nil {
				return ranks, nil
			}
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			log.Printf("[mmr] %s failed for %s|%s: %v", p.Name(), id.Platform, id.PrimaryID, err)
			lastErr = err
			if IsPermanent(err) {
				permanentCount++
			}
		}

		if tried == 0 {
			return nil, fmt.Errorf("mmr: no provider supports platform %s", id.Platform)
		}
		// All tried providers returned a permanent error (e.g. HTTP 403/404).
		// Retrying will not help; return immediately.
		if permanentCount == tried {
			return nil, lastErr
		}
	}
	return nil, lastErr
}

func waitContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
