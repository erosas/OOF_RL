package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

type StorageSettings struct {
	MatchMetadata    bool    `toml:"match_metadata"    json:"match_metadata"`
	PlayerMatchStats bool    `toml:"player_match_stats" json:"player_match_stats"`
	GoalEvents       bool    `toml:"goal_events"       json:"goal_events"`
	BallHitEvents    bool    `toml:"ball_hit_events"   json:"ball_hit_events"`
	TickSnapshots    bool    `toml:"tick_snapshots"    json:"tick_snapshots"`
	TickSnapshotRate float64 `toml:"tick_snapshot_rate" json:"tick_snapshot_rate"`
	OtherEvents      bool    `toml:"other_events"      json:"other_events"`
	RawPackets       bool    `toml:"raw_packets"       json:"raw_packets"`
	RawPacketsDir    string  `toml:"raw_packets_dir"   json:"raw_packets_dir"`
}

type Config struct {
	AppPort                int             `toml:"app_port"                  json:"app_port"`
	RLInstallPath          string          `toml:"rl_install_path"           json:"rl_install_path"`
	DBPath                 string          `toml:"db_path"                   json:"db_path"`
	OpenInBrowser          bool            `toml:"open_in_browser"           json:"open_in_browser"`
	TrackerCacheTTLMinutes int             `toml:"tracker_cache_ttl_minutes" json:"tracker_cache_ttl_minutes"`
	BallchasingAPIKey      string          `toml:"ballchasing_api_key"       json:"ballchasing_api_key"`
	OverlayHotkey          string          `toml:"overlay_hotkey"            json:"overlay_hotkey"`
	Storage                StorageSettings `toml:"storage"                   json:"storage"`
}

func Defaults() Config {
	return Config{
		AppPort:                8080,
		RLInstallPath:          DetectRLPath(),
		DBPath:                 "oof_rl.db",
		TrackerCacheTTLMinutes: 60,
		OverlayHotkey:          "F9",
		Storage: StorageSettings{
			MatchMetadata:    true,
			PlayerMatchStats: true,
			GoalEvents:       true,
			BallHitEvents:    false,
			TickSnapshots:    false,
			TickSnapshotRate: 1.0,
			OtherEvents:      true,
			RawPackets:       false,
			RawPacketsDir:    "captures",
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Defaults()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, Save(path, cfg)
	}
	_, err := toml.DecodeFile(path, &cfg)
	return cfg, err
}

func Save(path string, cfg Config) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

// DetectRLPath returns the RL install directory by reading Launch.log.
// Both Steam and Epic share the same user data directory and the same log,
// so this works regardless of launcher. The log reflects whichever install
// was run most recently.
func DetectRLPath() string {
	return rlPathFromLog()
}

// rlPathFromLog extracts the install directory from Launch.log.
// The log contains a line like:
//
//	Init: Base directory: C:\...\rocketleague\Binaries\Win64\
//
// Walking up two levels gives the install root.
func rlPathFromLog() string {
	dir := detectUserConfigDir()
	if dir == "" {
		return ""
	}
	logPath := filepath.Join(filepath.Dir(dir), "Logs", "Launch.log")
	f, err := os.Open(logPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		const prefix = "Init: Base directory: "
		if strings.HasPrefix(line, prefix) {
			base := strings.TrimPrefix(line, prefix)
			base = filepath.Clean(strings.TrimRight(base, `\/`))
			// base is Binaries\Win64 — go up two levels
			return filepath.Dir(filepath.Dir(base))
		}
	}

	return ""
}

// detectUserConfigDir locates the RL user config directory by finding
// Launch.log, which RL writes on every launch. Its presence is the most
// reliable signal that RL has run and that its sibling Config dir is valid.
// This handles OneDrive, custom Documents locations, and non-standard setups.
func detectUserConfigDir() string {
	home := os.Getenv("USERPROFILE")
	if home == "" {
		return ""
	}
	logCandidates := []string{
		filepath.Join(home, `OneDrive\Documents\My Games\Rocket League\TAGame\Logs\Launch.log`),
		filepath.Join(home, `Documents\My Games\Rocket League\TAGame\Logs\Launch.log`),
	}
	for _, logPath := range logCandidates {
		if _, err := os.Stat(logPath); err == nil {
			// Logs/ and Config/ are siblings under TAGame/
			return filepath.Join(filepath.Dir(filepath.Dir(logPath)), "Config")
		}
	}
	return ""
}

// INISettings holds the parsed Stats API config values.
type INISettings struct {
	PacketSendRate float64
	Port           int
}

// INIPath returns the install-directory DefaultStatsAPI.ini path.
// RL ignores this file in favour of the one in the user config dir; this is
// only used as a last-resort fallback when the user config dir cannot be found.
func INIPath(rlInstallPath string) string {
	return filepath.Join(rlInstallPath, `TAGame\Config\DefaultStatsAPI.ini`)
}

// userStatsAPIPath returns the path to the DefaultStatsAPI.ini in the user
// config dir, which RL reads ahead of the install-dir default.
func userStatsAPIPath() string {
	dir := detectUserConfigDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "DefaultStatsAPI.ini")
}

func ReadINI(rlInstallPath string) (INISettings, error) {
	s := INISettings{PacketSendRate: 0, Port: 49123}

	// Prefer the user runtime config; fall back to the install-dir default.
	path := userStatsAPIPath()
	if path == "" {
		path = INIPath(rlInstallPath)
	}

	f, err := os.Open(path)
	if err != nil {
		return s, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "PacketSendRate=") {
			val := strings.TrimPrefix(line, "PacketSendRate=")
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				s.PacketSendRate = v
			}
		}
		if strings.HasPrefix(line, "Port=") {
			val := strings.TrimPrefix(line, "Port=")
			if v, err := strconv.Atoi(val); err == nil {
				s.Port = v
			}
		}
	}
	return s, scanner.Err()
}

func WriteINI(rlInstallPath string, s INISettings) error {
	path := userStatsAPIPath()
	if path == "" {
		// Launch.log not found — RL has likely never been run on this machine.
		// Fall back to the install dir (note: current RL versions ignore this file,
		// so the user will need to launch RL once before the config takes effect).
		path = INIPath(rlInstallPath)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
	}

	// Read existing file, strip any existing [TAGame.MatchStatsExporter_TA]
	// section, then re-append the updated one. Preserves [IniVersion] etc.
	existing := ""
	if data, err := os.ReadFile(path); err == nil {
		existing = stripINISection(string(data), "TAGame.MatchStatsExporter_TA")
	}
	existing = strings.TrimRight(existing, "\r\n")
	return os.WriteFile(path, []byte(existing+"\n\n"+iniContent(s)), 0644)
}

func iniContent(s INISettings) string {
	return fmt.Sprintf("[TAGame.MatchStatsExporter_TA]\nPacketSendRate=%.0f\nPort=%d\n", s.PacketSendRate, s.Port)
}

// stripINISection removes a named section and its keys from INI content.
func stripINISection(content, section string) string {
	var out []string
	skip := false
	header := "[" + section + "]"
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			skip = strings.EqualFold(trimmed, header)
		}
		if !skip {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
