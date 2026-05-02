package db

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	sql *sql.DB
}

func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path+"?_journal=WAL&_timeout=5000")
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(1)
	d := &DB{sql: conn}
	return d, d.migrate()
}

func (d *DB) Close() error { return d.sql.Close() }

func (d *DB) migrate() error {
	_, err := d.sql.Exec(`
	CREATE TABLE IF NOT EXISTS players (
		primary_id TEXT PRIMARY KEY,
		name       TEXT NOT NULL,
		last_seen  DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS matches (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		match_guid      TEXT UNIQUE,
		arena           TEXT,
		started_at      DATETIME NOT NULL,
		ended_at        DATETIME,
		winner_team_num INTEGER,
		overtime        BOOLEAN DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS player_match_stats (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		match_id    INTEGER NOT NULL REFERENCES matches(id),
		primary_id  TEXT    NOT NULL REFERENCES players(primary_id),
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

	CREATE TABLE IF NOT EXISTS goal_events (
		id                   INTEGER PRIMARY KEY AUTOINCREMENT,
		match_id             INTEGER NOT NULL REFERENCES matches(id),
		scorer_id            TEXT REFERENCES players(primary_id),
		assister_id          TEXT REFERENCES players(primary_id),
		ball_last_touch_id   TEXT REFERENCES players(primary_id),
		goal_speed           REAL,
		goal_time            REAL,
		impact_x             REAL,
		impact_y             REAL,
		impact_z             REAL,
		scored_at            DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS ball_hit_events (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		match_id      INTEGER NOT NULL REFERENCES matches(id),
		player_id     TEXT REFERENCES players(primary_id),
		pre_hit_speed REAL,
		post_hit_speed REAL,
		loc_x         REAL,
		loc_y         REAL,
		loc_z         REAL,
		hit_at        DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS tick_snapshots (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		match_id    INTEGER NOT NULL REFERENCES matches(id),
		captured_at DATETIME NOT NULL,
		raw_json    TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS tracker_cache (
		primary_id TEXT PRIMARY KEY,
		data_json  TEXT NOT NULL,
		fetched_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS bc_uploads (
		replay_name    TEXT PRIMARY KEY,
		ballchasing_id TEXT NOT NULL,
		bc_url         TEXT NOT NULL,
		uploaded_at    DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_pms_primary_id ON player_match_stats(primary_id);
	CREATE INDEX IF NOT EXISTS idx_goal_scorer     ON goal_events(scorer_id);
	CREATE INDEX IF NOT EXISTS idx_goal_match      ON goal_events(match_id);
	CREATE INDEX IF NOT EXISTS idx_bh_player       ON ball_hit_events(player_id);
	CREATE INDEX IF NOT EXISTS idx_tick_match      ON tick_snapshots(match_id);
	`)
	if err != nil {
		return err
	}
	// Idempotent column additions for schema evolution.
	_, _ = d.sql.Exec(`ALTER TABLE matches ADD COLUMN playlist_type INTEGER`)
	_, _ = d.sql.Exec(`ALTER TABLE goal_events ADD COLUMN scorer_name TEXT NOT NULL DEFAULT ''`)
	_, _ = d.sql.Exec(`ALTER TABLE goal_events ADD COLUMN assister_name TEXT NOT NULL DEFAULT ''`)
	// Remove phantom goal rows emitted by RL during goal replays (empty scorer, zero speed).
	_, _ = d.sql.Exec(`DELETE FROM goal_events WHERE (scorer_id IS NULL OR scorer_id = '') AND scorer_name = '' AND goal_speed = 0`)
	return nil
}

func (d *DB) UpsertTrackerCache(primaryID, dataJSON string) error {
	_, err := d.sql.Exec(`
		INSERT INTO tracker_cache(primary_id, data_json, fetched_at) VALUES(?,?,?)
		ON CONFLICT(primary_id) DO UPDATE SET data_json=excluded.data_json, fetched_at=excluded.fetched_at`,
		primaryID, dataJSON, time.Now())
	return err
}

func (d *DB) GetTrackerCache(primaryID string) (dataJSON string, fetchedAt time.Time, found bool, err error) {
	scanErr := d.sql.QueryRow(`SELECT data_json, fetched_at FROM tracker_cache WHERE primary_id=?`, primaryID).
		Scan(&dataJSON, &fetchedAt)
	if scanErr == sql.ErrNoRows {
		return "", time.Time{}, false, nil
	}
	if scanErr != nil {
		return "", time.Time{}, false, scanErr
	}
	return dataJSON, fetchedAt, true, nil
}

// --- Players ---

func (d *DB) UpsertPlayer(primaryID, name string) error {
	_, err := d.sql.Exec(`
		INSERT INTO players(primary_id, name, last_seen) VALUES(?,?,?)
		ON CONFLICT(primary_id) DO UPDATE SET name=excluded.name, last_seen=excluded.last_seen`,
		primaryID, name, time.Now())
	return err
}

func (d *DB) AllPlayers() ([]Player, error) {
	rows, err := d.sql.Query(`SELECT primary_id, name, last_seen FROM players ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Player
	for rows.Next() {
		var p Player
		if err := rows.Scan(&p.PrimaryID, &p.Name, &p.LastSeen); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// --- Matches ---

func (d *DB) UpsertMatch(matchGUID, arena string, startedAt time.Time) (int64, error) {
	var guid interface{} = matchGUID
	if matchGUID == "" {
		guid = nil
	}
	res, err := d.sql.Exec(`
		INSERT INTO matches(match_guid, arena, started_at) VALUES(?,?,?)
		ON CONFLICT(match_guid) DO UPDATE SET arena=excluded.arena
		RETURNING id`,
		guid, arena, startedAt)
	if err != nil {
		// fallback for no RETURNING support
		res2, err2 := d.sql.Exec(`
			INSERT OR IGNORE INTO matches(match_guid, arena, started_at) VALUES(?,?,?)`,
			guid, arena, startedAt)
		if err2 != nil {
			return 0, err2
		}
		id, _ := res2.LastInsertId()
		if id == 0 {
			var found int64
			_ = d.sql.QueryRow(`SELECT id FROM matches WHERE match_guid=?`, matchGUID).Scan(&found)
			return found, nil
		}
		return id, nil
	}
	return res.LastInsertId()
}

func (d *DB) UpdateMatchPlaylist(matchID int64, playlistType int) error {
	_, err := d.sql.Exec(`UPDATE matches SET playlist_type=? WHERE id=?`, playlistType, matchID)
	return err
}

func (d *DB) EndMatch(matchID int64, winnerTeamNum int, overtime bool) error {
	_, err := d.sql.Exec(`
		UPDATE matches SET ended_at=?, winner_team_num=?, overtime=? WHERE id=?`,
		time.Now(), winnerTeamNum, overtime, matchID)
	return err
}

func (d *DB) Matches(playerID string) ([]Match, error) {
	query := `
		SELECT m.id, COALESCE(m.match_guid,''), COALESCE(m.arena,''), m.started_at,
		       COALESCE(m.ended_at,''), COALESCE(m.winner_team_num,-1), m.overtime,
		       m.playlist_type
		FROM matches m`
	var args []any
	if playerID != "" {
		query += ` JOIN player_match_stats p ON p.match_id=m.id WHERE p.primary_id=?`
		args = append(args, playerID)
	}
	query += ` ORDER BY m.started_at DESC LIMIT 200`
	rows, err := d.sql.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Match
	for rows.Next() {
		var m Match
		var pt sql.NullInt64
		if err := rows.Scan(&m.ID, &m.MatchGUID, &m.Arena, &m.StartedAt,
			&m.EndedAt, &m.WinnerTeamNum, &m.Overtime, &pt); err != nil {
			return nil, err
		}
		if pt.Valid {
			v := int(pt.Int64)
			m.PlaylistType = &v
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (d *DB) MatchPlayers(matchID int64) ([]PlayerMatchStats, error) {
	rows, err := d.sql.Query(`
		SELECT s.primary_id, p.name, s.team_num, s.score, s.goals, s.shots,
		       s.assists, s.saves, s.touches, s.car_touches, s.demos
		FROM player_match_stats s
		JOIN players p ON p.primary_id=s.primary_id
		WHERE s.match_id=?
		ORDER BY s.team_num, s.score DESC`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PlayerMatchStats
	for rows.Next() {
		var s PlayerMatchStats
		if err := rows.Scan(&s.PrimaryID, &s.Name, &s.TeamNum, &s.Score,
			&s.Goals, &s.Shots, &s.Assists, &s.Saves, &s.Touches, &s.CarTouches, &s.Demos); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// --- Player match stats ---

func (d *DB) UpsertPlayerMatchStats(matchID int64, primaryID string, teamNum, score, goals, shots, assists, saves, touches, carTouches, demos int) error {
	_, err := d.sql.Exec(`
		INSERT INTO player_match_stats(match_id,primary_id,team_num,score,goals,shots,assists,saves,touches,car_touches,demos)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(match_id,primary_id) DO UPDATE SET
			team_num=excluded.team_num, score=excluded.score, goals=excluded.goals,
			shots=excluded.shots, assists=excluded.assists, saves=excluded.saves,
			touches=excluded.touches, car_touches=excluded.car_touches, demos=excluded.demos`,
		matchID, primaryID, teamNum, score, goals, shots, assists, saves, touches, carTouches, demos)
	return err
}

func (d *DB) PlayerAggregate(primaryID string) (*PlayerAggregate, error) {
	var a PlayerAggregate
	err := d.sql.QueryRow(`
		SELECT p.name,
		       COUNT(DISTINCT s.match_id),
		       SUM(s.goals), SUM(s.shots), SUM(s.assists),
		       SUM(s.saves), SUM(s.demos), SUM(s.touches)
		FROM player_match_stats s
		JOIN players p ON p.primary_id=s.primary_id
		WHERE s.primary_id=?
		GROUP BY s.primary_id`, primaryID).Scan(
		&a.Name, &a.Matches, &a.Goals, &a.Shots,
		&a.Assists, &a.Saves, &a.Demos, &a.Touches)
	if err != nil {
		return nil, err
	}
	a.PrimaryID = primaryID
	return &a, nil
}

// AllTeamGoals returns a map of match_id → [team0Goals, team1Goals] summed from player stats.
func (d *DB) AllTeamGoals() (map[int64][2]int, error) {
	rows, err := d.sql.Query(`
		SELECT match_id, team_num, SUM(goals)
		FROM player_match_stats
		GROUP BY match_id, team_num`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64][2]int{}
	for rows.Next() {
		var mid int64
		var teamNum, goals int
		if err := rows.Scan(&mid, &teamNum, &goals); err != nil {
			return nil, err
		}
		v := out[mid]
		if teamNum == 0 {
			v[0] = goals
		} else if teamNum == 1 {
			v[1] = goals
		}
		out[mid] = v
	}
	return out, rows.Err()
}

// --- Goal events ---

func (d *DB) InsertGoal(matchID int64, scorerID, scorerName, assisterID, assisterName, lastTouchID string, speed, goalTime, ix, iy, iz float64) error {
	var as, lt interface{}
	if assisterID != "" {
		as = assisterID
	}
	if lastTouchID != "" {
		lt = lastTouchID
	}
	_, err := d.sql.Exec(`
		INSERT INTO goal_events(match_id,scorer_id,scorer_name,assister_id,assister_name,ball_last_touch_id,goal_speed,goal_time,impact_x,impact_y,impact_z,scored_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		matchID, scorerID, scorerName, as, assisterName, lt, speed, goalTime, ix, iy, iz, time.Now())
	return err
}

func (d *DB) MatchGoals(matchID int64) ([]GoalEvent, error) {
	rows, err := d.sql.Query(`
		SELECT g.id, COALESCE(g.scorer_id,''),
		       COALESCE(NULLIF(g.scorer_name,''), sp.name, ''),
		       COALESCE(g.assister_id,''),
		       COALESCE(NULLIF(g.assister_name,''), ap.name, ''),
		       g.goal_speed, g.goal_time, g.impact_x, g.impact_y, g.impact_z, g.scored_at
		FROM goal_events g
		LEFT JOIN players sp ON sp.primary_id=g.scorer_id
		LEFT JOIN players ap ON ap.primary_id=g.assister_id
		WHERE g.match_id=?
		ORDER BY g.scored_at`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GoalEvent
	for rows.Next() {
		var g GoalEvent
		if err := rows.Scan(&g.ID, &g.ScorerID, &g.ScorerName,
			&g.AssisterID, &g.AssisterName,
			&g.GoalSpeed, &g.GoalTime,
			&g.ImpactX, &g.ImpactY, &g.ImpactZ, &g.ScoredAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// --- Ball hits ---

func (d *DB) InsertBallHit(matchID int64, playerID string, pre, post, x, y, z float64) error {
	_, err := d.sql.Exec(`
		INSERT INTO ball_hit_events(match_id,player_id,pre_hit_speed,post_hit_speed,loc_x,loc_y,loc_z,hit_at)
		VALUES(?,?,?,?,?,?,?,?)`,
		matchID, playerID, pre, post, x, y, z, time.Now())
	return err
}

// --- Tick snapshots ---

func (d *DB) InsertTick(matchID int64, raw string) error {
	_, err := d.sql.Exec(`
		INSERT INTO tick_snapshots(match_id,captured_at,raw_json) VALUES(?,?,?)`,
		matchID, time.Now(), raw)
	return err
}

// --- Ballchasing upload cache ---

func (d *DB) UpsertBCUpload(replayName, bcID, bcURL string) error {
	_, err := d.sql.Exec(`
		INSERT INTO bc_uploads(replay_name, ballchasing_id, bc_url, uploaded_at) VALUES(?,?,?,?)
		ON CONFLICT(replay_name) DO UPDATE SET
			ballchasing_id=excluded.ballchasing_id,
			bc_url=excluded.bc_url,
			uploaded_at=excluded.uploaded_at`,
		replayName, bcID, bcURL, time.Now())
	return err
}

func (d *DB) AllBCUploads() (map[string]BCUpload, error) {
	rows, err := d.sql.Query(`SELECT replay_name, ballchasing_id, bc_url, uploaded_at FROM bc_uploads`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]BCUpload{}
	for rows.Next() {
		var u BCUpload
		if err := rows.Scan(&u.ReplayName, &u.BallchasingID, &u.BCURL, &u.UploadedAt); err != nil {
			return nil, err
		}
		out[u.ReplayName] = u
	}
	return out, rows.Err()
}

// MatchPlayerCounts returns a map of match_id → total player count (both teams).
func (d *DB) MatchPlayerCounts() (map[int64]int, error) {
	rows, err := d.sql.Query(`SELECT match_id, COUNT(*) FROM player_match_stats GROUP BY match_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]int{}
	for rows.Next() {
		var mid int64
		var cnt int
		if err := rows.Scan(&mid, &cnt); err != nil {
			return nil, err
		}
		out[mid] = cnt
	}
	return out, rows.Err()
}