package wasmhost

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/tetratelabs/wazero/api"

	"OOF_RL/internal/httputil"

	sdk "github.com/erosas/oof-plugin-sdk"
)

func (p *Plugin) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if p.fnHandleHTTP == nil {
		httputil.JSONError(w, http.StatusNotImplemented, "plugin has no HTTP handler")
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	var bodyStr string
	if r.Body != nil {
		if b, err := io.ReadAll(r.Body); err == nil {
			bodyStr = string(b)
		}
	}
	req := sdk.HTTPRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: bodyStr}
	reqJSON, _ := json.Marshal(req)

	reqPtr, reqSize, err := p.writeGuest(reqJSON)
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer p.free(reqPtr, reqSize)

	outPtr, err := p.malloc(respBufSize)
	if err != nil {
		httputil.JSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer p.free(outPtr, respBufSize)

	res, err := p.fnHandleHTTP.Call(p.ctx,
		api.EncodeU32(reqPtr), api.EncodeU32(uint32(len(reqJSON))),
		api.EncodeU32(outPtr), api.EncodeU32(respBufSize),
	)
	if err != nil {
		log.Printf("[wasm:%s] plugin_handle_http: %v", p.meta.ID, err)
		httputil.JSONError(w, http.StatusInternalServerError, "plugin error")
		return
	}

	respData, ok := p.mod.Memory().Read(outPtr, api.DecodeU32(res[0]))
	if !ok {
		httputil.JSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var resp sdk.HTTPResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		log.Printf("[wasm:%s] parse response: %v", p.meta.ID, err)
		httputil.JSONError(w, http.StatusInternalServerError, "plugin error")
		return
	}

	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}
	w.WriteHeader(resp.Status)
	fmt.Fprint(w, resp.Body)
}

// writeResult copies data into guest memory at outPtr.
// Returns the number of bytes written, or 0 if data exceeds outMax (truncation would
// corrupt JSON-encoded results, so it is treated as an error).
func (p *Plugin) writeResult(m api.Module, data []byte, outPtr, outMax uint32) uint32 {
	if uint32(len(data)) > outMax {
		log.Printf("[wasm:%s] writeResult: result size %d exceeds buffer %d", p.meta.ID, len(data), outMax)
		return 0
	}
	if !m.Memory().Write(outPtr, data) {
		return 0
	}
	return uint32(len(data))
}
