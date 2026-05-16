package overlayhud

import (
	"fmt"
	"net/http"
	"time"
)

const previewRoutePath = "/internal/momentum-overlay-preview"

func (p *Plugin) handlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	view, ok := p.momentumViewModel(time.Now())
	if !ok {
		http.Error(w, "momentum snapshot provider unavailable", http.StatusServiceUnavailable)
		return
	}

	model := buildRenderModel(view)
	svg := RenderSVG(model)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, "<!doctype html><html><head><meta charset=\"utf-8\"><title>Momentum Overlay Preview</title></head><body>%s</body></html>", svg)
}
