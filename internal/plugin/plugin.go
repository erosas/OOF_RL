package plugin

import (
	"io/fs"
	"net/http"

	"OOF_RL/internal/events"
)

// NavTab describes a navigation entry the plugin contributes to the UI.
type NavTab struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Order int    `json:"order"`
}

// Plugin is the contract every page/feature plugin must satisfy.
type Plugin interface {
	// ID returns a unique, stable string identifier ("live", "history", …).
	ID() string

	// DBPrefix returns the table-name prefix this plugin uses (e.g. "hist", "bc").
	// Must be lowercase letters only (a-z), max 8 chars. Return "" for no tables.
	// Convention: prefix_tablename, e.g. hist_matches, bc_uploads.
	DBPrefix() string

	// Requires returns plugin IDs this plugin depends on.
	// The UI prevents disabling a required plugin while a dependent is enabled.
	// Return nil for no dependencies.
	Requires() []string

	// NavTab returns the tab entry to show in the navigation bar.
	// Return a zero NavTab (empty ID) to add no nav entry.
	NavTab() NavTab

	// Routes registers the plugin's HTTP handlers on mux.
	Routes(mux *http.ServeMux)

	// SettingsSchema returns the settings this plugin contributes to the Settings page.
	SettingsSchema() []Setting

	// ApplySettings is called when the user saves settings; values are string-encoded.
	ApplySettings(values map[string]string) error

	// HandleEvent is called for every RL event envelope received from the game.
	HandleEvent(env events.Envelope)

	// Assets returns a filesystem of static files to serve at /plugins/{id}/.
	// Return nil if the plugin has no static assets.
	// A view.html file in the root is treated as the plugin's tab content fragment.
	Assets() fs.FS
}