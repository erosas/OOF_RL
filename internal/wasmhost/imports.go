package wasmhost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tetratelabs/wazero/api"

	"OOF_RL/internal/oofevents"

	sdk "github.com/erosas/oof-plugin-sdk"
)

// hostPublishEvent is called by the guest via the "host_publish_event" import.
// It publishes a RawEvent onto the bus so other plugins can subscribe to it.
func (p *Plugin) hostPublishEvent(_ context.Context, m api.Module, certainty, typePtr, typeLen, payloadPtr, payloadLen uint32) {
	if p.bus == nil {
		return
	}
	typeBytes, ok := m.Memory().Read(typePtr, typeLen)
	if !ok {
		log.Printf("[wasm:%s] host_publish_event: type read failed", p.meta.ID)
		return
	}
	payload, ok := m.Memory().Read(payloadPtr, payloadLen)
	if !ok {
		log.Printf("[wasm:%s] host_publish_event: payload read failed", p.meta.ID)
		return
	}
	e := oofevents.RawEvent{
		Base: oofevents.Base{
			EventType: string(typeBytes),
			At:        time.Now(),
			Cert:      oofevents.Certainty(certainty),
		},
		Payload: json.RawMessage(payload),
	}
	switch e.Cert {
	case oofevents.Authoritative:
		p.bus.PublishAuthoritative(e)
	case oofevents.Inferred:
		p.bus.PublishInferred(e)
	case oofevents.Signal:
		p.bus.PublishSignal(e)
	default:
		log.Printf("[wasm:%s] host_publish_event: unknown certainty %d, dropping", p.meta.ID, certainty)
	}
}

// hostLog is called by the guest via the "host_log" import and writes to the host's logger.
func (p *Plugin) hostLog(_ context.Context, m api.Module, _, ptr, length uint32) {
	data, _ := m.Memory().Read(ptr, length)
	log.Printf("[wasm:%s] %s", p.meta.ID, data)
}

// hostDBExec executes a SQL statement with JSON-encoded args ([]string).
// Writes the rows-affected int64 as JSON to outPtr. Returns bytes written, 0 on error.
func (p *Plugin) hostDBExec(_ context.Context, m api.Module, sqlPtr, sqlLen, argsPtr, argsLen, outPtr, outMax uint32) uint32 {
	if p.database == nil {
		return 0
	}
	sqlBytes, ok := m.Memory().Read(sqlPtr, sqlLen)
	if !ok {
		return 0
	}
	argsBytes, ok := m.Memory().Read(argsPtr, argsLen)
	if !ok {
		return 0
	}
	var args []any
	if err := json.Unmarshal(argsBytes, &args); err != nil {
		log.Printf("[wasm:%s] host_db_exec: parse args: %v", p.meta.ID, err)
		return 0
	}
	result, err := p.database.Conn().Exec(string(sqlBytes), args...)
	if err != nil {
		log.Printf("[wasm:%s] host_db_exec: %v", p.meta.ID, err)
		return 0
	}
	n, _ := result.RowsAffected()
	out, err := json.Marshal(n)
	if err != nil {
		return 0
	}
	return p.writeResult(m, out, outPtr, outMax)
}

// hostDBQuery executes a SQL query with JSON-encoded args and writes the result
// rows as a JSON array of column→value maps to outPtr.
func (p *Plugin) hostDBQuery(_ context.Context, m api.Module, sqlPtr, sqlLen, argsPtr, argsLen, outPtr, outMax uint32) uint32 {
	if p.database == nil {
		return 0
	}
	sqlBytes, ok := m.Memory().Read(sqlPtr, sqlLen)
	if !ok {
		return 0
	}
	argsBytes, ok := m.Memory().Read(argsPtr, argsLen)
	if !ok {
		return 0
	}
	var args []any
	if err := json.Unmarshal(argsBytes, &args); err != nil {
		log.Printf("[wasm:%s] host_db_query: parse args: %v", p.meta.ID, err)
		return 0
	}
	rows, err := p.database.Conn().Query(string(sqlBytes), args...)
	if err != nil {
		log.Printf("[wasm:%s] host_db_query: %v", p.meta.ID, err)
		return 0
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return 0
	}
	var result []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			log.Printf("[wasm:%s] host_db_query: scan row: %v", p.meta.ID, err)
			return 0
		}
		row := make(map[string]any, len(cols))
		for i, col := range cols {
			switch v := vals[i].(type) {
			case []byte:
				row[col] = string(v)
			case time.Time:
				row[col] = v.Format(time.RFC3339)
			default:
				row[col] = v
			}
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[wasm:%s] host_db_query: rows: %v", p.meta.ID, err)
		return 0
	}
	if result == nil {
		result = []map[string]any{}
	}
	out, err := json.Marshal(result)
	if err != nil {
		return 0
	}
	return p.writeResult(m, out, outPtr, outMax)
}

// hostHTTPFetch performs an outbound HTTP request.
// reqPtr/reqLen is a JSON object: {method, url, headers?, body?}.
// Writes a JSON-encoded HTTPFetchResult to outPtr.
func (p *Plugin) hostHTTPFetch(_ context.Context, m api.Module, reqPtr, reqLen, outPtr, outMax uint32) uint32 {
	reqBytes, ok := m.Memory().Read(reqPtr, reqLen)
	if !ok {
		return 0
	}
	var req sdk.HTTPFetchRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		log.Printf("[wasm:%s] host_http_fetch: parse req: %v", p.meta.ID, err)
		return 0
	}
	if req.Method == "" {
		req.Method = http.MethodGet
	}

	var bodyReader io.Reader
	if len(req.BodyBytes) > 0 {
		bodyReader = bytes.NewReader(req.BodyBytes)
	} else if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	}
	httpReq, err := http.NewRequest(req.Method, req.URL, bodyReader)
	if err != nil {
		return p.writeHTTPFetchError(m, "bad request: "+err.Error(), outPtr, outMax)
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return p.writeHTTPFetchError(m, err.Error(), outPtr, outMax)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, int64(outMax)+1)
	bodyBuilder, err := io.ReadAll(limited)
	if err != nil {
		return p.writeHTTPFetchError(m, "read body: "+err.Error(), outPtr, outMax)
	}
	if uint32(len(bodyBuilder)) > outMax {
		return p.writeHTTPFetchError(m, fmt.Sprintf("response body exceeds buffer (%d bytes)", outMax), outPtr, outMax)
	}

	respHeaders := make(map[string]string, len(resp.Header))
	for k := range resp.Header {
		respHeaders[k] = resp.Header.Get(k)
	}

	result := sdk.HTTPFetchResult{
		Status:  resp.StatusCode,
		Headers: respHeaders,
		Body:    string(bodyBuilder),
	}
	out, _ := json.Marshal(result)
	return p.writeResult(m, out, outPtr, outMax)
}

func (p *Plugin) writeHTTPFetchError(m api.Module, msg string, outPtr, outMax uint32) uint32 {
	result := sdk.HTTPFetchResult{Error: msg}
	out, _ := json.Marshal(result)
	return p.writeResult(m, out, outPtr, outMax)
}

// hostBroadcastWS sends a raw byte message to all connected WebSocket clients.
func (p *Plugin) hostBroadcastWS(_ context.Context, m api.Module, ptr, length uint32) {
	if p.hub == nil {
		return
	}
	data, ok := m.Memory().Read(ptr, length)
	if !ok {
		return
	}
	msg := make([]byte, len(data))
	copy(msg, data)
	p.hub.Broadcast(msg)
}

// publicConfigKeys are host config values any plugin may read. Everything
// else is per-plugin: a plugin can only read keys it declares in its own
// Settings metadata, so one plugin cannot read another's secrets (API keys)
// and exfiltrate them via host_http_fetch.
var publicConfigKeys = map[string]struct{}{
	"app_version": {},
	"data_dir":    {},
	"dev_mode":    {},
	"replay_dir":  {},
}

func (p *Plugin) configKeyAllowed(key string) bool {
	if _, ok := publicConfigKeys[key]; ok {
		return true
	}
	for _, s := range p.meta.Settings {
		if s.Key == key {
			return true
		}
	}
	return false
}

// hostGetConfig looks up a config key and writes the value string to outPtr.
// Only public keys and the plugin's own declared settings keys are served.
func (p *Plugin) hostGetConfig(_ context.Context, m api.Module, keyPtr, keyLen, outPtr, outMax uint32) uint32 {
	if p.cfg == nil {
		return 0
	}
	keyBytes, ok := m.Memory().Read(keyPtr, keyLen)
	if !ok {
		return 0
	}
	key := string(keyBytes)
	if !p.configKeyAllowed(key) {
		log.Printf("[wasm:%s] host_get_config: key %q not declared in plugin settings, denied", p.meta.ID, key)
		return 0
	}
	val := p.cfg.Lookup(key)
	return p.writeResult(m, []byte(val), outPtr, outMax)
}

// hostUploadFile streams a WASI-mounted file to a URL via multipart POST.
// The file is read directly by the host from disk — no file bytes pass through WASM memory.
// headers is a JSON-encoded map[string]string. fieldName is the multipart field name (e.g. "file").
// Writes a JSON-encoded HTTPFetchResult to outPtr. Returns bytes written, 0 on error.
func (p *Plugin) hostUploadFile(_ context.Context, m api.Module, pathPtr, pathLen, urlPtr, urlLen, headersPtr, headersLen, fieldNamePtr, fieldNameLen, outPtr, outMax uint32) uint32 {
	writeErr := func(msg string) uint32 {
		out, _ := json.Marshal(sdk.HTTPFetchResult{Error: msg})
		return p.writeResult(m, out, outPtr, outMax)
	}

	pathBytes, ok := m.Memory().Read(pathPtr, pathLen)
	if !ok {
		return writeErr("read path from guest memory failed")
	}
	urlBytes, ok := m.Memory().Read(urlPtr, urlLen)
	if !ok {
		return writeErr("read url from guest memory failed")
	}
	headersBytes, ok := m.Memory().Read(headersPtr, headersLen)
	if !ok {
		return writeErr("read headers from guest memory failed")
	}
	fieldBytes, ok := m.Memory().Read(fieldNamePtr, fieldNameLen)
	if !ok {
		return writeErr("read field name from guest memory failed")
	}

	fieldName := strings.TrimSpace(string(fieldBytes))
	if fieldName == "" {
		fieldName = "file"
	}

	var headers map[string]string
	if len(headersBytes) > 0 {
		if err := json.Unmarshal(headersBytes, &headers); err != nil {
			log.Printf("[wasm:%s] host_upload_file: parse headers: %v", p.meta.ID, err)
		}
	}

	realPath, err := p.resolveWASIPath(string(pathBytes))
	if err != nil {
		return writeErr(err.Error())
	}

	f, err := os.Open(realPath)
	if err != nil {
		return writeErr("open file: " + err.Error())
	}
	defer f.Close()

	// Pipe the multipart body so the file is never fully buffered in memory.
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		fw, werr := mw.CreateFormFile(fieldName, filepath.Base(realPath))
		if werr != nil {
			pw.CloseWithError(fmt.Errorf("create form file: %w", werr))
			return
		}
		if _, werr = io.Copy(fw, f); werr != nil {
			pw.CloseWithError(fmt.Errorf("copy file: %w", werr))
			return
		}
		pw.CloseWithError(mw.Close())
	}()

	req, err := http.NewRequest(http.MethodPost, string(urlBytes), pr)
	if err != nil {
		return writeErr("build request: " + err.Error())
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return writeErr(err.Error())
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, int64(outMax)+1))

	respHeaders := make(map[string]string, len(resp.Header))
	for k := range resp.Header {
		respHeaders[k] = resp.Header.Get(k)
	}
	result := sdk.HTTPFetchResult{
		Status:  resp.StatusCode,
		Headers: respHeaders,
		Body:    string(body),
	}
	out, _ := json.Marshal(result)
	return p.writeResult(m, out, outPtr, outMax)
}