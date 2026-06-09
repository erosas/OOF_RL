package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	sdk "github.com/erosas/oof-plugin-sdk"
)

func resetTestHooks(t *testing.T) {
	t.Helper()
	oldManifestURL := manifestURL
	oldCurrent := currentVersion
	oldStatePath := statePath
	oldDownloadDir := downloadDir
	oldNow := nowUTC
	oldFetch := fetchHTTP
	oldDownload := downloadHTTP
	dir := t.TempDir()
	manifestURL = defaultManifestURL
	currentVersion = "v1.0.0"
	statePath = filepath.Join(dir, "state.json")
	downloadDir = "/data/downloads"
	nowUTC = func() time.Time { return time.Date(2026, 6, 6, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() {
		manifestURL = oldManifestURL
		currentVersion = oldCurrent
		statePath = oldStatePath
		downloadDir = oldDownloadDir
		nowUTC = oldNow
		fetchHTTP = oldFetch
		downloadHTTP = oldDownload
	})
}

func manifestJSON(version, artifactSHA string) string {
	b, _ := json.Marshal(updateManifest{
		Version:        version,
		Channel:        "stable",
		NotesURL:       "https://example.test/notes",
		PublishedAt:    "2026-06-06T12:00:00Z",
		ArtifactURL:    "https://example.test/OOF_RL.zip",
		ArtifactName:   "OOF_RL.zip",
		ArtifactSHA256: artifactSHA,
	})
	return string(b)
}

func TestParseManifestValid(t *testing.T) {
	hash := fmt.Sprintf("%064x", 1)
	got, err := parseManifest([]byte(manifestJSON("v1.2.3", hash)))
	if err != nil {
		t.Fatalf("parseManifest: %v", err)
	}
	if got.Version != "v1.2.3" || got.ArtifactSHA256 != hash {
		t.Fatalf("manifest: got %+v", got)
	}
}

func TestParseManifestRejectsMissingRequiredFields(t *testing.T) {
	_, err := parseManifest([]byte(`{"version":"v1.2.3"}`))
	if err == nil {
		t.Fatal("expected missing artifact fields error")
	}
}

func TestCheckForUpdatesReportsAvailableAndWritesState(t *testing.T) {
	resetTestHooks(t)
	hash := fmt.Sprintf("%064x", 2)
	fetchHTTP = func(req sdk.HTTPFetchRequest) sdk.HTTPFetchResult {
		if req.URL != defaultManifestURL {
			t.Fatalf("manifest URL: got %q", req.URL)
		}
		return sdk.HTTPFetchResult{Status: http.StatusOK, Body: manifestJSON("v1.1.0", hash)}
	}

	st, err := checkForUpdates()
	if err != nil {
		t.Fatalf("checkForUpdates: %v", err)
	}
	if !st.UpdateAvailable || st.LatestVersion != "v1.1.0" {
		t.Fatalf("state: got %+v", st)
	}
	saved, err := readState()
	if err != nil {
		t.Fatalf("readState: %v", err)
	}
	if !saved.UpdateAvailable || saved.Manifest.ArtifactSHA256 != hash {
		t.Fatalf("saved state: got %+v", saved)
	}
}

func TestCheckForUpdatesReportsUpToDate(t *testing.T) {
	resetTestHooks(t)
	hash := fmt.Sprintf("%064x", 3)
	fetchHTTP = func(req sdk.HTTPFetchRequest) sdk.HTTPFetchResult {
		return sdk.HTTPFetchResult{Status: http.StatusOK, Body: manifestJSON("v1.0.0", hash)}
	}
	st, err := checkForUpdates()
	if err != nil {
		t.Fatalf("checkForUpdates: %v", err)
	}
	if st.UpdateAvailable {
		t.Fatalf("expected no update, got %+v", st)
	}
}

func TestStateRoundTrip(t *testing.T) {
	resetTestHooks(t)
	want := updateState{
		CurrentVersion: "v1.0.0",
		LatestVersion:  "v1.2.0",
		ManifestURL:    defaultManifestURL,
	}
	if err := writeState(want); err != nil {
		t.Fatalf("writeState: %v", err)
	}
	got, err := readState()
	if err != nil {
		t.Fatalf("readState: %v", err)
	}
	if got.LatestVersion != want.LatestVersion {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestDownloadUpdatePassesSHA(t *testing.T) {
	resetTestHooks(t)
	payload := []byte("release zip")
	hash := fmt.Sprintf("%x", sha256.Sum256(payload))
	if err := writeState(updateState{
		CurrentVersion:  "v1.0.0",
		LatestVersion:   "v1.1.0",
		ManifestURL:     defaultManifestURL,
		UpdateAvailable: true,
		Manifest: updateManifest{
			Version:        "v1.1.0",
			ArtifactURL:    "https://example.test/OOF_RL.zip",
			ArtifactName:   "OOF_RL.zip",
			ArtifactSHA256: hash,
		},
	}); err != nil {
		t.Fatalf("writeState: %v", err)
	}
	downloadHTTP = func(req sdk.HTTPDownloadRequest) sdk.HTTPDownloadResult {
		if req.Destination != "/data/downloads/OOF_RL.zip" {
			t.Fatalf("destination: got %q", req.Destination)
		}
		return sdk.HTTPDownloadResult{
			Status:      http.StatusOK,
			Destination: req.Destination,
			Bytes:       int64(len(payload)),
			SHA256:      hash,
		}
	}

	st, err := downloadUpdate()
	if err != nil {
		t.Fatalf("downloadUpdate: %v", err)
	}
	if !st.Downloaded || st.DownloadedSHA256 != hash {
		t.Fatalf("state: got %+v", st)
	}
}

func TestDownloadUpdateRejectsSHAMismatch(t *testing.T) {
	resetTestHooks(t)
	hash := fmt.Sprintf("%064x", 4)
	if err := writeState(updateState{
		CurrentVersion:  "v1.0.0",
		LatestVersion:   "v1.1.0",
		ManifestURL:     defaultManifestURL,
		UpdateAvailable: true,
		Manifest: updateManifest{
			Version:        "v1.1.0",
			ArtifactURL:    "https://example.test/OOF_RL.zip",
			ArtifactName:   "OOF_RL.zip",
			ArtifactSHA256: hash,
		},
	}); err != nil {
		t.Fatalf("writeState: %v", err)
	}
	downloadHTTP = func(req sdk.HTTPDownloadRequest) sdk.HTTPDownloadResult {
		return sdk.HTTPDownloadResult{Status: http.StatusOK, Destination: req.Destination, SHA256: fmt.Sprintf("%064x", 5)}
	}

	st, err := downloadUpdate()
	if err == nil {
		t.Fatal("expected SHA mismatch error")
	}
	if st.LastError == "" {
		t.Fatalf("expected last error in state: %+v", st)
	}
}

func TestCheckForUpdatesHandlesFetchError(t *testing.T) {
	resetTestHooks(t)
	fetchHTTP = func(req sdk.HTTPFetchRequest) sdk.HTTPFetchResult {
		return sdk.HTTPFetchResult{Error: "network down"}
	}
	st, err := checkForUpdates()
	if err == nil {
		t.Fatal("expected fetch error")
	}
	if st.LastError != "network down" {
		t.Fatalf("state: got %+v", st)
	}
}

func TestSafeArtifactName(t *testing.T) {
	if got := safeArtifactName(`..\bad.zip`); got != "bad.zip" {
		t.Fatalf("got %q", got)
	}
	if got := safeArtifactName(filepath.Join("nested", "OOF_RL.zip")); got != "OOF_RL.zip" {
		t.Fatalf("got %q", got)
	}
}
