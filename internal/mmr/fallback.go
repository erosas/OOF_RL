package mmr

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

const providerDelay = 2 * time.Second // between consecutive providers

// FallbackProvider tries providers in order until one succeeds.
// Providers that don't support the requested platform are skipped.
// A short delay separates consecutive provider attempts.
type FallbackProvider struct {
	providers []Provider
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

	var lastErr error
	tried := 0
	for i := 0; i < n; i++ {
		p := f.providers[i]
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
	}

	if tried == 0 {
		return nil, fmt.Errorf("mmr: no provider supports platform %s", id.Platform)
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
