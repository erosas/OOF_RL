package ballchasing

import (
	"database/sql"
	"time"
)

// store owns all DB operations for the ballchasing plugin.
type store struct {
	conn *sql.DB
}

// --- Models ---

type BCUpload struct {
	ReplayName    string `json:"replay_name"`
	BallchasingID string `json:"ballchasing_id"`
}

// MatchUploadStatus is returned by recentMatches.
type MatchUploadStatus struct {
	MatchGUID string    `json:"match_guid"`
	Arena     string    `json:"arena"`
	StartedAt time.Time `json:"started_at"`
}

// --- Queries ---

func (s *store) upsertBCUpload(replayName, bcID string) error {
	_, err := s.conn.Exec(`
		INSERT INTO bc_uploads(replay_name, ballchasing_id) VALUES(?,?)
		ON CONFLICT(replay_name) DO UPDATE SET
			ballchasing_id=excluded.ballchasing_id`,
		replayName, bcID)
	return err
}

// recentMatches returns matches that started at or after since, newest-first.
func (s *store) recentMatches(since time.Time) ([]MatchUploadStatus, error) {
	rows, err := s.conn.Query(`
		SELECT match_guid, COALESCE(arena,''), started_at
		FROM hist_matches
		WHERE match_guid IS NOT NULL AND match_guid != '' AND started_at >= ?
		ORDER BY started_at DESC
		LIMIT 200`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MatchUploadStatus
	for rows.Next() {
		var m MatchUploadStatus
		if err := rows.Scan(&m.MatchGUID, &m.Arena, &m.StartedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *store) allBCUploads() (map[string]BCUpload, error) {
	rows, err := s.conn.Query(`SELECT replay_name, ballchasing_id FROM bc_uploads`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]BCUpload{}
	for rows.Next() {
		var u BCUpload
		if err := rows.Scan(&u.ReplayName, &u.BallchasingID); err != nil {
			return nil, err
		}
		out[u.ReplayName] = u
	}
	return out, rows.Err()
}