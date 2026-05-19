package histstore

import "OOF_RL/internal/db"

const schema = `
DROP TABLE IF EXISTS ball_hit_events;
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
	incomplete      BOOLEAN DEFAULT 0,
	forfeit         BOOLEAN DEFAULT 0,
	playlist_type   INTEGER,
	team_score_0    INTEGER,
	team_score_1    INTEGER
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

CREATE TABLE IF NOT EXISTS hist_statfeed_events (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	match_id     INTEGER NOT NULL REFERENCES hist_matches(id),
	player_id    TEXT NOT NULL DEFAULT '',
	player_name  TEXT NOT NULL DEFAULT '',
	team_num     INTEGER NOT NULL DEFAULT 0,
	event_type   TEXT NOT NULL,
	target_id    TEXT NOT NULL DEFAULT '',
	target_name  TEXT NOT NULL DEFAULT '',
	occurred_at  DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_hist_pms_primary_id ON hist_player_match_stats(primary_id);
CREATE INDEX IF NOT EXISTS idx_hist_goal_scorer     ON hist_goal_events(scorer_id);
CREATE INDEX IF NOT EXISTS idx_hist_goal_match      ON hist_goal_events(match_id);
CREATE INDEX IF NOT EXISTS idx_hist_bh_player       ON hist_ball_hit_events(player_id);
CREATE INDEX IF NOT EXISTS idx_hist_matches_started_at ON hist_matches(started_at);
CREATE INDEX IF NOT EXISTS idx_hist_sf_match  ON hist_statfeed_events(match_id);
CREATE INDEX IF NOT EXISTS idx_hist_sf_player ON hist_statfeed_events(player_id);
`

// Migrate runs the schema and any additive column migrations against the DB.
func Migrate(database *db.DB) error {
	if err := database.RunMigration(schema); err != nil {
		return err
	}
	for _, col := range [][2]string{
		{"team_score_0", "INTEGER"},
		{"team_score_1", "INTEGER"},
		{"incomplete", "BOOLEAN DEFAULT 0"},
		{"forfeit", "BOOLEAN DEFAULT 0"},
	} {
		if err := database.AddColumnIfNotExists("hist_matches", col[0], col[1]); err != nil {
			return err
		}
	}
	return nil
}