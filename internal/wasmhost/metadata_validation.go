package wasmhost

import (
	"fmt"
	"net/http"
	"strings"

	sdk "github.com/erosas/oof-plugin-sdk"
)

// reservedPluginIDs are namespaces owned by the host. A plugin with one of
// these IDs could register routes that collide with core mux patterns —
// http.ServeMux panics on duplicate patterns, so a bad .wasm file would
// crash the app at startup.
var reservedPluginIDs = map[string]struct{}{
	"config":   {},
	"data-dir": {},
	"db":       {},
	"debug":    {},
	"history":  {},
	"matches":  {},
	"nav":      {},
	"overlay":  {},
	"players":  {},
	"plugins":  {},
	"settings": {},
	"tracker":  {},
	"update":   {},
	"ws":       {},
}

func validatePluginID(id string) error {
	if id == "" {
		return fmt.Errorf("plugin id is required")
	}
	if _, reserved := reservedPluginIDs[id]; reserved {
		return fmt.Errorf("plugin id %q is reserved by the host", id)
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return fmt.Errorf("plugin id %q: only lowercase letters, digits, '-' and '_' allowed", id)
		}
	}
	return nil
}

// validateRouteMeta checks a plugin's declared routes. Every route must live
// under /api/<pluginID>/ — this keeps plugins out of core and other plugins'
// namespaces (a duplicate mux pattern is a startup panic) and lets the
// frontend trust that a plugin-supplied path belongs to that plugin.
func validateRouteMeta(pluginID string, routes []sdk.RouteMeta) error {
	prefix := "/api/" + pluginID + "/"
	seenPath := make(map[string]struct{}, len(routes))
	for _, r := range routes {
		path := strings.TrimSpace(r.Path)
		if path == "" {
			return fmt.Errorf("route path is required")
		}
		if !strings.HasPrefix(path, prefix) || len(path) == len(prefix) {
			return fmt.Errorf("route path %q must be under %q", path, prefix)
		}
		// The prefix check above is a literal string comparison, so reject
		// dot segments, percent-escapes, and backslashes outright — none have
		// a legitimate use in a route pattern, and each is a way to register
		// a pattern that reads as one path but matches another.
		if strings.Contains(path, "..") || strings.ContainsAny(path, "%\\") {
			return fmt.Errorf("route path %q must not contain '..', '%%', or '\\'", path)
		}
		if _, exists := seenPath[path]; exists {
			return fmt.Errorf("duplicate route path %q", path)
		}
		seenPath[path] = struct{}{}
		method := strings.ToUpper(strings.TrimSpace(r.Method))
		if method == "" {
			continue
		}
		switch method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead, http.MethodOptions:
		default:
			return fmt.Errorf("unsupported route method %q for %q", method, path)
		}
	}
	return nil
}

func validateDeclaredEventsMeta(events []sdk.DeclaredEvent) error {
	seen := make(map[string]struct{}, len(events))
	for _, e := range events {
		typeName := strings.TrimSpace(e.Type)
		if typeName == "" {
			return fmt.Errorf("declared event type is required")
		}
		if _, exists := seen[typeName]; exists {
			return fmt.Errorf("duplicate declared event type %q", typeName)
		}
		seen[typeName] = struct{}{}
		if e.Certainty < sdk.Authoritative || e.Certainty > sdk.Signal {
			return fmt.Errorf("invalid certainty %d for declared event %q", e.Certainty, typeName)
		}
	}
	return nil
}