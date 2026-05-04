package ballchasing

import (
	"path/filepath"
	"testing"
	"time"

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
			ballchasing_id TEXT NOT NULL
		);
	`); err != nil {
		t.Fatalf("RunMigration: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return &store{conn: database.Conn()}
}

func TestUpsertAndFetchBCUploads(t *testing.T) {
	s := newTestStore(t)

	if err := s.upsertBCUpload("match1.replay", "bc-id-1"); err != nil {
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
}

func TestUpsertBCUploadIdempotent(t *testing.T) {
	s := newTestStore(t)

	s.upsertBCUpload("match1.replay", "bc-id-1")
	if err := s.upsertBCUpload("match1.replay", "bc-id-2"); err != nil {
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

	s.upsertBCUpload("a.replay", "id-a")
	s.upsertBCUpload("b.replay", "id-b")
	s.upsertBCUpload("c.replay", "id-c")

	uploads, err := s.allBCUploads()
	if err != nil {
		t.Fatalf("allBCUploads: %v", err)
	}
	if len(uploads) != 3 {
		t.Errorf("expected 3 uploads, got %d", len(uploads))
	}
}

// --- normalizeGUID ---

func TestNormalizeGUID(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"024690394AE0B6BB20BBD1A3EFB2DA1E", "024690394AE0B6BB20BBD1A3EFB2DA1E"},
		{"02469039-4ae0-b6bb-20bb-d1a3efb2da1e", "024690394AE0B6BB20BBD1A3EFB2DA1E"},
		{"{02469039-4AE0-B6BB-20BB-D1A3EFB2DA1E}", "{024690394AE0B6BB20BBD1A3EFB2DA1E}"},
		{"", ""},
	}
	for _, c := range cases {
		got := normalizeGUID(c.in)
		if got != c.want {
			t.Errorf("normalizeGUID(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// --- matchReplayFiles ---

func TestMatchReplayFilesBasic(t *testing.T) {
	now := time.Now()
	matches := []MatchUploadStatus{
		{MatchGUID: "A", StartedAt: now.Add(-20 * time.Minute)},
	}
	files := []replayFileEntry{
		{name: "game.replay", modTime: now.Add(-10 * time.Minute)},
	}
	got := matchReplayFiles(files, matches)
	if got[0] != "game.replay" {
		t.Errorf("expected game.replay assigned to match 0, got %q", got[0])
	}
}

func TestMatchReplayFilesOutsideWindow(t *testing.T) {
	now := time.Now()
	matches := []MatchUploadStatus{
		{MatchGUID: "A", StartedAt: now.Add(-60 * time.Minute)},
	}
	files := []replayFileEntry{
		{name: "old.replay", modTime: now.Add(-10 * time.Minute)},
	}
	got := matchReplayFiles(files, matches)
	if _, ok := got[0]; ok {
		t.Errorf("file should not be assigned to match outside 30-min window")
	}
}

func TestMatchReplayFilesFileBeforeMatch(t *testing.T) {
	now := time.Now()
	matches := []MatchUploadStatus{
		{MatchGUID: "A", StartedAt: now.Add(-5 * time.Minute)},
	}
	// File was written before the match started — should not match.
	files := []replayFileEntry{
		{name: "early.replay", modTime: now.Add(-10 * time.Minute)},
	}
	got := matchReplayFiles(files, matches)
	if _, ok := got[0]; ok {
		t.Errorf("file written before match should not be assigned")
	}
}

func TestMatchReplayFilesOneToOne(t *testing.T) {
	now := time.Now()
	// Two back-to-back matches.
	matches := []MatchUploadStatus{
		{MatchGUID: "A", StartedAt: now.Add(-50 * time.Minute)},
		{MatchGUID: "B", StartedAt: now.Add(-20 * time.Minute)},
	}
	// One file that falls in both windows — should go to the later (more recent) match.
	files := []replayFileEntry{
		{name: "game.replay", modTime: now.Add(-15 * time.Minute)},
	}
	got := matchReplayFiles(files, matches)
	if got[1] != "game.replay" {
		t.Errorf("expected file assigned to later match (idx 1), got idx 0=%q idx 1=%q", got[0], got[1])
	}
	if _, ok := got[0]; ok {
		t.Errorf("earlier match should have no file assigned")
	}
}

func TestMatchReplayFilesEachMatchGetsOneFile(t *testing.T) {
	now := time.Now()
	matches := []MatchUploadStatus{
		{MatchGUID: "A", StartedAt: now.Add(-80 * time.Minute)},
		{MatchGUID: "B", StartedAt: now.Add(-40 * time.Minute)},
	}
	files := []replayFileEntry{
		{name: "first.replay", modTime: now.Add(-70 * time.Minute)},
		{name: "second.replay", modTime: now.Add(-30 * time.Minute)},
	}
	got := matchReplayFiles(files, matches)
	if got[0] != "first.replay" {
		t.Errorf("match A: expected first.replay, got %q", got[0])
	}
	if got[1] != "second.replay" {
		t.Errorf("match B: expected second.replay, got %q", got[1])
	}
}