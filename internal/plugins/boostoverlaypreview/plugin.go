package boostoverlaypreview

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"OOF_RL/internal/plugin"
)

//go:embed preview.html preview.css preview.js fixture.json
var previewFS embed.FS

const previewRoute = "/internal/boost-overlay-preview"

// Plugin serves a hidden fixture-only Boost Overlay preview page.
// It has no nav tab, settings, event subscriptions, persistence, or live data wiring.
type Plugin struct {
	plugin.BasePlugin
}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) ID() string         { return "boost-overlay-preview" }
func (p *Plugin) DBPrefix() string   { return "" }
func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) NavTab() plugin.NavTab {
	return plugin.NavTab{}
}

func (p *Plugin) Routes(mux *http.ServeMux) {
	mux.HandleFunc(previewRoute, p.handlePreview)
	mux.HandleFunc(previewRoute+"/", p.handleAsset)
}

func (p *Plugin) SettingsSchema() []plugin.Setting        { return nil }
func (p *Plugin) ApplySettings(_ map[string]string) error { return nil }
func (p *Plugin) Assets() fs.FS                           { return nil }

func (p *Plugin) handlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	servePreviewFile(w, r, "preview.html")
}

func (p *Plugin) handleAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, previewRoute+"/")
	name = path.Clean("/" + name)[1:]
	switch name {
	case "preview.css", "preview.js", "fixture.json":
		servePreviewFile(w, r, name)
	default:
		http.NotFound(w, r)
	}
}

func servePreviewFile(w http.ResponseWriter, r *http.Request, name string) {
	b, err := previewFS.ReadFile(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch path.Ext(name) {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(b))
}
