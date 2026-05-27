package main

import (
	"encoding/json"
	"fmt"

	sdk "github.com/erosas/oof-plugin-sdk"
)

func initPlugin() uint32 {
	sdk.DBExec(`CREATE TABLE IF NOT EXISTS dash_layout (
		id          INTEGER PRIMARY KEY CHECK (id = 1),
		layout_json TEXT    NOT NULL DEFAULT '[]'
	)`, nil)
	return 0
}

func handleHTTP(req sdk.HTTPRequest) sdk.HTTPResponse {
	switch req.Path {
	case "/api/dashboard/layout":
		return handleLayout(req)
	default:
		return sdk.JSONError(404, "not found")
	}
}

func handleLayout(req sdk.HTTPRequest) sdk.HTTPResponse {
	switch req.Method {
	case "GET":
		rows := sdk.DBQuery(`SELECT layout_json FROM dash_layout WHERE id = 1`, nil)
		raw := "[]"
		if len(rows) > 0 {
			if v, ok := rows[0]["layout_json"].(string); ok {
				raw = v
			}
		}
		return sdk.HTTPResponse{
			Status:  200,
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    raw,
		}

	case "POST":
		var raw json.RawMessage
		if err := json.Unmarshal([]byte(req.Body), &raw); err != nil {
			return sdk.JSONError(400, err.Error())
		}
		items, err := validateLayout(raw)
		if err != nil {
			return sdk.JSONError(400, err.Error())
		}
		encoded, _ := json.Marshal(items)
		sdk.DBExec(`
			INSERT INTO dash_layout (id, layout_json) VALUES (1, ?)
			ON CONFLICT(id) DO UPDATE SET layout_json = excluded.layout_json`,
			[]string{string(encoded)})
		b, _ := json.Marshal(map[string]bool{"ok": true})
		return sdk.JSONResponse(b)

	default:
		return sdk.JSONError(405, "method not allowed")
	}
}

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
