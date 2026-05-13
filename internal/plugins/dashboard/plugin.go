package dashboard

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"OOF_RL/internal/db"
	"OOF_RL/internal/httputil"
	"OOF_RL/internal/plugin"
)

//go:embed view.html view.js gridstack.min.css gridstack-all.js gridstack-all.js.LICENSE.txt
var viewFS embed.FS

type Plugin struct {
	plugin.BasePlugin
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
func (p *Plugin) Assets() fs.FS                           { return viewFS }

// layoutItem is the validated shape of each entry in the dashboard layout array.
type layoutItem struct {
	ID string `json:"id"`
	X  int    `json:"x"`
	Y  int    `json:"y"`
	W  int    `json:"w"`
	H  int    `json:"h"`
}

const (
	maxLayoutItems = 100
	maxGridColumns = 12
	maxGridRows    = 1000
)

func validateLayout(raw json.RawMessage) ([]layoutItem, error) {
	var items []layoutItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("layout must be a JSON array: %w", err)
	}
	if len(items) > maxLayoutItems {
		return nil, fmt.Errorf("layout must contain no more than %d items", maxLayoutItems)
	}
	for i, item := range items {
		if item.ID == "" {
			return nil, fmt.Errorf("item %d missing id", i)
		}
		if item.X < 0 || item.Y < 0 {
			return nil, fmt.Errorf("item %d (%s): x and y must be >= 0", i, item.ID)
		}
		if item.W < 1 || item.H < 1 {
			return nil, fmt.Errorf("item %d (%s): w and h must be >= 1", i, item.ID)
		}
		if item.W > maxGridColumns {
			return nil, fmt.Errorf("item %d (%s): w must be <= %d", i, item.ID, maxGridColumns)
		}
		if item.H > maxGridRows {
			return nil, fmt.Errorf("item %d (%s): h must be <= %d", i, item.ID, maxGridRows)
		}
		if item.X >= maxGridColumns {
			return nil, fmt.Errorf("item %d (%s): x must be < %d", i, item.ID, maxGridColumns)
		}
		if item.X+item.W > maxGridColumns {
			return nil, fmt.Errorf("item %d (%s): x + w must be <= %d", i, item.ID, maxGridColumns)
		}
		if item.Y > maxGridRows {
			return nil, fmt.Errorf("item %d (%s): y must be <= %d", i, item.ID, maxGridRows)
		}
	}
	return items, nil
}

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
		var raw json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			httputil.JSONError(w, 400, err.Error())
			return
		}
		items, err := validateLayout(raw)
		if err != nil {
			httputil.JSONError(w, 400, err.Error())
			return
		}
		encoded, _ := json.Marshal(items)
		_, err = p.conn.ExecContext(r.Context(), `
			INSERT INTO dash_layout (id, layout_json) VALUES (1, ?)
			ON CONFLICT(id) DO UPDATE SET layout_json = excluded.layout_json
		`, string(encoded))
		if err != nil {
			httputil.JSONError(w, 500, err.Error())
			return
		}
		httputil.WriteJSON(w, map[string]bool{"ok": true})

	default:
		http.Error(w, "method not allowed", 405)
	}
}