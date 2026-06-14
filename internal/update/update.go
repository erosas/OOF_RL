// Package update implements the update checker: fetch a release manifest,
// compare versions, and surface release links for the user to download in
// their browser.
//
// This is host-core rather than a plugin because it reports on the host
// binary itself and its routes are host-reserved. The app never downloads
// release artifacts: the manifest is unsigned, so a SHA256 from the same
// manifest as the artifact URL proves transport integrity, not authorship.
// Instead the UI shows a dialog linking to the GitHub release page and the
// user downloads with their browser. As a second guard, manifest URLs are
// only surfaced to the UI when they point inside the project's GitHub repo
// (see SafeReleaseURL).
package update

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DefaultManifestURL is the stable channel: GitHub's releases/latest excludes
// prereleases, so regular users only ever see stable releases here.
const DefaultManifestURL = "https://github.com/erosas/OOF_RL/releases/latest/download/update-manifest.json"

// DevManifestURL is the dev channel: a rolling "dev" prerelease whose manifest
// asset is refreshed on every release (dev and stable), so dev-mode users are
// offered the newest build of either kind. See docs/dev/auto-update.md.
const DevManifestURL = "https://github.com/erosas/OOF_RL/releases/download/dev/update-manifest.json"

// Manifest is the update-manifest.json attached to each GitHub release.
type Manifest struct {
	Version        string `json:"version"`
	Channel        string `json:"channel"`
	NotesURL       string `json:"notes_url"`
	PublishedAt    string `json:"published_at"`
	ArtifactURL    string `json:"artifact_url"`
	ArtifactName   string `json:"artifact_name"`
	ArtifactSHA256 string `json:"artifact_sha256"`
}

// Status is the polling shape served by GET /api/update/status.
type Status struct {
	CurrentVersion  string `json:"current_version"`
	LastCheckedAt   string `json:"last_checked_at,omitempty"`
	LatestVersion   string `json:"latest_version,omitempty"`
	NotesURL        string `json:"notes_url,omitempty"`
	DownloadURL     string `json:"download_url,omitempty"`
	UpdateAvailable bool   `json:"update_available"`
	LastError       string `json:"last_error,omitempty"`
}

// Checker holds update state. All exported methods are safe for concurrent
// use.
type Checker struct {
	stableURL string
	devURL    string
	isDev     func() bool
	version   string
	client    *http.Client

	mu     sync.Mutex
	status Status
}

// New creates a Checker for the given running version (config.AppVersion).
// isDev is read on every check so toggling dev mode in Settings switches
// channels without a restart; a nil isDev pins the checker to stable.
func New(version string, isDev func() bool) *Checker {
	if isDev == nil {
		isDev = func() bool { return false }
	}
	return &Checker{
		stableURL: DefaultManifestURL,
		devURL:    DevManifestURL,
		isDev:     isDev,
		version:   version,
		client:    &http.Client{Timeout: 30 * time.Second},
		status:    Status{CurrentVersion: version},
	}
}

// manifestURL selects the channel to poll based on the live dev-mode setting.
func (c *Checker) manifestURL() string {
	if c.isDev() {
		return c.devURL
	}
	return c.stableURL
}

// Status returns a snapshot of the current update state.
func (c *Checker) Status() Status {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

// startupCheckDelay keeps the first manifest fetch out of the app's startup
// path. Variable so tests can shrink it.
var startupCheckDelay = 15 * time.Second

// RunPeriodic checks once after a short startup delay, then every interval
// until ctx is cancelled.
func (c *Checker) RunPeriodic(ctx context.Context, interval time.Duration) {
	select {
	case <-time.After(startupCheckDelay):
	case <-ctx.Done():
		return
	}
	c.Check(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.Check(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// Check fetches and validates the manifest, then records whether the
// manifest version is newer than the running version.
func (c *Checker) Check(ctx context.Context) Status {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.manifestURL(), nil)
	if err != nil {
		return c.failCheck("bad manifest request: " + err.Error())
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return c.failCheck(err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.failCheck(fmt.Sprintf("manifest HTTP %d", resp.StatusCode))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return c.failCheck("read manifest: " + err.Error())
	}
	manifest, err := ParseManifest(body)
	if err != nil {
		return c.failCheck(err.Error())
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.status = Status{
		CurrentVersion:  c.version,
		LastCheckedAt:   time.Now().UTC().Format(time.RFC3339),
		LatestVersion:   manifest.Version,
		NotesURL:        SafeReleaseURL(manifest.NotesURL),
		DownloadURL:     SafeReleaseURL(manifest.ArtifactURL),
		UpdateAvailable: IsNewer(c.version, manifest.Version),
	}
	return c.status
}

func (c *Checker) failCheck(msg string) Status {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status.LastCheckedAt = time.Now().UTC().Format(time.RFC3339)
	c.status.LastError = msg
	return c.status
}

// ParseManifest decodes and validates an update manifest. A UTF-8 BOM is
// tolerated — Windows tooling (PowerShell 5.1 Out-File) likes to prepend one.
// Only version is required: the checker compares versions and links out, it
// never fetches artifacts.
func ParseManifest(data []byte) (Manifest, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return m, fmt.Errorf("invalid manifest JSON: %w", err)
	}
	m.Version = strings.TrimSpace(m.Version)
	m.Channel = strings.TrimSpace(m.Channel)
	m.NotesURL = strings.TrimSpace(m.NotesURL)
	m.PublishedAt = strings.TrimSpace(m.PublishedAt)
	m.ArtifactURL = strings.TrimSpace(m.ArtifactURL)
	m.ArtifactName = strings.TrimSpace(m.ArtifactName)
	m.ArtifactSHA256 = NormalizeSHA256(m.ArtifactSHA256)
	if m.Version == "" {
		return m, fmt.Errorf("manifest version required")
	}
	return m, nil
}

// IsNewer reports whether latest is a newer release than current using
// semver-2 precedence: (v)MAJOR.MINOR.PATCH with an optional -prerelease
// suffix. A prerelease ranks below its release (v1.2.3-dev.1 < v1.2.3) and
// prerelease identifiers compare per the semver spec (numeric numerically,
// numeric below alphanumeric, otherwise ASCII order). This is what keeps a dev
// build from prompting a phantom downgrade to an earlier dev build, and makes
// a stable release supersede its own prereleases. When either side isn't
// semver-shaped (e.g. a local "dev" build), any difference counts as newer so
// dev builds still surface releases.
func IsNewer(current, latest string) bool {
	current = strings.TrimSpace(current)
	latest = strings.TrimSpace(latest)
	if latest == "" || current == latest {
		return false
	}
	cv, cok := parseSemver(current)
	lv, lok := parseSemver(latest)
	if cok && lok {
		return compareSemver(lv, cv) > 0
	}
	return true
}

// semver is a parsed (v)MAJOR.MINOR.PATCH version with optional dot-separated
// prerelease identifiers; pre is empty for a release.
type semver struct {
	nums [3]int
	pre  []string
}

func parseSemver(v string) (semver, bool) {
	var s semver
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	// Build metadata (everything after '+') is ignored for precedence.
	if plus := strings.IndexByte(v, '+'); plus >= 0 {
		v = v[:plus]
	}
	core := v
	if dash := strings.IndexByte(v, '-'); dash >= 0 {
		core = v[:dash]
		pre := v[dash+1:]
		if pre == "" {
			return s, false
		}
		s.pre = strings.Split(pre, ".")
		for _, id := range s.pre {
			if id == "" {
				return s, false
			}
		}
	}
	// Require a full MAJOR.MINOR.PATCH core. A short core (v1, v1.2) isn't
	// semver-shaped, so it falls back to IsNewer's "any difference is newer"
	// path rather than silently comparing with implied-zero components.
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return s, false
	}
	for i, part := range parts {
		n, ok := atoiStrict(part)
		if !ok {
			return s, false
		}
		s.nums[i] = n
	}
	return s, true
}

// compareSemver returns -1, 0, or 1 as a is less than, equal to, or greater
// than b under semver-2 precedence.
func compareSemver(a, b semver) int {
	for i := range a.nums {
		if a.nums[i] != b.nums[i] {
			if a.nums[i] > b.nums[i] {
				return 1
			}
			return -1
		}
	}
	// Equal core: a release outranks any prerelease of the same core.
	switch {
	case len(a.pre) == 0 && len(b.pre) == 0:
		return 0
	case len(a.pre) == 0:
		return 1
	case len(b.pre) == 0:
		return -1
	}
	for i := 0; i < len(a.pre) && i < len(b.pre); i++ {
		if c := comparePreID(a.pre[i], b.pre[i]); c != 0 {
			return c
		}
	}
	// All shared identifiers equal: more identifiers outrank fewer.
	switch {
	case len(a.pre) > len(b.pre):
		return 1
	case len(a.pre) < len(b.pre):
		return -1
	default:
		return 0
	}
}

// comparePreID compares two prerelease identifiers per semver-2: numeric
// identifiers compare numerically and rank below alphanumeric ones.
func comparePreID(a, b string) int {
	an, aNum := atoiStrict(a)
	bn, bNum := atoiStrict(b)
	switch {
	case aNum && bNum:
		switch {
		case an > bn:
			return 1
		case an < bn:
			return -1
		default:
			return 0
		}
	case aNum:
		return -1
	case bNum:
		return 1
	default:
		return strings.Compare(a, b)
	}
}

func atoiStrict(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	// Semver-2 numeric identifiers must not carry leading zeros; "0" is valid
	// but "01" is not numeric and is compared as an alphanumeric identifier.
	if len(s) > 1 && s[0] == '0' {
		return 0, false
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}

// SafeReleaseURL returns raw only when it points inside the project's GitHub
// repo over HTTPS; otherwise "". Applied to every manifest URL before it
// reaches the UI: the manifest is unsigned, so without this allowlist a
// tampered manifest could put an arbitrary link in a dialog the user is
// primed to click.
func SafeReleaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if u.Scheme != "https" || u.Host != "github.com" || u.User != nil {
		return ""
	}
	if !strings.HasPrefix(u.Path, "/erosas/OOF_RL/") {
		return ""
	}
	// Reject dot segments and backslashes in both the raw and the decoded
	// path, so percent-encoded forms (%2e%2e, %5c) can't smuggle traversal
	// past the prefix check; release/tag paths never contain either.
	for _, p := range []string{u.EscapedPath(), u.Path} {
		if strings.Contains(p, "..") || strings.Contains(p, "\\") {
			return ""
		}
	}
	return raw
}

// NormalizeSHA256 lowercases and validates a hex SHA256; returns "" if the
// value is not exactly 64 hex characters.
func NormalizeSHA256(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) != 64 {
		return ""
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return ""
		}
	}
	return value
}
