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

const DefaultManifestURL = "https://github.com/erosas/OOF_RL/releases/latest/download/update-manifest.json"

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
	manifestURL string
	version     string
	client      *http.Client

	mu     sync.Mutex
	status Status
}

// New creates a Checker for the given running version (config.AppVersion).
func New(version string) *Checker {
	return &Checker{
		manifestURL: DefaultManifestURL,
		version:     version,
		client:      &http.Client{Timeout: 30 * time.Second},
		status:      Status{CurrentVersion: version},
	}
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.manifestURL, nil)
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

// IsNewer reports whether latest is a newer release than current. When both
// parse as semver-ish (v)MAJOR.MINOR.PATCH they compare numerically; otherwise
// any difference counts as newer so dev builds still surface releases.
func IsNewer(current, latest string) bool {
	current = strings.TrimSpace(current)
	latest = strings.TrimSpace(latest)
	if latest == "" || current == latest {
		return false
	}
	cv, cok := parseVersion(current)
	lv, lok := parseVersion(latest)
	if cok && lok {
		for i := range lv {
			if lv[i] != cv[i] {
				return lv[i] > cv[i]
			}
		}
		return false
	}
	return true
}

func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.Split(v, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return out, false
	}
	for i, part := range parts {
		if part == "" {
			return out, false
		}
		n := 0
		for _, r := range part {
			if r < '0' || r > '9' {
				return out, false
			}
			n = n*10 + int(r-'0')
		}
		out[i] = n
	}
	return out, true
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
