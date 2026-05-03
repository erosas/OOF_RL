package mmr

import "log"

// FallbackProvider tries each provider in order, returning the first success.
// On error from a provider it logs and moves to the next.
type FallbackProvider struct {
	providers []Provider
}

func NewFallbackProvider(providers ...Provider) *FallbackProvider {
	return &FallbackProvider{providers: providers}
}

func (f *FallbackProvider) Name() string {
	names := ""
	for i, p := range f.providers {
		if i > 0 {
			names += "/"
		}
		names += p.Name()
	}
	return names
}

func (f *FallbackProvider) Lookup(id PlayerIdentity) ([]PlaylistRank, error) {
	var lastErr error
	for _, p := range f.providers {
		ranks, err := p.Lookup(id)
		if err == nil {
			return ranks, nil
		}
		log.Printf("[mmr] %s failed for %s|%s: %v — trying next provider", p.Name(), id.Platform, id.PrimaryID, err)
		lastErr = err
	}
	return nil, lastErr
}