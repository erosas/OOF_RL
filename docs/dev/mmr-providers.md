# MMR Provider Framework

OOF RL looks up each player's ranked MMR, tier, and division from external stat sites. The provider framework is a small set of types in `internal/mmr/` that define a common contract, handle platform routing, layer in caching, and automatically fall back to a secondary source when the primary is unavailable.

---

## Packages at a glance

| Package | Role |
|---|---|
| `internal/mmr` | Core types: `Provider` interface, `PlayerIdentity`, `PlaylistRank`, `FallbackProvider`, `CachedProvider` |
| `internal/mmr/trackergg` | Provider that calls the tracker.gg API (JSON) |
| `internal/mmr/rlstats` | Provider that scrapes rlstats.net (HTML table parsing) |

---

## The `Provider` interface

Every data source implements three methods:

```go
type Provider interface {
    Name() string
    Supports(platform Platform) bool
    Lookup(ctx context.Context, id PlayerIdentity) ([]PlaylistRank, error)
}
```

**`Name()`** returns a short display string shown in API responses so the caller knows which source served the data (e.g. `"tracker.gg"`, `"rlstats.net"`).

**`Supports(platform)`** lets the framework skip a provider without even making a network call. `trackergg.Provider` returns `true` for all platforms. `rlstats.Provider` returns `false` for `PlatformSwitch` because rlstats.net dropped Nintendo Switch support.

**`Lookup(ctx, id)`** performs the actual fetch and returns a slice of `PlaylistRank`, one entry per ranked playlist the player has played. Providers must honor context cancellation so UI/API requests have bounded wait time.

### `PlayerIdentity`

```go
type PlayerIdentity struct {
    PrimaryID   string   // SteamID64 for Steam; display name for all others
    DisplayName string   // in-game name (always populated)
    Platform    Platform // PlatformSteam | PlatformEpic | PlatformPSN | PlatformXbox | PlatformSwitch
}
```

Build one from the raw strings the RL event system gives you:

```go
id := mmr.NewPlayerIdentity("PS4", "nssbali_", "nssbali_")
```

`NewPlayerIdentity` normalises the raw platform string (`"PS4"`, `"XboxOne"`, `"Epic"`, etc.) to the canonical `Platform` constants.

### `PlaylistRank`

```go
type PlaylistRank struct {
    PlaylistID   int     // 10=duel, 11=doubles, 13=standard, 27=hoops, 28=rumble, …
    PlaylistName string  // "Ranked Duel 1v1", "Ranked Doubles 2v2", …
    MMR          float64
    Tier         int     // numeric tier (0=unranked); populated by trackergg only
    TierName     string  // "Diamond II", "Supersonic Legend", …
    Division     int     // 0-indexed (0=Division I … 3=Division IV)
    IconURL      string  // rank badge URL; populated by trackergg only
}
```

---

## How a lookup flows end-to-end

```
HTTP GET /api/tracker/profile?id=steam|76561198144145654&name=Squishy
        │
        ▼
mmr.Handler
  1. Parse platform + primaryID from the ?id= parameter
  2. Build a PlayerIdentity
  3. Create an 8-second lookup context and call mmrProvider.Lookup(ctx, identity)
        |
        v
CachedProvider.Lookup
  4. Check DB cache (GetTrackerCache); return cached ranks if still fresh
  5. Call the inner provider on a cache miss
        │
        ▼
FallbackProvider.Lookup
  6. Iterate over registered providers from a rotating cursor
  7. Skip any provider where Supports(platform) == false
  8. Call supported providers until one succeeds, or retry/fail by policy
        │
        ▼
trackergg.Provider.Lookup  (or rlstats.Provider.Lookup)
  9. Build the provider-specific URL from the platform slug + identity
 10. Make an HTTP GET with browser-like headers
 11. Parse the response (JSON for trackergg, HTML table for rlstats)
 12. Return []PlaylistRank
        │
        ▼
CachedProvider.Lookup (continued)
 13. JSON-marshal ranks and store in DB cache (UpsertTrackerCache)
 14. Return ranks to the handler
        |
        v
mmr.Handler (continued)
 15. Write {"fetched_at":"...","source":"tracker.gg/rlstats.net","ranks":[...]} to client
```

### Cache key

The cache key is `ranks:platform|primaryID` (e.g. `"ranks:steam|76561198144145654"`). In `main.go`, production wiring currently uses a hard-coded 10-minute TTL. There is no user-editable `config.toml` field for this TTL. A cache hit short-circuits the fallback/provider chain.

---

## The `FallbackProvider`

`FallbackProvider` wraps a list of providers and tries them from a rotating cursor:

```go
trnProvider := mmr.NewFallbackProvider(trackergg.New(), rlstats.New())
```

- trackergg is registered first — it returns richer data (tier number, icon URL, 10 playlists including 4v4 Quads).
- rlstats is also registered — it covers the same 8 main playlists with tier name and division, useful if tracker.gg rate-limits or returns a non-200.
- For **Switch players**, `rlstats.Supports(PlatformSwitch)` returns `false`, so the fallback skips it immediately without a network round-trip and goes straight to trackergg.

If every provider either doesn't support the platform or returns an error, `FallbackProvider.Lookup` returns the last error it saw. If no provider supports the platform at all, it returns `"mmr: no provider supports platform X"`.

---

## The `CachedProvider`

`CachedProvider` is a decorator that wraps any `Provider` with DB-backed caching. It is wired in `main.go` around the `FallbackProvider`:

```go
cached := mmr.NewCachedProvider(
    mmr.NewFallbackProvider(trackergg.New(), rlstats.New()),
    database,
    10*time.Minute,
)
ranks, err := cached.Lookup(ctx, id) // hits DB first, then network
```

Cache entries are stored as JSON under the key `"ranks:platform|primaryID"`.

---

## Adding a new provider

1. **Create a package** under `internal/mmr/<name>/`.

2. **Implement the interface:**

```go
type Provider struct { /* http.Client, config, etc. */ }

func (p *Provider) Name() string { return "mysite.com" }

func (p *Provider) Supports(platform mmr.Platform) bool {
    // return false for any platform this site doesn't cover
    return platform != mmr.PlatformSwitch
}

func (p *Provider) Lookup(ctx context.Context, id mmr.PlayerIdentity) ([]mmr.PlaylistRank, error) {
    // fetch + parse; map your site's playlist names to the canonical IDs below
    // use http.NewRequestWithContext(ctx, ...) for network calls
}
```

3. **Map playlist names to canonical IDs.** The IDs match the RL API and tracker.gg:

| ID | Name |
|---|---|
| 10 | Ranked Duel 1v1 |
| 11 | Ranked Doubles 2v2 |
| 13 | Ranked Standard 3v3 |
| 27 | Hoops |
| 28 | Rumble |
| 29 | Dropshot |
| 30 | Snowday |
| 34 | Tournament Matches |
| 61 | Ranked 4v4 Quads |
| 63 | Heatseeker |

4. **Register it** in `main.go`:

```go
trnProvider := mmr.NewCachedProvider(
    mmr.NewFallbackProvider(trackergg.New(), rlstats.New(), mysite.New()),
    database,
    trackerCacheTTL,
)
```

5. **Add tests.** Write a `capture_test.go` (`//go:build manual`) that hits the live site and saves response fixtures under `internal/mmr/testdata/<name>/`. Then write a `provider_test.go` (no build tag) that parses those fixtures offline — these run in CI without any network dependency.

---

## Platform support matrix

| Platform | trackergg | rlstats |
|---|---|---|
| Steam | ✓ | ✓ |
| Epic | ✓ | ✓ |
| PSN | ✓ | ✓ |
| Xbox | ✓ | ✓ |
| Switch | ✓ | ✗ (deprecated by rlstats.net) |
