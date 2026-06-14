package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func manifestJSON(version, notesURL, artifactURL string) string {
	b, _ := json.Marshal(Manifest{
		Version:        version,
		Channel:        "stable",
		NotesURL:       notesURL,
		PublishedAt:    "2026-06-06T12:00:00Z",
		ArtifactURL:    artifactURL,
		ArtifactName:   "OOF_RL.zip",
		ArtifactSHA256: fmt.Sprintf("%064x", 1),
	})
	return string(b)
}

const (
	goodNotesURL    = "https://github.com/erosas/OOF_RL/releases/tag/v1.1.0"
	goodArtifactURL = "https://github.com/erosas/OOF_RL/releases/download/v1.1.0/OOF_RL-v1.1.0.zip"
)

func TestParseManifestValid(t *testing.T) {
	got, err := ParseManifest([]byte(manifestJSON("v1.2.3", goodNotesURL, goodArtifactURL)))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if got.Version != "v1.2.3" || got.ArtifactURL != goodArtifactURL {
		t.Fatalf("manifest: got %+v", got)
	}
}

func TestParseManifestToleratesBOM(t *testing.T) {
	body := append([]byte{0xEF, 0xBB, 0xBF}, []byte(manifestJSON("v1.2.3", goodNotesURL, goodArtifactURL))...)
	if _, err := ParseManifest(body); err != nil {
		t.Fatalf("ParseManifest with BOM: %v", err)
	}
}

func TestParseManifestRequiresOnlyVersion(t *testing.T) {
	// Artifact fields are optional now: the checker never fetches artifacts.
	if _, err := ParseManifest([]byte(`{"version":"v1.2.3"}`)); err != nil {
		t.Errorf("version-only manifest: %v", err)
	}
	for _, body := range []string{
		`{"artifact_url":"https://x/y.zip","artifact_name":"y.zip"}`,
		`{"version":"  "}`,
		`not json`,
	} {
		if _, err := ParseManifest([]byte(body)); err == nil {
			t.Errorf("body %q: expected error", body)
		}
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"v1.0.0", "v1.1.0", true},
		{"v1.1.0", "v1.0.0", false},
		{"v1.0.0", "v1.0.0", false},
		{"v1.9.0", "v1.10.0", true},
		{"1.0.0", "v1.0.1", true},
		{"dev", "v1.0.0", true}, // non-semver current: any release surfaces
		{"v1.0.0", "", false},
		// Prerelease (dev channel) ordering.
		{"v1.2.3-dev.1", "v1.2.3-dev.2", true},   // later dev build is newer
		{"v1.2.3-dev.2", "v1.2.3-dev.1", false},  // no phantom downgrade
		{"v1.2.3-dev.2", "v1.2.3-dev.10", true},  // numeric, not lexical
		{"v1.2.3-dev.1", "v1.2.3-dev.1", false},  // identical dev build
		{"v1.2.3-dev.1", "v1.2.3", true},         // stable supersedes its prereleases
		{"v1.2.3", "v1.2.3-dev.1", false},        // prerelease never beats its release
		{"v1.2.3-dev.5", "v1.2.4", true},         // a higher core wins regardless of pre
		{"v1.2.4", "v1.2.3-dev.9", false},        // ...and a lower core never surfaces
		{"v1.2", "v1.2.0", true},                 // short core isn't semver: falls back to any-difference
		{"v1.2.3-dev.2", "v1.2.3-dev.02", true},  // leading-zero id is alphanumeric, outranks numeric 2
	}
	for _, c := range cases {
		if got := IsNewer(c.current, c.latest); got != c.want {
			t.Errorf("IsNewer(%q, %q): got %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}

func TestSafeReleaseURL(t *testing.T) {
	for _, ok := range []string{goodNotesURL, goodArtifactURL} {
		if got := SafeReleaseURL(ok); got != ok {
			t.Errorf("%q: got %q, want unchanged", ok, got)
		}
	}
	for _, bad := range []string{
		"",
		"https://evil.test/OOF_RL.zip",
		"http://github.com/erosas/OOF_RL/releases", // not HTTPS
		"https://github.com/evil/repo/releases",
		"https://github.com/erosas/OOF_RL/../../evil/repo",
		"https://github.com/erosas/OOF_RL/%2e%2e/%2e%2e/evil/repo", // encoded traversal
		"https://github.com/erosas/OOF_RL/%5c..%5cevil",           // encoded backslash
		"https://github.com/erosas/OOF_RLx/releases",              // prefix must end at the repo
		"https://user@github.com/erosas/OOF_RL/releases",          // userinfo
		"https://github.com:8443/erosas/OOF_RL/releases",          // explicit port
		"https://github.com.evil.test/erosas/OOF_RL/releases",     // host suffix trick
		"javascript:alert(1)",
	} {
		if got := SafeReleaseURL(bad); got != "" {
			t.Errorf("%q: got %q, want empty", bad, got)
		}
	}
}

func TestNormalizeSHA256(t *testing.T) {
	upper := fmt.Sprintf("%064X", 7)
	if got := NormalizeSHA256(upper); got != fmt.Sprintf("%064x", 7) {
		t.Fatalf("got %q", got)
	}
	for _, bad := range []string{"", "abc", fmt.Sprintf("%063x", 1) + "g"} {
		if got := NormalizeSHA256(bad); got != "" {
			t.Errorf("%q: got %q, want empty", bad, got)
		}
	}
}

func checkerForManifest(t *testing.T, body string) *Checker {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	c := New("v1.0.0", nil)
	c.stableURL = srv.URL
	c.devURL = srv.URL
	return c
}

func TestCheckSelectsChannelByDevMode(t *testing.T) {
	var mu sync.Mutex
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPath = r.URL.Path
		mu.Unlock()
		fmt.Fprint(w, manifestJSON("v1.1.0", goodNotesURL, goodArtifactURL))
	}))
	t.Cleanup(srv.Close)

	dev := false
	c := New("v1.0.0", func() bool { return dev })
	c.stableURL = srv.URL + "/stable"
	c.devURL = srv.URL + "/dev"

	c.Check(context.Background())
	mu.Lock()
	stablePath := gotPath
	mu.Unlock()
	if stablePath != "/stable" {
		t.Fatalf("dev mode off: requested %q, want /stable", stablePath)
	}

	dev = true // toggled live, mid-session
	c.Check(context.Background())
	mu.Lock()
	devPath := gotPath
	mu.Unlock()
	if devPath != "/dev" {
		t.Fatalf("dev mode on: requested %q, want /dev", devPath)
	}
}

func TestCheckReportsUpdateAvailable(t *testing.T) {
	c := checkerForManifest(t, manifestJSON("v1.1.0", goodNotesURL, goodArtifactURL))
	st := c.Check(context.Background())
	if st.LastError != "" {
		t.Fatalf("Check: %s", st.LastError)
	}
	if !st.UpdateAvailable || st.LatestVersion != "v1.1.0" {
		t.Fatalf("status: got %+v", st)
	}
	if st.NotesURL != goodNotesURL || st.DownloadURL != goodArtifactURL {
		t.Fatalf("links: got %+v", st)
	}
}

func TestCheckFiltersUntrustedLinks(t *testing.T) {
	c := checkerForManifest(t, manifestJSON("v1.1.0", "https://evil.test/notes", "https://evil.test/OOF_RL.zip"))
	st := c.Check(context.Background())
	if st.LastError != "" || !st.UpdateAvailable {
		t.Fatalf("status: got %+v", st)
	}
	if st.NotesURL != "" || st.DownloadURL != "" {
		t.Fatalf("untrusted links must not surface: got %+v", st)
	}
}

func TestCheckReportsUpToDate(t *testing.T) {
	c := checkerForManifest(t, manifestJSON("v1.0.0", goodNotesURL, goodArtifactURL))
	st := c.Check(context.Background())
	if st.LastError != "" || st.UpdateAvailable {
		t.Fatalf("status: got %+v", st)
	}
}

func TestRunPeriodic(t *testing.T) {
	old := startupCheckDelay
	startupCheckDelay = time.Millisecond
	defer func() { startupCheckDelay = old }()

	c := checkerForManifest(t, manifestJSON("v1.1.0", goodNotesURL, goodArtifactURL))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		c.RunPeriodic(ctx, 5*time.Millisecond, 5*time.Millisecond)
		close(done)
	}()

	deadline := time.Now().Add(5 * time.Second)
	for c.Status().LastCheckedAt == "" {
		if time.Now().After(deadline) {
			t.Fatal("periodic check never ran")
		}
		time.Sleep(time.Millisecond)
	}
	if st := c.Status(); !st.UpdateAvailable {
		t.Fatalf("status: got %+v", st)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("RunPeriodic did not stop on context cancel")
	}
}

func TestRunPeriodicUsesDevIntervalInDevMode(t *testing.T) {
	old := startupCheckDelay
	startupCheckDelay = time.Millisecond
	defer func() { startupCheckDelay = old }()

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		fmt.Fprint(w, manifestJSON("v1.1.0", goodNotesURL, goodArtifactURL))
	}))
	t.Cleanup(srv.Close)

	c := New("v1.0.0", func() bool { return true })
	c.stableURL = srv.URL
	c.devURL = srv.URL

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go c.RunPeriodic(ctx, time.Hour, 2*time.Millisecond)

	// In dev mode the 2ms dev interval fires repeatedly; the 1h stable interval
	// would have produced only the single post-startup check.
	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&hits) < 3 {
		if time.Now().After(deadline) {
			t.Fatalf("dev cadence: got %d checks, want the dev interval to fire repeatedly", atomic.LoadInt32(&hits))
		}
		time.Sleep(time.Millisecond)
	}
}

func TestCheckRecordsFetchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := New("v1.0.0", nil)
	c.stableURL = srv.URL
	c.devURL = srv.URL

	st := c.Check(context.Background())
	if st.LastError == "" {
		t.Fatalf("expected error in status, got %+v", st)
	}
}

func TestHandlers(t *testing.T) {
	c := checkerForManifest(t, manifestJSON("v1.1.0", goodNotesURL, goodArtifactURL))

	w := httptest.NewRecorder()
	c.HandleStatus(w, httptest.NewRequest(http.MethodGet, "/api/update/status", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}

	w = httptest.NewRecorder()
	c.HandleCheck(w, httptest.NewRequest(http.MethodGet, "/api/update/check", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("check GET: got %d, want 405", w.Code)
	}

	w = httptest.NewRecorder()
	c.HandleCheck(w, httptest.NewRequest(http.MethodPost, "/api/update/check", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("check POST: got %d — %s", w.Code, w.Body.String())
	}
	var st Status
	if err := json.Unmarshal(w.Body.Bytes(), &st); err != nil {
		t.Fatalf("check body: %v", err)
	}
	if !st.UpdateAvailable || st.DownloadURL != goodArtifactURL {
		t.Fatalf("check status: got %+v", st)
	}
}
