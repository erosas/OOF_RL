package dashboard

import (
	"path/filepath"
	"testing"

	"OOF_RL/internal/db"
)

func TestDashboardInitRunsMigration(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	p := &Plugin{}
	if err := p.Init(nil, nil, database); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if p.conn == nil {
		t.Fatal("Init should set conn when database is provided")
	}

	// Verify migration ran by inserting a row.
	_, err = database.Conn().Exec(`INSERT OR IGNORE INTO dash_layout (id, layout_json) VALUES (1, '[]')`)
	if err != nil {
		t.Fatalf("dash_layout table not created by Init: %v", err)
	}
}

func TestDashboardInitNilDatabase(t *testing.T) {
	p := &Plugin{}
	if err := p.Init(nil, nil, nil); err != nil {
		t.Fatalf("Init with nil db should not error: %v", err)
	}
	if p.conn != nil {
		t.Fatal("conn should remain nil when no database provided")
	}
}