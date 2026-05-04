package ballchasing

import (
	"path/filepath"
	"testing"

	"OOF_RL/internal/db"
)

func newTestStore(t *testing.T) *store {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := database.RunMigration(`
		CREATE TABLE IF NOT EXISTS bc_uploads (
			replay_name    TEXT PRIMARY KEY,
			ballchasing_id TEXT NOT NULL,
			bc_url         TEXT NOT NULL,
			uploaded_at    DATETIME NOT NULL
		);
	`); err != nil {
		t.Fatalf("RunMigration: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return &store{conn: database.Conn()}
}

func TestUpsertAndFetchBCUploads(t *testing.T) {
	s := newTestStore(t)

	if err := s.upsertBCUpload("match1.replay", "bc-id-1", "https://ballchasing.com/replay/bc-id-1"); err != nil {
		t.Fatalf("upsertBCUpload: %v", err)
	}

	uploads, err := s.allBCUploads()
	if err != nil {
		t.Fatalf("allBCUploads: %v", err)
	}
	if len(uploads) != 1 {
		t.Fatalf("expected 1 upload, got %d", len(uploads))
	}
	u := uploads["match1.replay"]
	if u.BallchasingID != "bc-id-1" {
		t.Errorf("BallchasingID: got %q, want bc-id-1", u.BallchasingID)
	}
	if u.BCURL != "https://ballchasing.com/replay/bc-id-1" {
		t.Errorf("BCURL: got %q", u.BCURL)
	}
}

func TestUpsertBCUploadIdempotent(t *testing.T) {
	s := newTestStore(t)

	s.upsertBCUpload("match1.replay", "bc-id-1", "https://ballchasing.com/replay/bc-id-1")
	if err := s.upsertBCUpload("match1.replay", "bc-id-2", "https://ballchasing.com/replay/bc-id-2"); err != nil {
		t.Fatalf("upsertBCUpload update: %v", err)
	}

	uploads, _ := s.allBCUploads()
	if len(uploads) != 1 {
		t.Fatalf("expected 1 upload after upsert, got %d", len(uploads))
	}
	if uploads["match1.replay"].BallchasingID != "bc-id-2" {
		t.Errorf("expected updated BallchasingID=bc-id-2, got %q", uploads["match1.replay"].BallchasingID)
	}
}

func TestAllBCUploadsMultiple(t *testing.T) {
	s := newTestStore(t)

	s.upsertBCUpload("a.replay", "id-a", "https://ballchasing.com/replay/id-a")
	s.upsertBCUpload("b.replay", "id-b", "https://ballchasing.com/replay/id-b")
	s.upsertBCUpload("c.replay", "id-c", "https://ballchasing.com/replay/id-c")

	uploads, err := s.allBCUploads()
	if err != nil {
		t.Fatalf("allBCUploads: %v", err)
	}
	if len(uploads) != 3 {
		t.Errorf("expected 3 uploads, got %d", len(uploads))
	}
}