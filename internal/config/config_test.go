package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.AppPort != 8080 {
		t.Errorf("AppPort: got %d, want 8080", cfg.AppPort)
	}
	if cfg.DataDir == "" {
		t.Error("DataDir should not be empty")
	}
	if !strings.HasSuffix(cfg.DBPath(), "oof_rl.db") {
		t.Errorf("DBPath: got %q, want suffix oof_rl.db", cfg.DBPath())
	}
	if !strings.HasSuffix(cfg.CapturesDir(), "captures") {
		t.Errorf("CapturesDir: got %q, want suffix captures", cfg.CapturesDir())
	}
	s := cfg.Storage
	if s.BallHitEvents {
		t.Error("BallHitEvents should default to false")
	}
	if s.RawPackets {
		t.Error("RawPackets should default to false")
	}
}

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := Defaults()
	cfg.AppPort = 9090
	cfg.DataDir = filepath.Join(dir, "data")
	cfg.Storage.BallHitEvents = true
	cfg.Storage.RawPackets = true

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.AppPort != 9090 {
		t.Errorf("AppPort: got %d, want 9090", loaded.AppPort)
	}
	if loaded.DataDir != cfg.DataDir {
		t.Errorf("DataDir: got %q, want %q", loaded.DataDir, cfg.DataDir)
	}
	if !loaded.Storage.BallHitEvents {
		t.Error("BallHitEvents should be true after round-trip")
	}
	if !loaded.Storage.RawPackets {
		t.Error("RawPackets should be true after round-trip")
	}
}

func TestLoadCreatesFileWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load on missing file: %v", err)
	}
	if cfg.AppPort != 8080 {
		t.Errorf("AppPort: got %d, want 8080", cfg.AppPort)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Load should create config file when it does not exist")
	}
}

func TestStripINISection(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		section string
		want    string
	}{
		{
			name:    "removes target section",
			input:   "[IniVersion]\nVersion=3\n\n[TAGame.MatchStatsExporter_TA]\nPacketSendRate=60\nPort=49123\n",
			section: "TAGame.MatchStatsExporter_TA",
			want:    "[IniVersion]\nVersion=3\n",
		},
		{
			name:    "no-op when section absent",
			input:   "[IniVersion]\nVersion=3\n",
			section: "TAGame.MatchStatsExporter_TA",
			want:    "[IniVersion]\nVersion=3\n",
		},
		{
			name:    "only target section",
			input:   "[TAGame.MatchStatsExporter_TA]\nPacketSendRate=60\n",
			section: "TAGame.MatchStatsExporter_TA",
			want:    "",
		},
		{
			name:    "section followed by another section",
			input:   "[TAGame.MatchStatsExporter_TA]\nPort=49123\n\n[Other]\nFoo=bar\n",
			section: "TAGame.MatchStatsExporter_TA",
			want:    "[Other]\nFoo=bar\n",
		},
		{
			name:    "case-insensitive match",
			input:   "[tagame.matchstatsexporter_ta]\nPort=49123\n",
			section: "TAGame.MatchStatsExporter_TA",
			want:    "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := stripINISection(tc.input, tc.section)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestINIContent(t *testing.T) {
	s := INISettings{PacketSendRate: 60, Port: 49123}
	got := iniContent(s)
	if !strings.Contains(got, "[TAGame.MatchStatsExporter_TA]") {
		t.Error("iniContent should contain [TAGame.MatchStatsExporter_TA]")
	}
	if !strings.Contains(got, "PacketSendRate=60") {
		t.Error("iniContent should contain PacketSendRate=60")
	}
	if !strings.Contains(got, "Port=49123") {
		t.Error("iniContent should contain Port=49123")
	}
}

func TestReadWriteINI(t *testing.T) {
	t.Setenv("USERPROFILE", "")
	installDir := t.TempDir()
	settings := INISettings{PacketSendRate: 60, Port: 51234}
	if err := WriteINI(installDir, settings); err != nil {
		t.Fatalf("WriteINI: %v", err)
	}
	got, err := ReadINI(installDir)
	if err != nil {
		t.Fatalf("ReadINI: %v", err)
	}
	if got.PacketSendRate != 60 {
		t.Errorf("PacketSendRate: got %f, want 60", got.PacketSendRate)
	}
	if got.Port != 51234 {
		t.Errorf("Port: got %d, want 51234", got.Port)
	}
}

func TestWriteINIPreservesExistingContent(t *testing.T) {
	t.Setenv("USERPROFILE", "")
	installDir := t.TempDir()
	iniDir := filepath.Join(installDir, "TAGame", "Config")
	if err := os.MkdirAll(iniDir, 0755); err != nil {
		t.Fatal(err)
	}
	existing := "[IniVersion]\nVersion=3\n\n[TAGame.MatchStatsExporter_TA]\nPacketSendRate=30\nPort=12345\n"
	if err := os.WriteFile(filepath.Join(iniDir, "DefaultStatsAPI.ini"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}
	if err := WriteINI(installDir, INISettings{PacketSendRate: 60, Port: 49123}); err != nil {
		t.Fatalf("WriteINI: %v", err)
	}
	got, err := ReadINI(installDir)
	if err != nil {
		t.Fatalf("ReadINI: %v", err)
	}
	if got.PacketSendRate != 60 {
		t.Errorf("PacketSendRate: got %f, want 60", got.PacketSendRate)
	}
	if got.Port != 49123 {
		t.Errorf("Port: got %d, want 49123", got.Port)
	}
	data, _ := os.ReadFile(filepath.Join(iniDir, "DefaultStatsAPI.ini"))
	if !strings.Contains(string(data), "[IniVersion]") {
		t.Error("WriteINI should preserve existing [IniVersion] section")
	}
}

func TestReadINIFileNotFound(t *testing.T) {
	t.Setenv("USERPROFILE", "")
	_, err := ReadINI(t.TempDir())
	if err == nil {
		t.Error("expected error when INI file does not exist")
	}
}

func TestConfigLookup(t *testing.T) {
	cfg := Config{PluginSettings: map[string]string{
		"ballchasing_api_key":            "abc",
		"ballchasing_delete_after_upload": "true",
	}}
	if got := cfg.Lookup("ballchasing_api_key"); got != "abc" {
		t.Errorf("api_key: got %q, want %q", got, "abc")
	}
	if got := cfg.Lookup("ballchasing_delete_after_upload"); got != "true" {
		t.Errorf("delete_after_upload: got %q", got)
	}
	if got := cfg.Lookup("replay_dir"); got != DetectReplayDir() {
		t.Errorf("replay_dir: got %q, want %q", got, DetectReplayDir())
	}
	if got := cfg.Lookup("unknown_key"); got != "" {
		t.Errorf("unknown_key: got %q, want empty", got)
	}
}

func TestConfigSet(t *testing.T) {
	cfg := Config{}
	cfg.Set("ballchasing_api_key", "mykey")
	if got := cfg.PluginSettings["ballchasing_api_key"]; got != "mykey" {
		t.Errorf("api_key: got %q", got)
	}
	cfg.Set("ballchasing_delete_after_upload", "true")
	if got := cfg.PluginSettings["ballchasing_delete_after_upload"]; got != "true" {
		t.Errorf("delete_after_upload: got %q", got)
	}
	cfg.Set("unknown_key", "val")
	if got := cfg.PluginSettings["unknown_key"]; got != "val" {
		t.Errorf("unknown_key: got %q", got)
	}
}

func TestDetectReplayDir_NoEnv(t *testing.T) {
	t.Setenv("USERPROFILE", "")
	t.Setenv("OneDriveConsumer", "")
	t.Setenv("OneDrive", "")
	dir := DetectReplayDir()
	// With no env vars the function returns "" (no candidates to check).
	if dir != "" {
		t.Logf("DetectReplayDir returned %q (may be valid on this machine)", dir)
	}
}
