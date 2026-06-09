package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	sdk "github.com/erosas/oof-plugin-sdk"
)

const (
	defaultManifestURL = "https://github.com/erosas/OOF_RL/releases/latest/download/update-manifest.json"
)

type updateManifest struct {
	Version        string `json:"version"`
	Channel        string `json:"channel"`
	NotesURL       string `json:"notes_url"`
	PublishedAt    string `json:"published_at"`
	ArtifactURL    string `json:"artifact_url"`
	ArtifactName   string `json:"artifact_name"`
	ArtifactSHA256 string `json:"artifact_sha256"`
}

type updateState struct {
	LastCheckedAt    string         `json:"last_checked_at,omitempty"`
	CurrentVersion   string         `json:"current_version"`
	LatestVersion    string         `json:"latest_version,omitempty"`
	ManifestURL      string         `json:"manifest_url"`
	UpdateAvailable  bool           `json:"update_available"`
	Downloaded       bool           `json:"downloaded"`
	DownloadedPath   string         `json:"downloaded_path,omitempty"`
	DownloadedSHA256 string         `json:"downloaded_sha256,omitempty"`
	LastError        string         `json:"last_error,omitempty"`
	Manifest         updateManifest `json:"manifest,omitempty"`
}

var (
	manifestURL    = defaultManifestURL
	currentVersion = "dev"
	statePath      = "/data/state.json"
	downloadDir    = "/data/downloads"
	nowUTC         = func() time.Time { return time.Now().UTC() }
	fetchHTTP      = sdk.HTTPFetch
	downloadHTTP   = sdk.HTTPDownload
)

func initPlugin() uint32 {
	if v := strings.TrimSpace(sdk.GetConfig("app_version")); v != "" {
		currentVersion = v
	}
	if st, err := readState(); err == nil && st.ManifestURL != "" {
		manifestURL = st.ManifestURL
	}
	return 0
}

func applySettings(data []byte) uint32 {
	var settings map[string]string
	if err := json.Unmarshal(data, &settings); err != nil {
		return 1
	}
	if v := strings.TrimSpace(settings["autoupdate_check_url"]); v != "" {
		manifestURL = v
	}
	return 0
}

func handleHTTP(req sdk.HTTPRequest) sdk.HTTPResponse {
	switch req.Path {
	case "/api/autoupdate/status":
		if req.Method != http.MethodGet {
			return sdk.JSONError(http.StatusMethodNotAllowed, "method not allowed")
		}
		return jsonStateResponse(statusState())
	case "/api/autoupdate/check":
		if req.Method != http.MethodPost {
			return sdk.JSONError(http.StatusMethodNotAllowed, "method not allowed")
		}
		st, err := checkForUpdates()
		if err != nil {
			return jsonStateError(st, err)
		}
		return jsonStateResponse(st)
	case "/api/autoupdate/download":
		if req.Method != http.MethodPost {
			return sdk.JSONError(http.StatusMethodNotAllowed, "method not allowed")
		}
		st, err := downloadUpdate()
		if err != nil {
			return jsonStateError(st, err)
		}
		return jsonStateResponse(st)
	default:
		return sdk.JSONError(http.StatusNotFound, "not found")
	}
}

func statusState() updateState {
	st, err := readState()
	if err != nil {
		return updateState{CurrentVersion: currentVersion, ManifestURL: manifestURL}
	}
	if st.CurrentVersion == "" {
		st.CurrentVersion = currentVersion
	}
	if st.ManifestURL == "" {
		st.ManifestURL = manifestURL
	}
	return st
}

func checkForUpdates() (updateState, error) {
	st := updateState{
		LastCheckedAt:  nowUTC().Format(time.RFC3339),
		CurrentVersion: currentVersion,
		ManifestURL:    manifestURL,
	}
	res := fetchHTTP(sdk.HTTPFetchRequest{
		Method:  http.MethodGet,
		URL:     manifestURL,
		Headers: map[string]string{"Accept": "application/json"},
	})
	if res.Error != "" {
		return writeErrorState(st, res.Error)
	}
	if res.Status < 200 || res.Status >= 300 {
		return writeErrorState(st, fmt.Sprintf("manifest HTTP %d", res.Status))
	}

	manifest, err := parseManifest([]byte(res.Body))
	if err != nil {
		return writeErrorState(st, err.Error())
	}
	st.Manifest = manifest
	st.LatestVersion = manifest.Version
	st.UpdateAvailable = isUpdateAvailable(currentVersion, manifest.Version)
	st.Downloaded = false
	st.DownloadedPath = ""
	st.DownloadedSHA256 = ""
	if err := writeState(st); err != nil {
		return st, err
	}
	return st, nil
}

func downloadUpdate() (updateState, error) {
	st := statusState()
	if st.Manifest.ArtifactURL == "" || st.Manifest.ArtifactSHA256 == "" {
		return writeErrorState(st, "no checked update manifest available")
	}
	if !st.UpdateAvailable {
		return writeErrorState(st, "no update available")
	}

	name := safeArtifactName(st.Manifest.ArtifactName)
	if name == "" {
		name = "OOF_RL-" + strings.TrimPrefix(st.Manifest.Version, "v") + ".zip"
	}
	dest := path.Join(downloadDir, name)
	res := downloadHTTP(sdk.HTTPDownloadRequest{
		URL:         st.Manifest.ArtifactURL,
		Destination: dest,
		Headers:     map[string]string{"Accept": "application/zip,application/octet-stream,*/*"},
	})
	if res.Error != "" {
		return writeErrorState(st, res.Error)
	}
	if res.Status < 200 || res.Status >= 300 {
		return writeErrorState(st, fmt.Sprintf("download HTTP %d", res.Status))
	}
	want := normalizeSHA256(st.Manifest.ArtifactSHA256)
	got := normalizeSHA256(res.SHA256)
	if want == "" || got == "" || want != got {
		_ = os.Remove(dest)
		return writeErrorState(st, "download SHA256 mismatch")
	}

	st.LastError = ""
	st.Downloaded = true
	st.DownloadedPath = res.Destination
	st.DownloadedSHA256 = got
	if err := writeState(st); err != nil {
		return st, err
	}
	return st, nil
}

func parseManifest(data []byte) (updateManifest, error) {
	var manifest updateManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("invalid manifest JSON: %w", err)
	}
	manifest.Version = strings.TrimSpace(manifest.Version)
	manifest.Channel = strings.TrimSpace(manifest.Channel)
	manifest.NotesURL = strings.TrimSpace(manifest.NotesURL)
	manifest.PublishedAt = strings.TrimSpace(manifest.PublishedAt)
	manifest.ArtifactURL = strings.TrimSpace(manifest.ArtifactURL)
	manifest.ArtifactName = strings.TrimSpace(manifest.ArtifactName)
	manifest.ArtifactSHA256 = normalizeSHA256(manifest.ArtifactSHA256)
	switch {
	case manifest.Version == "":
		return manifest, fmt.Errorf("manifest version required")
	case manifest.ArtifactURL == "":
		return manifest, fmt.Errorf("manifest artifact_url required")
	case manifest.ArtifactName == "":
		return manifest, fmt.Errorf("manifest artifact_name required")
	case manifest.ArtifactSHA256 == "":
		return manifest, fmt.Errorf("manifest artifact_sha256 required")
	}
	return manifest, nil
}

func writeErrorState(st updateState, msg string) (updateState, error) {
	st.LastError = msg
	_ = writeState(st)
	return st, fmt.Errorf("%s", msg)
}

func jsonStateResponse(st updateState) sdk.HTTPResponse {
	b, _ := json.Marshal(st)
	return sdk.JSONResponse(b)
}

func jsonStateError(st updateState, _ error) sdk.HTTPResponse {
	b, _ := json.Marshal(st)
	return sdk.HTTPResponse{
		Status:  http.StatusBadGateway,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    string(b),
	}
}

func readState() (updateState, error) {
	var st updateState
	b, err := os.ReadFile(statePath)
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return st, err
	}
	return st, nil
}

func writeState(st updateState) error {
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(statePath), 0755); err != nil {
		return err
	}
	return os.WriteFile(statePath, b, 0644)
}

func isUpdateAvailable(current, latest string) bool {
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
	return latest != current
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

func safeArtifactName(name string) string {
	name = strings.TrimSpace(name)
	base := path.Base(strings.ReplaceAll(name, "\\", "/"))
	if base == "." || base == "/" || base == "" {
		return ""
	}
	return base
}

func normalizeSHA256(value string) string {
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
