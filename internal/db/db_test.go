package db

import (
	"path/filepath"
	"testing"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestMigrate(t *testing.T) {
	d := newTestDB(t)
	if d.Conn() == nil {
		t.Error("Conn() returned nil after Open")
	}
}

func TestTrackerCache(t *testing.T) {
	d := newTestDB(t)

	_, _, found, err := d.GetTrackerCache("pid-x")
	if err != nil {
		t.Fatalf("GetTrackerCache: %v", err)
	}
	if found {
		t.Error("expected not found before insert")
	}

	payload := `{"rating":1234}`
	if err := d.UpsertTrackerCache("pid-x", payload); err != nil {
		t.Fatalf("UpsertTrackerCache: %v", err)
	}

	data, fetchedAt, found, err := d.GetTrackerCache("pid-x")
	if err != nil {
		t.Fatalf("GetTrackerCache after insert: %v", err)
	}
	if !found {
		t.Fatal("expected found after insert")
	}
	if data != payload {
		t.Errorf("data: got %q, want %q", data, payload)
	}
	if fetchedAt.IsZero() {
		t.Error("fetchedAt should not be zero")
	}

	updated := `{"rating":9999}`
	if err := d.UpsertTrackerCache("pid-x", updated); err != nil {
		t.Fatalf("UpsertTrackerCache update: %v", err)
	}
	data, _, _, _ = d.GetTrackerCache("pid-x")
	if data != updated {
		t.Errorf("updated data: got %q, want %q", data, updated)
	}
}