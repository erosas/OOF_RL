package update

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func manifestJSON(version, artifactURL, artifactSHA string) string {
	b, _ := json.Marshal(Manifest{
		Version:        version,
		Channel:        "stable",
		NotesURL:       "https://example.test/notes",
		PublishedAt:    "2026-06-06T12:00:00Z",
		ArtifactURL:    artifactURL,
		ArtifactName:   "OOF_RL.zip",
		ArtifactSHA256: artifactSHA,
	})
	return string(b)
}

func TestParseManifestValid(t *testing.T) {
	hash := fmt.Sprintf("%064x", 1)
	got, err := ParseManifest([]byte(manifestJSON("v1.2.3", "https://example.test/OOF_RL.zip", hash)))
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if got.Version != "v1.2.3" || got.ArtifactSHA256 != hash {
		t.Fatalf("manifest: got %+v", got)
	}
}

func TestParseManifestToleratesBOM(t *testing.T) {
	hash := fmt.Sprintf("%064x", 1)
	body := append([]byte{0xEF, 0xBB, 0xBF}, []byte(manifestJSON("v1.2.3", "https://example.test/OOF_RL.zip", hash))...)
	if _, err := ParseManifest(body); err != nil {
		t.Fatalf("ParseManifest with BOM: %v", err)
	}
}

func TestParseManifestRejectsMissingRequiredFields(t *testing.T) {
	for _, body := range []string{
		`{"version":"v1.2.3"}`,
		`{"artifact_url":"https://x/y.zip","artifact_name":"y.zip","artifact_sha256":"` + fmt.Sprintf("%064x", 1) + `"}`,
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
	}
	for _, c := range cases {
		if got := IsNewer(c.current, c.latest); got != c.want {
			t.Errorf("IsNewer(%q, %q): got %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}

func TestSafeArtifactName(t *testing.T) {
	if got := SafeArtifactName(`..\bad.zip`); got != "bad.zip" {
		t.Fatalf("got %q", got)
	}
	if got := SafeArtifactName("nested/OOF_RL.zip"); got != "OOF_RL.zip" {
		t.Fatalf("got %q", got)
	}
	if got := SafeArtifactName(".."); got != "" {
		t.Fatalf("got %q, want empty", got)
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

func TestCheckReportsUpdateAvailable(t *testing.T) {
	hash := fmt.Sprintf("%064x", 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, manifestJSON("v1.1.0", "https://example.test/OOF_RL.zip", hash))
	}))
	defer srv.Close()

	c := New("v1.0.0", t.TempDir())
	c.manifestURL = srv.URL

	st := c.Check(context.Background())
	if st.LastError != "" {
		t.Fatalf("Check: %s", st.LastError)
	}
	if !st.UpdateAvailable || st.LatestVersion != "v1.1.0" || st.NotesURL == "" {
		t.Fatalf("status: got %+v", st)
	}
}

func TestCheckReportsUpToDate(t *testing.T) {
	hash := fmt.Sprintf("%064x", 3)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, manifestJSON("v1.0.0", "https://example.test/OOF_RL.zip", hash))
	}))
	defer srv.Close()

	c := New("v1.0.0", t.TempDir())
	c.manifestURL = srv.URL

	st := c.Check(context.Background())
	if st.LastError != "" || st.UpdateAvailable {
		t.Fatalf("status: got %+v", st)
	}
}

func TestCheckRecordsFetchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := New("v1.0.0", t.TempDir())
	c.manifestURL = srv.URL

	st := c.Check(context.Background())
	if st.LastError == "" {
		t.Fatalf("expected error in status, got %+v", st)
	}
}

// waitDownload polls until the background download finishes.
func waitDownload(t *testing.T, c *Checker) Status {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		st := c.Status()
		if !st.Downloading {
			return st
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("download did not finish")
	return Status{}
}

func checkedChecker(t *testing.T, dir string, payload []byte, sha string) *Checker {
	t.Helper()
	artifact := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(payload)
	}))
	t.Cleanup(artifact.Close)
	manifest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, manifestJSON("v1.1.0", artifact.URL+"/OOF_RL.zip", sha))
	}))
	t.Cleanup(manifest.Close)

	c := New("v1.0.0", dir)
	c.manifestURL = manifest.URL
	if st := c.Check(context.Background()); st.LastError != "" {
		t.Fatalf("Check: %s", st.LastError)
	}
	return c
}

func TestDownloadVerifiesSHA(t *testing.T) {
	payload := []byte("release zip bytes")
	sha := fmt.Sprintf("%x", sha256.Sum256(payload))
	dir := t.TempDir()
	c := checkedChecker(t, dir, payload, sha)

	if _, err := c.StartDownload(); err != nil {
		t.Fatalf("StartDownload: %v", err)
	}
	st := waitDownload(t, c)
	if st.LastError != "" || st.DownloadedSHA256 != sha {
		t.Fatalf("status: got %+v", st)
	}
	saved, err := os.ReadFile(filepath.Join(dir, "OOF_RL.zip"))
	if err != nil {
		t.Fatalf("read download: %v", err)
	}
	if string(saved) != string(payload) {
		t.Fatalf("payload mismatch: got %q", saved)
	}
	if st.BytesDownloaded != int64(len(payload)) {
		t.Errorf("bytes downloaded: got %d, want %d", st.BytesDownloaded, len(payload))
	}
}

func TestDownloadRejectsSHAMismatch(t *testing.T) {
	payload := []byte("release zip bytes")
	dir := t.TempDir()
	c := checkedChecker(t, dir, payload, fmt.Sprintf("%064x", 9))

	if _, err := c.StartDownload(); err != nil {
		t.Fatalf("StartDownload: %v", err)
	}
	st := waitDownload(t, c)
	if st.LastError == "" || st.DownloadedPath != "" {
		t.Fatalf("expected SHA mismatch, got %+v", st)
	}
	if _, err := os.Stat(filepath.Join(dir, "OOF_RL.zip")); !os.IsNotExist(err) {
		t.Fatal("rejected artifact must not be kept")
	}
	if _, err := os.Stat(filepath.Join(dir, "OOF_RL.zip.tmp")); !os.IsNotExist(err) {
		t.Fatal("tmp file must be cleaned up")
	}
}

func TestStartDownloadWithoutCheck(t *testing.T) {
	c := New("v1.0.0", t.TempDir())
	if _, err := c.StartDownload(); err == nil {
		t.Fatal("expected error without a checked manifest")
	}
}

func TestStartDownloadWhenUpToDate(t *testing.T) {
	hash := fmt.Sprintf("%064x", 3)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, manifestJSON("v1.0.0", "https://example.test/OOF_RL.zip", hash))
	}))
	defer srv.Close()
	c := New("v1.0.0", t.TempDir())
	c.manifestURL = srv.URL
	c.Check(context.Background())

	if _, err := c.StartDownload(); err == nil {
		t.Fatal("expected error when no update is available")
	}
}

func TestHandlers(t *testing.T) {
	payload := []byte("release zip bytes")
	sha := fmt.Sprintf("%x", sha256.Sum256(payload))
	c := checkedChecker(t, t.TempDir(), payload, sha)

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
	c.HandleDownload(w, httptest.NewRequest(http.MethodPost, "/api/update/download", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("download: got %d — %s", w.Code, w.Body.String())
	}
	waitDownload(t, c)
}