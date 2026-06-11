// Package update implements the manual update checker: fetch a release
// manifest, compare versions, and download the release zip with SHA256
// verification.
//
// This is host-core rather than a plugin because the end state of an updater
// (replace the running exe and restart) can never live inside the WASM
// sandbox. Milestone 1 is check + verified download only: nothing is
// installed, extracted, or restarted.
//
// Trust boundary: the SHA256 comes from the same unsigned manifest as the
// artifact URL, so verification proves transport integrity, not authorship.
// Signed manifests (Ed25519 detached signature, pinned public key) are a
// prerequisite for any auto-install milestone.
package update

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	CurrentVersion   string `json:"current_version"`
	LastCheckedAt    string `json:"last_checked_at,omitempty"`
	LatestVersion    string `json:"latest_version,omitempty"`
	NotesURL         string `json:"notes_url,omitempty"`
	UpdateAvailable  bool   `json:"update_available"`
	Downloading      bool   `json:"downloading"`
	BytesDownloaded  int64  `json:"bytes_downloaded,omitempty"`
	BytesTotal       int64  `json:"bytes_total,omitempty"`
	DownloadedPath   string `json:"downloaded_path,omitempty"`
	DownloadedSHA256 string `json:"downloaded_sha256,omitempty"`
	LastError        string `json:"last_error,omitempty"`
}

// Checker holds update state. All exported methods are safe for concurrent
// use; the download runs on its own goroutine and is observed via Status.
type Checker struct {
	manifestURL string
	version     string
	downloadDir string

	// client serves the small manifest fetch; a total timeout is fine there.
	// dlClient streams release zips of arbitrary size over arbitrary links, so
	// it bounds only connect/header latency, never the body read.
	client   *http.Client
	dlClient *http.Client

	mu       sync.Mutex
	status   Status
	manifest Manifest
}

// New creates a Checker for the given running version (config.AppVersion).
// Downloads land in downloadDir.
func New(version, downloadDir string) *Checker {
	return &Checker{
		manifestURL: DefaultManifestURL,
		version:     version,
		downloadDir: downloadDir,
		client:      &http.Client{Timeout: 30 * time.Second},
		dlClient: &http.Client{
			Transport: &http.Transport{
				ResponseHeaderTimeout: 30 * time.Second,
				Proxy:                 http.ProxyFromEnvironment,
			},
		},
		status: Status{CurrentVersion: version},
	}
}

// Status returns a snapshot of the current update state.
func (c *Checker) Status() Status {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
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
	c.manifest = manifest
	c.status = Status{
		CurrentVersion:  c.version,
		LastCheckedAt:   time.Now().UTC().Format(time.RFC3339),
		LatestVersion:   manifest.Version,
		NotesURL:        manifest.NotesURL,
		UpdateAvailable: IsNewer(c.version, manifest.Version),
		Downloading:     c.status.Downloading,
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

// StartDownload begins downloading the artifact from the last successful
// Check on a background goroutine. Progress and the verified result are
// observed via Status. A second call while a download is running is a no-op.
func (c *Checker) StartDownload() (Status, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.status.Downloading {
		return c.status, nil
	}
	if c.manifest.ArtifactURL == "" {
		return c.status, fmt.Errorf("no checked update manifest available")
	}
	if !c.status.UpdateAvailable {
		return c.status, fmt.Errorf("no update available")
	}
	manifest := c.manifest
	c.status.Downloading = true
	c.status.LastError = ""
	c.status.BytesDownloaded = 0
	c.status.BytesTotal = 0
	c.status.DownloadedPath = ""
	c.status.DownloadedSHA256 = ""
	go c.download(manifest)
	return c.status, nil
}

func (c *Checker) download(manifest Manifest) {
	finish := func(fn func(s *Status)) {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.status.Downloading = false
		fn(&c.status)
	}
	fail := func(msg string) {
		finish(func(s *Status) { s.LastError = msg })
	}

	name := SafeArtifactName(manifest.ArtifactName)
	if name == "" {
		name = "OOF_RL-" + strings.TrimPrefix(manifest.Version, "v") + ".zip"
	}
	if err := os.MkdirAll(c.downloadDir, 0755); err != nil {
		fail("create download dir: " + err.Error())
		return
	}
	dest := filepath.Join(c.downloadDir, name)

	req, err := http.NewRequest(http.MethodGet, manifest.ArtifactURL, nil)
	if err != nil {
		fail("bad artifact request: " + err.Error())
		return
	}
	req.Header.Set("Accept", "application/zip,application/octet-stream,*/*")
	resp, err := c.dlClient.Do(req)
	if err != nil {
		fail(err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fail(fmt.Sprintf("download HTTP %d", resp.StatusCode))
		return
	}
	c.mu.Lock()
	c.status.BytesTotal = resp.ContentLength
	c.mu.Unlock()

	tmpPath := dest + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		fail("create file: " + err.Error())
		return
	}
	hasher := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(f, hasher, c.progressWriter()), resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		fail("write file: " + copyErr.Error())
		return
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		fail("close file: " + closeErr.Error())
		return
	}

	got := fmt.Sprintf("%x", hasher.Sum(nil))
	want := NormalizeSHA256(manifest.ArtifactSHA256)
	if want == "" || got != want {
		_ = os.Remove(tmpPath)
		fail("download SHA256 mismatch")
		return
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		_ = os.Remove(tmpPath)
		fail("finalize file: " + err.Error())
		return
	}

	finish(func(s *Status) {
		s.DownloadedPath = dest
		s.DownloadedSHA256 = got
	})
}

// progressWriter counts streamed bytes into Status.BytesDownloaded so the
// frontend can poll progress.
func (c *Checker) progressWriter() io.Writer {
	return writerFunc(func(p []byte) (int, error) {
		c.mu.Lock()
		c.status.BytesDownloaded += int64(len(p))
		c.mu.Unlock()
		return len(p), nil
	})
}

type writerFunc func([]byte) (int, error)

func (w writerFunc) Write(p []byte) (int, error) { return w(p) }

// ParseManifest decodes and validates an update manifest.
func ParseManifest(data []byte) (Manifest, error) {
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
	switch {
	case m.Version == "":
		return m, fmt.Errorf("manifest version required")
	case m.ArtifactURL == "":
		return m, fmt.Errorf("manifest artifact_url required")
	case m.ArtifactName == "":
		return m, fmt.Errorf("manifest artifact_name required")
	case m.ArtifactSHA256 == "":
		return m, fmt.Errorf("manifest artifact_sha256 required")
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

// SafeArtifactName reduces a manifest-supplied file name to a bare base name
// so it cannot traverse outside the download directory.
func SafeArtifactName(name string) string {
	name = strings.TrimSpace(name)
	base := filepath.Base(filepath.FromSlash(strings.ReplaceAll(name, "\\", "/")))
	if base == "." || base == ".." || base == string(filepath.Separator) || base == "" {
		return ""
	}
	return base
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