package session

import (
	"database/sql"
	"time"
)

// store owns all DB operations for the session plugin.
type store struct {
	conn *sql.DB
}

// --- Models ---

type SavedSession struct {
	ID        int64     `json:"id"`
	PlayerID  string    `json:"player_id"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Games     int       `json:"games"`
	Wins      int       `json:"wins"`
	Losses    int       `json:"losses"`
	Goals     int       `json:"goals"`
	Assists   int       `json:"assists"`
	Saves     int       `json:"saves"`
	Shots     int       `json:"shots"`
	Demos     int       `json:"demos"`
}

type SessionMatch struct {
	MatchID       int64     `json:"match_id"`
	Arena         string    `json:"arena"`
	StartedAt     time.Time `json:"started_at"`
	WinnerTeamNum int       `json:"winner_team_num"`
	Incomplete    bool      `json:"incomplete"`
	Forfeit       bool      `json:"forfeit"`
	PlayerTeam    int       `json:"player_team"`
	Goals         int       `json:"goals"`
	Assists       int       `json:"assists"`
	Saves         int       `json:"saves"`
	Shots         int       `json:"shots"`
	Demos         int       `json:"demos"`
	Score         int       `json:"score"`
	PlaylistType  *int      `json:"playlist_type"`
	PlayerCount   int       `json:"player_count"`
}

type player struct {
	primaryID string
	name      string
}

// --- Sessions ---

func (s *store) createSession(playerID string, startedAt, endedAt time.Time) (int64, error) {
	res, err := s.conn.Exec(
		`INSERT INTO sessions(player_id, started_at, ended_at) VALUES(?,?,?)`,
		playerID, startedAt.UTC(), endedAt.UTC())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *store) listSessionsWithStats(playerID string) ([]SavedSession, error) {
	rows, err := s.conn.Query(
		`SELECT id, player_id, started_at, ended_at FROM sessions WHERE player_id=? ORDER BY started_at DESC LIMIT 50`,
		playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []SavedSession
	for rows.Next() {
		var sess SavedSession
		if err := rows.Scan(&sess.ID, &sess.PlayerID, &sess.StartedAt, &sess.EndedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range sessions {
		_ = s.conn.QueryRow(`
			SELECT
				COUNT(*),
				COALESCE(SUM(ms.goals),0), COALESCE(SUM(ms.assists),0),
				COALESCE(SUM(ms.saves),0),  COALESCE(SUM(ms.shots),0),
				COALESCE(SUM(ms.demos),0),
				COALESCE(SUM(CASE WHEN COALESCE(hm.incomplete,0)=0 AND hm.winner_team_num = ms.team_num THEN 1 ELSE 0 END),0),
				COALESCE(SUM(CASE WHEN COALESCE(hm.incomplete,0)=0 AND hm.winner_team_num >= 0 AND hm.winner_team_num != ms.team_num THEN 1 ELSE 0 END),0)
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

func (s *store) deleteSession(id int64) error {
	_, err := s.conn.Exec(`DELETE FROM sessions WHERE id=?`, id)
	return err
}

func (s *store) updateSession(id int64, startedAt, endedAt time.Time) error {
	_, err := s.conn.Exec(
		`UPDATE sessions SET started_at=?, ended_at=? WHERE id=?`,
		startedAt.UTC(), endedAt.UTC(), id)
	return err
}

// --- Queries over history tables ---

// sessionMatchesByPlayer returns per-player match stats for matches starting at or after since.
func (s *store) sessionMatchesByPlayer(since time.Time, playerID string) ([]SessionMatch, error) {
	rows, err := s.conn.Query(`
		SELECT m.id, COALESCE(m.arena,''), m.started_at, COALESCE(m.winner_team_num,-1),
		       COALESCE(m.incomplete,0), COALESCE(m.forfeit,0),
		       s.team_num, s.goals, s.assists, s.saves, s.shots, s.demos, s.score,
		       m.playlist_type,
		       (SELECT COUNT(*) FROM hist_player_match_stats WHERE match_id = m.id)
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
		var pt sql.NullInt64
		if err := rows.Scan(&sm.MatchID, &sm.Arena, &sm.StartedAt, &sm.WinnerTeamNum,
			&sm.Incomplete, &sm.Forfeit, &sm.PlayerTeam, &sm.Goals, &sm.Assists, &sm.Saves, &sm.Shots, &sm.Demos, &sm.Score,
			&pt, &sm.PlayerCount); err != nil {
			return nil, err
		}
		if pt.Valid {
			v := int(pt.Int64)
			sm.PlaylistType = &v
		}
		out = append(out, sm)
	}
	return out, rows.Err()
}

// mostFrequentPlayer returns the player who appears in the most matches.
// Returns nil, nil when no match data exists yet.
func (s *store) mostFrequentPlayer() (*player, error) {
	var p player
	err := s.conn.QueryRow(`
		SELECT s.primary_id, pl.name
		FROM hist_player_match_stats s
		JOIN hist_players pl ON pl.primary_id = s.primary_id
		GROUP BY s.primary_id
		ORDER BY COUNT(*) DESC
		LIMIT 1`).Scan(&p.primaryID, &p.name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}