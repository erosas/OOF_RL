package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
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

// Conn exposes the raw *sql.DB so plugins can own their own query logic.
func (d *DB) Conn() *sql.DB { return d.sql }

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

// AddColumnIfNotExists adds a column to a table if it doesn't already exist.
// Safe to call on every startup — silently ignores "duplicate column name" errors.
func (d *DB) AddColumnIfNotExists(table, column, columnDef string) error {
	_, err := d.sql.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + columnDef)
	if err != nil && (strings.Contains(err.Error(), "duplicate column name") || strings.Contains(err.Error(), "no such table")) {
		return nil
	}
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
