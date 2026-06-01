package mmr_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"OOF_RL/internal/mmr"
)

// ---- PermanentError --------------------------------------------------------

func TestPermanentError_ErrorAndUnwrap(t *testing.T) {
	inner := errors.New("not found")
	err := &mmr.PermanentError{Err: inner}
	if err.Error() != "not found" {
		t.Errorf("Error() = %q, want %q", err.Error(), "not found")
	}
	if err.Unwrap() != inner {
		t.Error("Unwrap() did not return inner error")
	}
}

func TestIsPermanent(t *testing.T) {
	perm := mmr.Permanentf("gone: %d", 404)
	if !mmr.IsPermanent(perm) {
		t.Error("IsPermanent(Permanentf(...)) should be true")
	}
	if mmr.IsPermanent(errors.New("transient")) {
		t.Error("IsPermanent(plain error) should be false")
	}
	wrapped := fmt.Errorf("wrap: %w", perm)
	if !mmr.IsPermanent(wrapped) {
		t.Error("IsPermanent(wrapped permanent) should be true")
	}
}

func TestPermanentf(t *testing.T) {
	err := mmr.Permanentf("http %d: %s", 403, "forbidden")
	if err.Error() != "http 403: forbidden" {
		t.Errorf("Permanentf() = %q, want %q", err.Error(), "http 403: forbidden")
	}
	if !mmr.IsPermanent(err) {
		t.Error("Permanentf result should be permanent")
	}
}

// ---- NormalizePlatform -----------------------------------------------------

func TestNormalizePlatform(t *testing.T) {
	cases := []struct {
		in   string
		want mmr.Platform
	}{
		{"steam", mmr.PlatformSteam},
		{"Steam", mmr.PlatformSteam},
		{"STEAM", mmr.PlatformSteam},
		{"epic", mmr.PlatformEpic},
		{"epicgames", mmr.PlatformEpic},
		{"ps4", mmr.PlatformPSN},
		{"ps5", mmr.PlatformPSN},
		{"psn", mmr.PlatformPSN},
		{"playstation", mmr.PlatformPSN},
		{"xboxone", mmr.PlatformXbox},
		{"xbox", mmr.PlatformXbox},
		{"xbl", mmr.PlatformXbox},
		{"nintendo", mmr.PlatformSwitch},
		{"switch", mmr.PlatformSwitch},
		{"unknown", mmr.Platform("unknown")},
		{"MyPlatform", mmr.Platform("myplatform")},
	}
	for _, tc := range cases {
		got := mmr.NormalizePlatform(tc.in)
		if got != tc.want {
			t.Errorf("NormalizePlatform(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ---- strutil ---------------------------------------------------------------

func TestOrFallback(t *testing.T) {
	if got := mmr.OrFallback("a", "b"); got != "a" {
		t.Errorf("OrFallback(non-empty, ...) = %q, want %q", got, "a")
	}
	if got := mmr.OrFallback("", "b"); got != "b" {
		t.Errorf("OrFallback(\"\", fallback) = %q, want %q", got, "b")
	}
}

func TestIsAllAsterisks(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"***", true},
		{"*", true},
		{"", false},
		{"*a*", false},
		{"abc", false},
	}
	for _, tc := range cases {
		if got := mmr.IsAllAsterisks(tc.in); got != tc.want {
			t.Errorf("IsAllAsterisks(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// ---- NewPlayerIdentity -----------------------------------------------------

func TestNewPlayerIdentity(t *testing.T) {
	id := mmr.NewPlayerIdentity("Steam", "76561198025501695", "RocketPlayer")
	if id.Platform != mmr.PlatformSteam {
		t.Errorf("Platform = %q, want %q", id.Platform, mmr.PlatformSteam)
	}
	if id.PrimaryID != "76561198025501695" {
		t.Errorf("PrimaryID = %q", id.PrimaryID)
	}
	if id.DisplayName != "RocketPlayer" {
		t.Errorf("DisplayName = %q", id.DisplayName)
	}
}

func TestNewPlayerIdentity_EmptyDisplayName_FallsBackToPrimaryID(t *testing.T) {
	id := mmr.NewPlayerIdentity("Epic", "epicuser123", "")
	if id.DisplayName != "epicuser123" {
		t.Errorf("DisplayName = %q, want %q", id.DisplayName, "epicuser123")
	}
}

// ---- CachedProvider helpers ------------------------------------------------

type stubProvider struct {
	name     string
	supports bool
	ranks    []mmr.PlaylistRank
	err      error
	calls    int
}

func (s *stubProvider) Name() string                                   { return s.name }
func (s *stubProvider) Supports(_ mmr.Platform) bool                   { return s.supports }
func (s *stubProvider) Lookup(_ mmr.PlayerIdentity) ([]mmr.PlaylistRank, error) {
	s.calls++
	return s.ranks, s.err
}

type stubStore struct {
	data      map[string]string
	fetchedAt map[string]time.Time
	getErr    error
}

func newStubStore() *stubStore {
	return &stubStore{data: map[string]string{}, fetchedAt: map[string]time.Time{}}
}

func (s *stubStore) GetTrackerCache(key string) (string, time.Time, bool, error) {
	if s.getErr != nil {
		return "", time.Time{}, false, s.getErr
	}
	d, ok := s.data[key]
	return d, s.fetchedAt[key], ok, nil
}

func (s *stubStore) UpsertTrackerCache(key, data string) error {
	s.data[key] = data
	s.fetchedAt[key] = time.Now()
	return nil
}

// ---- CachedProvider --------------------------------------------------------

func TestCachedProvider_Passthrough(t *testing.T) {
	inner := &stubProvider{name: "prov", supports: true}
	cp := mmr.NewCachedProvider(inner, newStubStore(), time.Minute)
	if cp.Name() != "prov" {
		t.Errorf("Name() = %q, want %q", cp.Name(), "prov")
	}
	if !cp.Supports(mmr.PlatformSteam) {
		t.Error("Supports should delegate to inner")
	}
}

func TestCachedProvider_CacheMiss_CallsInner(t *testing.T) {
	inner := &stubProvider{name: "p", supports: true, ranks: []mmr.PlaylistRank{{PlaylistID: 10, MMR: 500}}}
	cp := mmr.NewCachedProvider(inner, newStubStore(), time.Minute)

	ranks, err := cp.Lookup(mmr.PlayerIdentity{Platform: mmr.PlatformSteam, PrimaryID: "123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranks) != 1 || ranks[0].MMR != 500 {
		t.Errorf("unexpected ranks: %+v", ranks)
	}
	if inner.calls != 1 {
		t.Errorf("inner calls = %d, want 1", inner.calls)
	}
}

func TestCachedProvider_CacheHit_SkipsInner(t *testing.T) {
	inner := &stubProvider{name: "p", supports: true, ranks: []mmr.PlaylistRank{{PlaylistID: 10, MMR: 500}}}
	cp := mmr.NewCachedProvider(inner, newStubStore(), time.Minute)
	id := mmr.PlayerIdentity{Platform: mmr.PlatformSteam, PrimaryID: "123"}

	_, _ = cp.Lookup(id) // populate cache
	_, err := cp.Lookup(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.calls != 1 {
		t.Errorf("inner calls = %d, want 1 (second call should hit cache)", inner.calls)
	}
}

func TestCachedProvider_CacheExpired_RefetchesInner(t *testing.T) {
	inner := &stubProvider{name: "p", supports: true, ranks: []mmr.PlaylistRank{{PlaylistID: 10, MMR: 500}}}
	cp := mmr.NewCachedProvider(inner, newStubStore(), time.Millisecond)
	id := mmr.PlayerIdentity{Platform: mmr.PlatformSteam, PrimaryID: "123"}

	_, _ = cp.Lookup(id)
	time.Sleep(5 * time.Millisecond) // let TTL expire
	_, err := cp.Lookup(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inner.calls != 2 {
		t.Errorf("inner calls = %d, want 2 (expired entry should re-fetch)", inner.calls)
	}
}

func TestCachedProvider_TTLZero_NoCaching(t *testing.T) {
	inner := &stubProvider{name: "p", supports: true, ranks: []mmr.PlaylistRank{{PlaylistID: 10}}}
	cp := mmr.NewCachedProvider(inner, newStubStore(), 0)
	id := mmr.PlayerIdentity{Platform: mmr.PlatformSteam, PrimaryID: "123"}

	_, _ = cp.Lookup(id)
	_, _ = cp.Lookup(id)
	if inner.calls != 2 {
		t.Errorf("inner calls = %d, want 2 (ttl=0 disables caching)", inner.calls)
	}
}

func TestCachedProvider_InnerError_NotCached(t *testing.T) {
	inner := &stubProvider{name: "p", supports: true, err: errors.New("provider down")}
	store := newStubStore()
	cp := mmr.NewCachedProvider(inner, store, time.Minute)

	_, err := cp.Lookup(mmr.PlayerIdentity{Platform: mmr.PlatformSteam, PrimaryID: "123"})
	if err == nil {
		t.Fatal("expected error")
	}
	if len(store.data) != 0 {
		t.Error("error result should not be written to cache")
	}
}

func TestCachedProvider_StoreGetError_FallsThrough(t *testing.T) {
	inner := &stubProvider{name: "p", supports: true, ranks: []mmr.PlaylistRank{{PlaylistID: 10, MMR: 700}}}
	store := &stubStore{data: map[string]string{}, fetchedAt: map[string]time.Time{}, getErr: errors.New("db error")}
	cp := mmr.NewCachedProvider(inner, store, time.Minute)

	ranks, err := cp.Lookup(mmr.PlayerIdentity{Platform: mmr.PlatformSteam, PrimaryID: "123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranks) == 0 || ranks[0].MMR != 700 {
		t.Errorf("unexpected ranks: %+v", ranks)
	}
	if inner.calls != 1 {
		t.Errorf("inner calls = %d, want 1", inner.calls)
	}
}

// ---- FallbackProvider ------------------------------------------------------

func TestFallbackProvider_Name(t *testing.T) {
	fp := mmr.NewFallbackProvider(&stubProvider{name: "tgg"}, &stubProvider{name: "rlstats"})
	if fp.Name() != "tgg/rlstats" {
		t.Errorf("Name() = %q, want %q", fp.Name(), "tgg/rlstats")
	}
}

func TestFallbackProvider_Supports(t *testing.T) {
	fp := mmr.NewFallbackProvider(&stubProvider{name: "a", supports: false}, &stubProvider{name: "b", supports: true})
	if !fp.Supports(mmr.PlatformSteam) {
		t.Error("Supports should be true if any inner provider supports the platform")
	}
	fp2 := mmr.NewFallbackProvider(&stubProvider{name: "a", supports: false})
	if fp2.Supports(mmr.PlatformSteam) {
		t.Error("Supports should be false if no inner provider supports the platform")
	}
}

func TestFallbackProvider_NoProviders(t *testing.T) {
	fp := mmr.NewFallbackProvider()
	_, err := fp.Lookup(mmr.PlayerIdentity{Platform: mmr.PlatformSteam, PrimaryID: "x"})
	if err == nil {
		t.Fatal("expected error with no providers")
	}
}

func TestFallbackProvider_NoSupportingProvider(t *testing.T) {
	fp := mmr.NewFallbackProvider(&stubProvider{name: "p", supports: false})
	_, err := fp.Lookup(mmr.PlayerIdentity{Platform: mmr.PlatformSteam, PrimaryID: "x"})
	if err == nil {
		t.Fatal("expected error when no provider supports the platform")
	}
}

func TestFallbackProvider_FirstSucceeds(t *testing.T) {
	ranks := []mmr.PlaylistRank{{PlaylistID: 10, MMR: 750}}
	fp := mmr.NewFallbackProvider(&stubProvider{name: "p", supports: true, ranks: ranks})

	got, err := fp.Lookup(mmr.PlayerIdentity{Platform: mmr.PlatformSteam, PrimaryID: "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].MMR != 750 {
		t.Errorf("unexpected ranks: %+v", got)
	}
}

func TestFallbackProvider_PermanentError_NoRetry(t *testing.T) {
	// Single provider returning PermanentError must not trigger the retry loop.
	// If the retry loop ran it would sleep 10+ seconds; the test would time out.
	p := &stubProvider{name: "p", supports: true, err: mmr.Permanentf("http 404")}
	fp := mmr.NewFallbackProvider(p)

	_, err := fp.Lookup(mmr.PlayerIdentity{Platform: mmr.PlatformSteam, PrimaryID: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if p.calls != 1 {
		t.Errorf("provider called %d times, want 1 (permanent error must not retry)", p.calls)
	}
}