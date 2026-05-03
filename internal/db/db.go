package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	sql *sql.DB
}

func Open(path string) (*DB, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}
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
	CREATE TABLE IF NOT EXISTS tracker_cache (
		primary_id TEXT PRIMARY KEY,
		data_json  TEXT NOT NULL,
		fetched_at DATETIME NOT NULL
	);
	`)
	return err
}

// RunMigration executes a plugin-provided DDL string. Plugins call this from
// their New() constructor to create their own tables.
func (d *DB) RunMigration(schema string) error {
	_, err := d.sql.Exec(schema)
	return err
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
		INSERT INTO hist_players(primary_id, name, last_seen) VALUES(?,?,?)
		ON CONFLICT(primary_id) DO UPDATE SET name=excluded.name, last_seen=excluded.last_seen`,
		primaryID, name, time.Now())
	return err
}

func (d *DB) AllPlayers() ([]Player, error) {
	rows, err := d.sql.Query(`SELECT primary_id, name, last_seen FROM hist_players ORDER BY name`)
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
		INSERT INTO hist_matches(match_guid, arena, started_at) VALUES(?,?,?)
		ON CONFLICT(match_guid) DO UPDATE SET arena=excluded.arena
		RETURNING id`,
		guid, arena, startedAt)
	if err != nil {
		// fallback for no RETURNING support
		res2, err2 := d.sql.Exec(`
			INSERT OR IGNORE INTO hist_matches(match_guid, arena, started_at) VALUES(?,?,?)`,
			guid, arena, startedAt)
		if err2 != nil {
			return 0, err2
		}
		id, _ := res2.LastInsertId()
		if id == 0 {
			var found int64
			_ = d.sql.QueryRow(`SELECT id FROM hist_matches WHERE match_guid=?`, matchGUID).Scan(&found)
			return found, nil
		}
		return id, nil
	}
	return res.LastInsertId()
}

func (d *DB) UpdateMatchPlaylist(matchID int64, playlistType int) error {
	_, err := d.sql.Exec(`UPDATE hist_matches SET playlist_type=? WHERE id=?`, playlistType, matchID)
	return err
}

func (d *DB) EndMatch(matchID int64, winnerTeamNum int, overtime bool) error {
	_, err := d.sql.Exec(`
		UPDATE hist_matches SET ended_at=?, winner_team_num=?, overtime=? WHERE id=?`,
		time.Now(), winnerTeamNum, overtime, matchID)
	return err
}

func (d *DB) Matches(playerID string) ([]Match, error) {
	query := `
		SELECT m.id, COALESCE(m.match_guid,''), COALESCE(m.arena,''), m.started_at,
		       COALESCE(m.ended_at,''), COALESCE(m.winner_team_num,-1), m.overtime,
		       m.playlist_type
		FROM hist_matches m`
	var args []any
	if playerID != "" {
		query += ` JOIN hist_player_match_stats p ON p.match_id=m.id WHERE p.primary_id=?`
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
		FROM hist_player_match_stats s
		JOIN hist_players p ON p.primary_id=s.primary_id
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
		INSERT INTO hist_player_match_stats(match_id,primary_id,team_num,score,goals,shots,assists,saves,touches,car_touches,demos)
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
		FROM hist_player_match_stats s
		JOIN hist_players p ON p.primary_id=s.primary_id
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
		FROM hist_player_match_stats
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
		INSERT INTO hist_goal_events(match_id,scorer_id,scorer_name,assister_id,assister_name,ball_last_touch_id,goal_speed,goal_time,impact_x,impact_y,impact_z,scored_at)
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
		FROM hist_goal_events g
		LEFT JOIN hist_players sp ON sp.primary_id=g.scorer_id
		LEFT JOIN hist_players ap ON ap.primary_id=g.assister_id
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
		INSERT INTO hist_ball_hit_events(match_id,player_id,pre_hit_speed,post_hit_speed,loc_x,loc_y,loc_z,hit_at)
		VALUES(?,?,?,?,?,?,?,?)`,
		matchID, playerID, pre, post, x, y, z, time.Now())
	return err
}

// --- Tick snapshots ---

func (d *DB) InsertTick(matchID int64, raw string) error {
	_, err := d.sql.Exec(`
		INSERT INTO hist_tick_snapshots(match_id,captured_at,raw_json) VALUES(?,?,?)`,
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

// SessionMatchesByPlayer returns per-player match stats for matches starting at or after since.
func (d *DB) SessionMatchesByPlayer(since time.Time, playerID string) ([]SessionMatch, error) {
	rows, err := d.sql.Query(`
		SELECT m.id, COALESCE(m.arena,''), m.started_at, COALESCE(m.winner_team_num,-1),
		       s.team_num, s.goals, s.assists, s.saves, s.shots, s.demos, s.score
		FROM hist_matches m
		JOIN hist_player_match_stats s ON s.match_id = m.id
		WHERE m.started_at >= ? AND s.primary_id = ?
		ORDER BY m.started_at ASC
		LIMIT 200`, since, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SessionMatch
	for rows.Next() {
		var sm SessionMatch
		if err := rows.Scan(&sm.MatchID, &sm.Arena, &sm.StartedAt, &sm.WinnerTeamNum,
			&sm.PlayerTeam, &sm.Goals, &sm.Assists, &sm.Saves, &sm.Shots, &sm.Demos, &sm.Score); err != nil {
			return nil, err
		}
		out = append(out, sm)
	}
	return out, rows.Err()
}

// MostFrequentPlayer returns the player who appears in the most matches.
// Returns nil, nil when no match data exists yet.
func (d *DB) MostFrequentPlayer() (*Player, error) {
	var p Player
	err := d.sql.QueryRow(`
		SELECT s.primary_id, pl.name
		FROM hist_player_match_stats s
		JOIN hist_players pl ON pl.primary_id = s.primary_id
		GROUP BY s.primary_id
		ORDER BY COUNT(*) DESC
		LIMIT 1`).Scan(&p.PrimaryID, &p.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// --- Sessions ---

func (d *DB) CreateSession(playerID string, startedAt, endedAt time.Time) (int64, error) {
	res, err := d.sql.Exec(
		`INSERT INTO sessions(player_id, started_at, ended_at) VALUES(?,?,?)`,
		playerID, startedAt.UTC(), endedAt.UTC())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) ListSessionsWithStats(playerID string) ([]SavedSession, error) {
	rows, err := d.sql.Query(
		`SELECT id, player_id, started_at, ended_at FROM sessions WHERE player_id=? ORDER BY started_at DESC LIMIT 50`,
		playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []SavedSession
	for rows.Next() {
		var s SavedSession
		if err := rows.Scan(&s.ID, &s.PlayerID, &s.StartedAt, &s.EndedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range sessions {
		_ = d.sql.QueryRow(`
			SELECT
				COUNT(*),
				COALESCE(SUM(ms.goals),0), COALESCE(SUM(ms.assists),0),
				COALESCE(SUM(ms.saves),0),  COALESCE(SUM(ms.shots),0),
				COALESCE(SUM(ms.demos),0),
				COALESCE(SUM(CASE WHEN hm.winner_team_num = ms.team_num THEN 1 ELSE 0 END),0),
				COALESCE(SUM(CASE WHEN hm.winner_team_num >= 0 AND hm.winner_team_num != ms.team_num THEN 1 ELSE 0 END),0)
			FROM hist_matches hm
			JOIN hist_player_match_stats ms ON ms.match_id = hm.id
			WHERE hm.started_at >= ? AND hm.started_at < ? AND ms.primary_id = ?`,
			sessions[i].StartedAt, sessions[i].EndedAt, sessions[i].PlayerID).Scan(
			&sessions[i].Games, &sessions[i].Goals, &sessions[i].Assists,
			&sessions[i].Saves, &sessions[i].Shots, &sessions[i].Demos,
			&sessions[i].Wins, &sessions[i].Losses)
	}
	return sessions, nil
}

func (d *DB) DeleteSession(id int64) error {
	_, err := d.sql.Exec(`DELETE FROM sessions WHERE id=?`, id)
	return err
}

func (d *DB) UpdateSession(id int64, startedAt, endedAt time.Time) error {
	_, err := d.sql.Exec(
		`UPDATE sessions SET started_at=?, ended_at=? WHERE id=?`,
		startedAt.UTC(), endedAt.UTC(), id)
	return err
}

// MatchPlayerCounts returns a map of match_id → total player count (both teams).
func (d *DB) MatchPlayerCounts() (map[int64]int, error) {
	rows, err := d.sql.Query(`SELECT match_id, COUNT(*) FROM hist_player_match_stats GROUP BY match_id`)
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