package history

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strconv"

	"OOF_RL/internal/config"
	"OOF_RL/internal/db"
	"OOF_RL/internal/events"
	"OOF_RL/internal/httputil"
	"OOF_RL/internal/plugin"
)

//go:embed view.html view.js
var viewFS embed.FS

const historySchema = `
DROP TABLE IF EXISTS ball_hit_events;
DROP TABLE IF EXISTS tick_snapshots;
DROP TABLE IF EXISTS goal_events;
DROP TABLE IF EXISTS player_match_stats;
DROP TABLE IF EXISTS matches;
DROP TABLE IF EXISTS players;

CREATE TABLE IF NOT EXISTS hist_players (
	primary_id TEXT PRIMARY KEY,
	name       TEXT NOT NULL,
	last_seen  DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS hist_matches (
	id              INTEGER PRIMARY KEY AUTOINCREMENT,
	match_guid      TEXT UNIQUE,
	arena           TEXT,
	started_at      DATETIME NOT NULL,
	ended_at        DATETIME,
	winner_team_num INTEGER,
	overtime        BOOLEAN DEFAULT 0,
	playlist_type   INTEGER
);

CREATE TABLE IF NOT EXISTS hist_player_match_stats (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	match_id    INTEGER NOT NULL REFERENCES hist_matches(id),
	primary_id  TEXT    NOT NULL REFERENCES hist_players(primary_id),
	team_num    INTEGER,
	score       INTEGER DEFAULT 0,
	goals       INTEGER DEFAULT 0,
	shots       INTEGER DEFAULT 0,
	assists     INTEGER DEFAULT 0,
	saves       INTEGER DEFAULT 0,
	touches     INTEGER DEFAULT 0,
	car_touches INTEGER DEFAULT 0,
	demos       INTEGER DEFAULT 0,
	UNIQUE(match_id, primary_id)
);

CREATE TABLE IF NOT EXISTS hist_goal_events (
	id                   INTEGER PRIMARY KEY AUTOINCREMENT,
	match_id             INTEGER NOT NULL REFERENCES hist_matches(id),
	scorer_id            TEXT REFERENCES hist_players(primary_id),
	scorer_name          TEXT NOT NULL DEFAULT '',
	assister_id          TEXT REFERENCES hist_players(primary_id),
	assister_name        TEXT NOT NULL DEFAULT '',
	ball_last_touch_id   TEXT REFERENCES hist_players(primary_id),
	goal_speed           REAL,
	goal_time            REAL,
	impact_x             REAL,
	impact_y             REAL,
	impact_z             REAL,
	scored_at            DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS hist_ball_hit_events (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	match_id       INTEGER NOT NULL REFERENCES hist_matches(id),
	player_id      TEXT REFERENCES hist_players(primary_id),
	pre_hit_speed  REAL,
	post_hit_speed REAL,
	loc_x          REAL,
	loc_y          REAL,
	loc_z          REAL,
	hit_at         DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS hist_tick_snapshots (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	match_id    INTEGER NOT NULL REFERENCES hist_matches(id),
	captured_at DATETIME NOT NULL,
	raw_json    TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_hist_pms_primary_id ON hist_player_match_stats(primary_id);
CREATE INDEX IF NOT EXISTS idx_hist_goal_scorer     ON hist_goal_events(scorer_id);
CREATE INDEX IF NOT EXISTS idx_hist_goal_match      ON hist_goal_events(match_id);
CREATE INDEX IF NOT EXISTS idx_hist_bh_player       ON hist_ball_hit_events(player_id);
CREATE INDEX IF NOT EXISTS idx_hist_tick_match      ON hist_tick_snapshots(match_id);
CREATE INDEX IF NOT EXISTS idx_hist_matches_started_at ON hist_matches(started_at);
`

type Plugin struct {
	cfg *config.Config
	db  *db.DB
}

func New(cfg *config.Config, database *db.DB) *Plugin {
	if err := database.RunMigration(historySchema); err != nil {
		log.Printf("[history] migrate: %v", err)
	}
	return &Plugin{cfg: cfg, db: database}
}

func (p *Plugin) ID() string         { return "history" }
func (p *Plugin) DBPrefix() string   { return "hist" }
func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) NavTab() plugin.NavTab {
	return plugin.NavTab{ID: "history", Label: "History", Order: 20}
}

func (p *Plugin) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/players", p.handlePlayers)
	mux.HandleFunc("/api/matches", p.handleMatches)
	mux.HandleFunc("/api/matches/", p.handleMatchDetail)
}

func (p *Plugin) SettingsSchema() []plugin.Setting {
	return []plugin.Setting{
		{
			Key:         "storage.ball_hit_events",
			Label:       "Ball hit events",
			Type:        plugin.SettingTypeCheckbox,
			Default:     "false",
			Description: "Every ball touch. Can generate large amounts of data.",
		},
		{
			Key:         "storage.tick_snapshots",
			Label:       "Tick snapshots",
			Type:        plugin.SettingTypeCheckbox,
			Default:     "false",
			Description: "Full game state at regular intervals. Produces very large data.",
		},
		{
			Key:         "storage.tick_snapshot_rate",
			Label:       "Tick rate",
			Type:        plugin.SettingTypeNumber,
			Default:     "1",
			Description: "Snapshots per second when tick snapshots are enabled.",
		},
		{
			Key:         "storage.raw_packets",
			Label:       "Capture raw packets (developer)",
			Type:        plugin.SettingTypeCheckbox,
			Default:     "false",
			Description: "Save raw UDP packets to disk under captures/ in the data directory.",
		},
	}
}

func (p *Plugin) ApplySettings(values map[string]string) error {
	if v, ok := values["storage.ball_hit_events"]; ok {
		p.cfg.Storage.BallHitEvents = v == "true"
	}
	if v, ok := values["storage.tick_snapshots"]; ok {
		p.cfg.Storage.TickSnapshots = v == "true"
	}
	if v, ok := values["storage.tick_snapshot_rate"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			p.cfg.Storage.TickSnapshotRate = f
		}
	}
	if v, ok := values["storage.raw_packets"]; ok {
		p.cfg.Storage.RawPackets = v == "true"
	}
	return nil
}

func (p *Plugin) HandleEvent(_ events.Envelope) {}
func (p *Plugin) Assets() fs.FS                 { return viewFS }

// -- handlers --

func (p *Plugin) handlePlayers(w http.ResponseWriter, r *http.Request) {
	players, err := p.db.AllPlayers()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	httputil.WriteJSON(w, players)
}

func (p *Plugin) handleMatches(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("player")
	matches, err := p.db.Matches(playerID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	teamGoals, _ := p.db.AllTeamGoals()
	playerCounts, _ := p.db.MatchPlayerCounts()

	type matchRow struct {
		db.Match
		Team0Goals  int `json:"team0_goals"`
		Team1Goals  int `json:"team1_goals"`
		PlayerCount int `json:"player_count"`
	}
	out := make([]matchRow, len(matches))
	for i, m := range matches {
		goals := teamGoals[m.ID]
		out[i] = matchRow{
			Match:       m,
			Team0Goals:  goals[0],
			Team1Goals:  goals[1],
			PlayerCount: playerCounts[m.ID],
		}
	}
	httputil.WriteJSON(w, out)
}

func (p *Plugin) handleMatchDetail(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/matches/"):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	players, err := p.db.MatchPlayers(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	goals, err := p.db.MatchGoals(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	httputil.WriteJSON(w, map[string]any{"players": players, "goals": goals})
}