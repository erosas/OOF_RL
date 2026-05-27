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

func TestRunMigration(t *testing.T) {
	d := newTestDB(t)
	schema := `CREATE TABLE IF NOT EXISTS test_migration (id INTEGER PRIMARY KEY, name TEXT)`
	if err := d.RunMigration(schema); err != nil {
		t.Fatalf("RunMigration: %v", err)
	}
	// Verify the table exists by inserting a row.
	if _, err := d.Conn().Exec(`INSERT INTO test_migration(name) VALUES(?)`, "hello"); err != nil {
		t.Fatalf("insert after migration: %v", err)
	}
}

func TestRunMigrationBadSQL(t *testing.T) {
	d := newTestDB(t)
	if err := d.RunMigration("THIS IS NOT SQL"); err == nil {
		t.Fatal("expected error for invalid SQL")
	}
}

func TestAddColumnIfNotExists(t *testing.T) {
	d := newTestDB(t)
	if err := d.RunMigration(`CREATE TABLE IF NOT EXISTS test_cols (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Add a new column — should succeed.
	if err := d.AddColumnIfNotExists("test_cols", "nickname", "TEXT"); err != nil {
		t.Fatalf("AddColumnIfNotExists new column: %v", err)
	}
	// Add the same column again — silently ignored.
	if err := d.AddColumnIfNotExists("test_cols", "nickname", "TEXT"); err != nil {
		t.Fatalf("AddColumnIfNotExists duplicate column: %v", err)
	}
	// Add column to a non-existent table — silently ignored.
	if err := d.AddColumnIfNotExists("no_such_table", "col", "TEXT"); err != nil {
		t.Fatalf("AddColumnIfNotExists nonexistent table: %v", err)
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