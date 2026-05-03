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
	BallHitEvents    bool    `toml:"ball_hit_events"   json:"ball_hit_events"`
	TickSnapshots    bool    `toml:"tick_snapshots"    json:"tick_snapshots"`
	TickSnapshotRate float64 `toml:"tick_snapshot_rate" json:"tick_snapshot_rate"`
	RawPackets       bool    `toml:"raw_packets"       json:"raw_packets"`
}

type Config struct {
	AppPort                int             `toml:"app_port"                  json:"app_port"`
	DataDir                string          `toml:"data_dir"                  json:"data_dir"`
	RLInstallPath          string          `toml:"rl_install_path"           json:"rl_install_path"`
	TrackerCacheTTLMinutes int             `toml:"tracker_cache_ttl_minutes" json:"tracker_cache_ttl_minutes"`
	BallchasingAPIKey      string          `toml:"ballchasing_api_key"       json:"ballchasing_api_key"`
	OverlayHotkey          string          `toml:"overlay_hotkey"            json:"overlay_hotkey"`
	OverlayX               int             `toml:"overlay_x"                 json:"overlay_x"`
	OverlayY               int             `toml:"overlay_y"                 json:"overlay_y"`
	OverlayWidth           int             `toml:"overlay_width"             json:"overlay_width"`
	OverlayHeight          int             `toml:"overlay_height"            json:"overlay_height"`
	OverlayOpacity         float64         `toml:"overlay_opacity"           json:"overlay_opacity"`
	OverlayHoldMode        bool            `toml:"overlay_hold_mode"         json:"overlay_hold_mode"`
	Storage                StorageSettings `toml:"storage"                   json:"storage"`
	DisabledPlugins        []string        `toml:"disabled_plugins"          json:"disabled_plugins"`
}

// defaultDataDir returns %LOCALAPPDATA%\OOF_RL — the Windows standard for
// per-user, non-roaming application data.
func defaultDataDir() string {
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		return filepath.Join(local, "OOF_RL")
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "OOF_RL")
	}
	return "OOF_RL"
}

// ConfigPath returns the canonical location of the app's TOML config file.
func ConfigPath() string {
	return filepath.Join(defaultDataDir(), "config.toml")
}

// DBPath returns the SQLite database file path.
func (c *Config) DBPath() string { return filepath.Join(c.DataDir, "oof_rl.db") }

// LogPath returns the log file path.
func (c *Config) LogPath() string { return filepath.Join(c.DataDir, "oof_rl.log") }

// CapturesDir returns the directory for raw packet captures.
func (c *Config) CapturesDir() string { return filepath.Join(c.DataDir, "captures") }

func Defaults() Config {
	dataDir := defaultDataDir()
	return Config{
		AppPort:                8080,
		DataDir:                dataDir,
		RLInstallPath:          DetectRLPath(),
		TrackerCacheTTLMinutes: 5,
		OverlayHotkey:          "F9",
		OverlayX:               -1,
		OverlayY:               -1,
		OverlayWidth:           860,
		OverlayHeight:          620,
		OverlayOpacity:         1.0,
		OverlayHoldMode:        false,
		Storage: StorageSettings{
			BallHitEvents:    false,
			TickSnapshots:    false,
			TickSnapshotRate: 1.0,
			RawPackets:       false,
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Defaults()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if mkErr := os.MkdirAll(filepath.Dir(path), 0755); mkErr != nil {
			return cfg, mkErr
		}
		return cfg, Save(path, cfg)
	}
	_, err := toml.DecodeFile(path, &cfg)
	// Back-fill fields that didn't exist in older config files.
	if cfg.DataDir == "" {
		cfg.DataDir = defaultDataDir()
	}
	if cfg.OverlayOpacity == 0 {
		cfg.OverlayOpacity = 1.0
	}
	if cfg.OverlayWidth == 0 {
		cfg.OverlayWidth = 860
	}
	if cfg.OverlayHeight == 0 {
		cfg.OverlayHeight = 620
	}
	return cfg, err
}

func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}

// DetectRLPath returns the RL install directory by reading Launch.log.
func DetectRLPath() string {
	return rlPathFromLog()
}

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
			return filepath.Dir(filepath.Dir(base))
		}
	}
	return ""
}

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
			return filepath.Join(filepath.Dir(filepath.Dir(logPath)), "Config")
		}
	}
	return ""
}

type INISettings struct {
	PacketSendRate float64
	Port           int
}

func INIPath(rlInstallPath string) string {
	return filepath.Join(rlInstallPath, `TAGame\Config\DefaultStatsAPI.ini`)
}

func userStatsAPIPath() string {
	dir := detectUserConfigDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "DefaultStatsAPI.ini")
}

func ReadINI(rlInstallPath string) (INISettings, error) {
	s := INISettings{PacketSendRate: 0, Port: 49123}
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
		path = INIPath(rlInstallPath)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
	}
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