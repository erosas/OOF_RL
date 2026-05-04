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
	ReplayName    string    `json:"replay_name"`
	BallchasingID string    `json:"ballchasing_id"`
	BCURL         string    `json:"bc_url"`
	UploadedAt    time.Time `json:"uploaded_at"`
}

// --- Queries ---

func (s *store) upsertBCUpload(replayName, bcID, bcURL string) error {
	_, err := s.conn.Exec(`
		INSERT INTO bc_uploads(replay_name, ballchasing_id, bc_url, uploaded_at) VALUES(?,?,?,?)
		ON CONFLICT(replay_name) DO UPDATE SET
			ballchasing_id=excluded.ballchasing_id,
			bc_url=excluded.bc_url,
			uploaded_at=excluded.uploaded_at`,
		replayName, bcID, bcURL, time.Now())
	return err
}

func (s *store) allBCUploads() (map[string]BCUpload, error) {
	rows, err := s.conn.Query(`SELECT replay_name, ballchasing_id, bc_url, uploaded_at FROM bc_uploads`)
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