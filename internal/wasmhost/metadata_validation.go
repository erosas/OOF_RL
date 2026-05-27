package wasmhost

import (
	"fmt"
	"net/http"
	"strings"

	sdk "github.com/erosas/oof-plugin-sdk"
)

func validateRouteMeta(routes []sdk.RouteMeta) error {
	seenPath := make(map[string]struct{}, len(routes))
	for _, r := range routes {
		path := strings.TrimSpace(r.Path)
		if path == "" {
			return fmt.Errorf("route path is required")
		}
		if !strings.HasPrefix(path, "/") {
			return fmt.Errorf("route path must start with '/': %q", path)
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
