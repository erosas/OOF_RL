package overlayhud

import (
	"net/http"
	"time"
)

const previewRoutePath = "/internal/momentum-overlay-preview"

func (p *Plugin) handlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	html, ok := NewDisplayAdapter(p.momentum).RenderHTML(time.Now())
	if !ok {
		http.Error(w, "momentum snapshot provider unavailable", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}
