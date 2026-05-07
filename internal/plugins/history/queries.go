package history

import (
	"database/sql"
	"time"
)

// store owns all DB operations for the history plugin.
type store struct {
	conn *sql.DB
}

// --- Models ---

type Player struct {
	PrimaryID string    `json:"PrimaryID"`
	Name      string    `json:"Name"`
	LastSeen  time.Time `json:"LastSeen"`
}

type Match struct {
	ID            int64
	MatchGUID     string
	Arena         string
	StartedAt     time.Time
	EndedAt       string
	WinnerTeamNum int
	Overtime      bool
	Incomplete    bool
	Forfeit       bool
	PlaylistType  *int
	TeamScore0    *int
	TeamScore1    *int
}

type PlayerMatchStats struct {
	PrimaryID  string
	Name       string
	TeamNum    int
	Score      int
	Goals      int
	Shots      int
	Assists    int
	Saves      int
	Touches    int
	CarTouches int
	Demos      int
}

type GoalEvent struct {
	ID           int64
	ScorerID     string
	ScorerName   string
	AssisterID   string
	AssisterName string
	GoalSpeed    float64
	GoalTime     float64
	ImpactX      float64
	ImpactY      float64
	ImpactZ      float64
	ScoredAt     time.Time
}

type StatfeedEvent struct {
	ID         int64     `json:"id"`
	PlayerID   string    `json:"player_id"`
	PlayerName string    `json:"player_name"`
	TeamNum    int       `json:"team_num"`
	EventType  string    `json:"event_type"`
	TargetID   string    `json:"target_id"`
	TargetName string    `json:"target_name"`
	OccurredAt time.Time `json:"occurred_at"`
}

type PlayerAggregate struct {
	PrimaryID string
	Name      string
	Matches   int
	Goals     int
	Shots     int
	Assists   int
	Saves     int
	Demos     int
	Touches   int
}

// --- Single match lookup ---

func (s *store) matchByID(id int64) (*Match, error) {
	var m Match
	var pt, ts0, ts1 sql.NullInt64
	err := s.conn.QueryRow(`
		SELECT id, COALESCE(match_guid,''), COALESCE(arena,''), started_at,
		       COALESCE(ended_at,''), COALESCE(winner_team_num,-1), overtime,
		       COALESCE(incomplete,0), COALESCE(forfeit,0), playlist_type, team_score_0, team_score_1
		FROM hist_matches WHERE id=?`, id).Scan(
		&m.ID, &m.MatchGUID, &m.Arena, &m.StartedAt,
		&m.EndedAt, &m.WinnerTeamNum, &m.Overtime, &m.Incomplete, &m.Forfeit, &pt, &ts0, &ts1)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if pt.Valid {
		v := int(pt.Int64)
		m.PlaylistType = &v
	}
	if ts0.Valid {
		v := int(ts0.Int64)
		m.TeamScore0 = &v
	}
	if ts1.Valid {
		v := int(ts1.Int64)
		m.TeamScore1 = &v
	}
	return &m, nil
}

// --- Players ---

func (s *store) upsertPlayer(primaryID, name string) error {
	_, err := s.conn.Exec(`
		INSERT INTO hist_players(primary_id, name, last_seen) VALUES(?,?,?)
		ON CONFLICT(primary_id) DO UPDATE SET name=excluded.name, last_seen=excluded.last_seen`,
		primaryID, name, time.Now())
	return err
}

func (s *store) allPlayers() ([]Player, error) {
	rows, err := s.conn.Query(`SELECT primary_id, name, last_seen FROM hist_players ORDER BY name`)
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

func (s *store) upsertMatch(matchGUID, arena string, startedAt time.Time) (int64, error) {
	var guid interface{} = matchGUID
	if matchGUID == "" {
		guid = nil
	}
	res, err := s.conn.Exec(`
		INSERT INTO hist_matches(match_guid, arena, started_at) VALUES(?,?,?)
		ON CONFLICT(match_guid) DO UPDATE SET arena=CASE WHEN excluded.arena!='' THEN excluded.arena ELSE arena END
		RETURNING id`,
		guid, arena, startedAt)
	if err != nil {
		res2, err2 := s.conn.Exec(`INSERT OR IGNORE INTO hist_matches(match_guid, arena, started_at) VALUES(?,?,?)`,
			guid, arena, startedAt)
		if err2 != nil {
			return 0, err2
		}
		id, _ := res2.LastInsertId()
		if id == 0 {
			var found int64
			_ = s.conn.QueryRow(`SELECT id FROM hist_matches WHERE match_guid=?`, matchGUID).Scan(&found)
			return found, nil
		}
		return id, nil
	}
	return res.LastInsertId()
}

func (s *store) updateMatchPlaylist(matchID int64, playlistType int) error {
	_, err := s.conn.Exec(`UPDATE hist_matches SET playlist_type=? WHERE id=?`, playlistType, matchID)
	return err
}

func (s *store) endMatch(matchID int64, winnerTeamNum int, overtime, incomplete, forfeit bool) error {
	_, err := s.conn.Exec(`
		UPDATE hist_matches SET ended_at=?, winner_team_num=?, overtime=?, incomplete=?, forfeit=? WHERE id=?`,
		time.Now(), winnerTeamNum, overtime, incomplete, forfeit, matchID)
	return err
}

func (s *store) updateTeamScores(matchID int64, score0, score1 int) error {
	_, err := s.conn.Exec(`UPDATE hist_matches SET team_score_0=?, team_score_1=? WHERE id=?`, score0, score1, matchID)
	return err
}

func (s *store) matches(playerID string) ([]Match, error) {
	query := `
		SELECT m.id, COALESCE(m.match_guid,''), COALESCE(m.arena,''), m.started_at,
		       COALESCE(m.ended_at,''), COALESCE(m.winner_team_num,-1), m.overtime,
		       COALESCE(m.incomplete,0), COALESCE(m.forfeit,0), m.playlist_type, m.team_score_0, m.team_score_1
		FROM hist_matches m`
	var args []any
	if playerID != "" {
		query += ` JOIN hist_player_match_stats p ON p.match_id=m.id WHERE p.primary_id=?`
		args = append(args, playerID)
	}
	query += ` ORDER BY m.started_at DESC LIMIT 200`
	rows, err := s.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Match
	for rows.Next() {
		var m Match
		var pt, ts0, ts1 sql.NullInt64
		if err := rows.Scan(&m.ID, &m.MatchGUID, &m.Arena, &m.StartedAt,
			&m.EndedAt, &m.WinnerTeamNum, &m.Overtime, &m.Incomplete, &m.Forfeit, &pt, &ts0, &ts1); err != nil {
			return nil, err
		}
		if pt.Valid {
			v := int(pt.Int64)
			m.PlaylistType = &v
		}
		if ts0.Valid {
			v := int(ts0.Int64)
			m.TeamScore0 = &v
		}
		if ts1.Valid {
			v := int(ts1.Int64)
			m.TeamScore1 = &v
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *store) matchPlayers(matchID int64) ([]PlayerMatchStats, error) {
	rows, err := s.conn.Query(`
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

func (s *store) matchPlayerCounts() (map[int64]int, error) {
	rows, err := s.conn.Query(`SELECT match_id, COUNT(*) FROM hist_player_match_stats GROUP BY match_id`)
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

func (s *store) matchBotCounts() (map[int64]int, error) {
	rows, err := s.conn.Query(`
		SELECT match_id, COUNT(*)
		FROM hist_player_match_stats
		WHERE lower(primary_id) LIKE 'unknown|%' OR lower(primary_id) LIKE 'bot:%'
		GROUP BY match_id`)
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

// --- Player match stats ---

func (s *store) upsertPlayerMatchStats(matchID int64, primaryID string, teamNum, score, goals, shots, assists, saves, touches, carTouches, demos int) error {
	_, err := s.conn.Exec(`
		INSERT INTO hist_player_match_stats(match_id,primary_id,team_num,score,goals,shots,assists,saves,touches,car_touches,demos)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(match_id,primary_id) DO UPDATE SET
			team_num=excluded.team_num, score=excluded.score, goals=excluded.goals,
			shots=excluded.shots, assists=excluded.assists, saves=excluded.saves,
			touches=excluded.touches, car_touches=excluded.car_touches, demos=excluded.demos`,
		matchID, primaryID, teamNum, score, goals, shots, assists, saves, touches, carTouches, demos)
	return err
}

func (s *store) playerAggregate(primaryID string) (*PlayerAggregate, error) {
	var a PlayerAggregate
	err := s.conn.QueryRow(`
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

func (s *store) allTeamGoals() (map[int64][2]int, error) {
	rows, err := s.conn.Query(`
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

func (s *store) insertGoal(matchID int64, scorerID, scorerName, assisterID, assisterName, lastTouchID string, speed, goalTime, ix, iy, iz float64) error {
	var as, lt interface{}
	if assisterID != "" {
		as = assisterID
	}
	if lastTouchID != "" {
		lt = lastTouchID
	}
	_, err := s.conn.Exec(`
		INSERT INTO hist_goal_events(match_id,scorer_id,scorer_name,assister_id,assister_name,ball_last_touch_id,goal_speed,goal_time,impact_x,impact_y,impact_z,scored_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		matchID, scorerID, scorerName, as, assisterName, lt, speed, goalTime, ix, iy, iz, time.Now())
	return err
}

func (s *store) matchGoals(matchID int64) ([]GoalEvent, error) {
	rows, err := s.conn.Query(`
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

func (s *store) insertBallHit(matchID int64, playerID string, pre, post, x, y, z float64) error {
	_, err := s.conn.Exec(`
		INSERT INTO hist_ball_hit_events(match_id,player_id,pre_hit_speed,post_hit_speed,loc_x,loc_y,loc_z,hit_at)
		VALUES(?,?,?,?,?,?,?,?)`,
		matchID, playerID, pre, post, x, y, z, time.Now())
	return err
}

// --- Tick snapshots ---

func (s *store) insertTick(matchID int64, raw string) error {
	_, err := s.conn.Exec(`
		INSERT INTO hist_tick_snapshots(match_id,captured_at,raw_json) VALUES(?,?,?)`,
		matchID, time.Now(), raw)
	return err
}

// --- Statfeed events ---

func (s *store) insertStatfeedEvent(matchID int64, playerID, playerName string, teamNum int, eventType, targetID, targetName string) error {
	_, err := s.conn.Exec(`
		INSERT INTO hist_statfeed_events(match_id,player_id,player_name,team_num,event_type,target_id,target_name,occurred_at)
		VALUES(?,?,?,?,?,?,?,?)`,
		matchID, playerID, playerName, teamNum, eventType, targetID, targetName, time.Now())
	return err
}

func (s *store) matchStatfeedEvents(matchID int64) ([]StatfeedEvent, error) {
	rows, err := s.conn.Query(`
		SELECT id, player_id, player_name, team_num, event_type, target_id, target_name, occurred_at
		FROM hist_statfeed_events
		WHERE match_id=?
		ORDER BY occurred_at ASC`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StatfeedEvent
	for rows.Next() {
		var e StatfeedEvent
		if err := rows.Scan(&e.ID, &e.PlayerID, &e.PlayerName, &e.TeamNum, &e.EventType, &e.TargetID, &e.TargetName, &e.OccurredAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
