// Package sdk defines the ABI contract between the OOF_RL host and WASM plugins.
// Both sides import this package: the host to drive the protocol, plugins to
// implement it. Only abi.go is compiled on the host side; pdk.go is guest-only
// (wasip1 build tag).
package sdk

// Certainty mirrors oofevents.Certainty on the guest side.
// Values are intentionally identical so the host can cast without mapping.
type Certainty int

const (
	Authoritative Certainty = iota
	Inferred
	Signal
)

// DeclaredEvent describes an event type the plugin may emit via PublishEvent.
type DeclaredEvent struct {
	Type        string    `json:"type"`
	Certainty   Certainty `json:"certainty"`
	Description string    `json:"description,omitempty"`
}

// SettingSchema declares a single configurable value the plugin needs.
// The host resolves each key against its config store and passes the
// resulting map to plugin_apply_settings on startup (and on change).
type SettingSchema struct {
	Key         string `json:"key"`
	Description string `json:"description,omitempty"`
	Secret      bool   `json:"secret,omitempty"` // hint to UI: render as password field
}

// HTTPFetchRequest describes an outbound HTTP request sent via host_http_fetch.
type HTTPFetchRequest struct {
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body,omitempty"`
	BodyBytes []byte            `json:"body_bytes,omitempty"` // binary body; takes precedence over Body
}

// HTTPFetchResult is returned by the host's host_http_fetch import.
type HTTPFetchResult struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body"`
	Error   string            `json:"error,omitempty"`
}

// PluginMeta is returned by plugin_metadata() at load time.
// It is the plugin's identity card — the host uses it to wire routes,
// nav tabs, event subscriptions, and dependency ordering without calling any
// other function first.
type PluginMeta struct {
	ID             string           `json:"id"`
	NavTab         NavTabMeta       `json:"nav_tab"`
	Routes         []string         `json:"routes"`
	Events         []string         `json:"events"`                    // event types to subscribe to
	Requires       []string         `json:"requires,omitempty"`        // plugin IDs that must init first
	DeclaredEvents []DeclaredEvent `json:"declared_events,omitempty"` // event types this plugin may emit
	Settings       []SettingSchema `json:"settings,omitempty"`        // config keys the plugin needs
}

// NavTabMeta describes the navigation tab the plugin contributes to the UI.
// A zero-value NavTabMeta (empty ID) means the plugin has no visible tab.
type NavTabMeta struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Order int    `json:"order"`
}

// HTTPRequest is JSON-encoded by the host and passed to plugin_handle_http().
type HTTPRequest struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Query  string `json:"query,omitempty"`
	Body   string `json:"body,omitempty"`
}

// HTTPResponse is JSON-encoded by the plugin and written into the output buffer.
type HTTPResponse struct {
	Status    int               `json:"status"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body"`
	BodyBytes []byte            `json:"body_bytes,omitempty"` // binary body; takes precedence over Body
}
