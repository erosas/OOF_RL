package wasmhost

import (
	"net/http"
	"strings"

	"OOF_RL/internal/httputil"
)

func (p *Plugin) RoutePaths() []string {
	paths := make([]string, 0, len(p.meta.Routes))
	for _, r := range p.meta.Routes {
		if r.Path != "" {
			paths = append(paths, r.Path)
		}
	}
	return paths
}

func (p *Plugin) Routes(mux *http.ServeMux) {
	for _, route := range p.meta.Routes {
		path := route.Path
		if path == "" {
			continue
		}
		method := strings.ToUpper(strings.TrimSpace(route.Method))
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if method != "" && r.Method != method {
				httputil.JSONError(w, http.StatusMethodNotAllowed, "method not allowed")
				return
			}
			p.serveHTTP(w, r)
		})
	}
}
