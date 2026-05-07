package dashboard

import (
	"database/sql"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"

	"OOF_RL/internal/db"
	"OOF_RL/internal/events"
	"OOF_RL/internal/httputil"
	"OOF_RL/internal/plugin"
)

//go:embed view.html view.js
var viewFS embed.FS

type Plugin struct {
	conn *sql.DB
}

func New(database *db.DB) *Plugin {
	if err := database.RunMigration(`
		CREATE TABLE IF NOT EXISTS dash_layout (
			id          INTEGER PRIMARY KEY CHECK (id = 1),
			layout_json TEXT    NOT NULL DEFAULT '[]'
		);
	`); err != nil {
		log.Printf("[dashboard] migrate: %v", err)
	}
	return &Plugin{conn: database.Conn()}
}

func (p *Plugin) ID() string         { return "dashboard" }
func (p *Plugin) DBPrefix() string   { return "dash" }
func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) NavTab() plugin.NavTab {
	return plugin.NavTab{ID: "dashboard", Label: "Dashboard", Order: 50}
}

func (p *Plugin) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/dashboard/layout", p.handleLayout)
}

func (p *Plugin) SettingsSchema() []plugin.Setting        { return nil }
func (p *Plugin) ApplySettings(_ map[string]string) error { return nil }
func (p *Plugin) HandleEvent(_ events.Envelope)           {}
func (p *Plugin) Assets() fs.FS                           { return viewFS }

func (p *Plugin) handleLayout(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		var raw string
		err := p.conn.QueryRowContext(r.Context(),
			`SELECT layout_json FROM dash_layout WHERE id = 1`).Scan(&raw)
		if err == sql.ErrNoRows {
			raw = "[]"
			err = nil
		}
		if err != nil {
			httputil.JSONError(w, 500, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(raw))

	case http.MethodPost:
		var payload json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			httputil.JSONError(w, 400, err.Error())
			return
		}
		_, err := p.conn.ExecContext(r.Context(), `
			INSERT INTO dash_layout (id, layout_json) VALUES (1, ?)
			ON CONFLICT(id) DO UPDATE SET layout_json = excluded.layout_json
		`, string(payload))
		if err != nil {
			httputil.JSONError(w, 500, err.Error())
			return
		}
		httputil.WriteJSON(w, map[string]bool{"ok": true})

	default:
		http.Error(w, "method not allowed", 405)
	}
}
