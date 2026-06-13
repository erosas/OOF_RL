package update

import (
	"encoding/json"
	"net/http"

	"OOF_RL/internal/httputil"
)

// HandleStatus serves GET /api/update/status.
func (c *Checker) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httputil.JSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	httputil.WriteJSON(w, c.Status())
}

// HandleCheck serves POST /api/update/check.
func (c *Checker) HandleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.JSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	st := c.Check(r.Context())
	if st.LastError != "" {
		// Failed checks still return the full status (not just an error
		// string) so the UI can keep showing version state; 502 because the
		// failure is upstream (manifest fetch/parse).
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(st)
		return
	}
	httputil.WriteJSON(w, st)
}