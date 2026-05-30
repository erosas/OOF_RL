//go:build wasip1

package sdk

import (
	"encoding/json"
	"unsafe"
)

// host_log is exported by the host into every plugin module.
//
//go:wasmimport env host_log
func hostLog(level uint32, ptr uint32, length uint32)

// host_publish_event is provided by the host and pushes an event onto the bus.
//
//go:wasmimport env host_publish_event
func hostPublishEvent(certainty, typePtr, typeLen, payloadPtr, payloadLen uint32)

// host_db_exec executes a SQL statement (INSERT/UPDATE/DELETE) with JSON-encoded args.
// Writes the rows-affected int64 as JSON to outPtr. Returns bytes written, or 0 on error.
//
//go:wasmimport env host_db_exec
func hostDBExec(sqlPtr, sqlLen, argsPtr, argsLen, outPtr, outMax uint32) uint32

// host_db_query executes a SQL query and writes the JSON result rows to outPtr.
// Returns the number of bytes written, or 0 on error.
//
//go:wasmimport env host_db_query
func hostDBQuery(sqlPtr, sqlLen, argsPtr, argsLen, outPtr, outMax uint32) uint32

// host_http_fetch performs an outbound HTTP request from a JSON-encoded HTTPFetchRequest.
// Writes a JSON-encoded HTTPFetchResult to outPtr. Returns bytes written, 0 on error.
//
//go:wasmimport env host_http_fetch
func hostHTTPFetch(reqPtr, reqLen, outPtr, outMax uint32) uint32

// host_broadcast_ws sends a raw byte message to all connected WebSocket clients.
//
//go:wasmimport env host_broadcast_ws
func hostBroadcastWS(ptr, length uint32)

// host_get_config looks up a config key (by ptr/len) and writes the value string to outPtr.
// Returns bytes written, or 0 if the key is unknown.
//
//go:wasmimport env host_get_config
func hostGetConfig(keyPtr, keyLen, outPtr, outMax uint32) uint32

// host_upload_file streams a WASI-mounted file to a URL via multipart POST.
// The file is read by the host directly from disk — no file bytes pass through WASM memory.
// headers is a JSON-encoded map[string]string. fieldName is the multipart field name.
// Writes a JSON-encoded HTTPFetchResult to outPtr. Returns bytes written, 0 on error.
//
//go:wasmimport env host_upload_file
func hostUploadFile(pathPtr, pathLen, urlPtr, urlLen, headersPtr, headersLen, fieldNamePtr, fieldNameLen, outPtr, outMax uint32) uint32

// Output buffer sizes for each host call.
const (
	dbExecBufSize    = 32         // int64 JSON is at most 20 chars
	dbQueryBufSize   = 256 * 1024 // 256 KB
	httpFetchBufSize = 4 * 1024 * 1024 // 4 MB — external APIs can return large payloads
	getConfigBufSize = 4096
	uploadFileBufSize = 64 * 1024 // 64 KB — upload responses are always small JSON
)

// readResult calls the given host function with a fresh output buffer of outSize
// bytes and returns a copy of the bytes written. Returns nil if the host writes nothing.
func readResult(call func(outPtr, outMax uint32) uint32, outSize uint32) []byte {
	outBuf := make([]byte, outSize)
	n := call(ptrOf(outBuf), outSize)
	if n == 0 {
		return nil
	}
	out := make([]byte, n)
	copy(out, outBuf[:n])
	return out
}

// Log writes msg to the host's logger.
func Log(msg string) {
	b := []byte(msg)
	if len(b) == 0 {
		return
	}
	hostLog(1, ptrOf(b), uint32(len(b)))
}

// ReadBytes returns a slice backed by guest linear memory at [ptr, ptr+length).
// The caller must not retain it past the current function call — the host may
// free the underlying allocation immediately after the export returns.
func ReadBytes(ptr, length uint32) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), length)
}

// WriteOutput copies data into the pre-allocated guest buffer at outPtr.
// Returns the number of bytes written (min of len(data) and maxLen).
func WriteOutput(data []byte, outPtr, maxLen uint32) uint32 {
	n := uint32(len(data))
	if n > maxLen {
		n = maxLen
	}
	out := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(outPtr))), n)
	copy(out, data)
	return n
}

// WriteJSONOutput marshals v to JSON and writes it into the output buffer.
func WriteJSONOutput(v any, outPtr, outMax uint32) uint32 {
	b, _ := json.Marshal(v)
	return WriteOutput(b, outPtr, outMax)
}

// WriteMetadata writes PluginMeta JSON into the output buffer.
func WriteMetadata(meta PluginMeta, outPtr, outMax uint32) uint32 {
	return WriteJSONOutput(meta, outPtr, outMax)
}

// HandleHTTPExport decodes HTTPRequest from guest memory, invokes handler, and
// returns a packed uint64 (hi32=ptr, lo32=len) pointing to a plugin-owned buffer
// containing the JSON-encoded HTTPResponse. The host must call free(ptr, len)
// after reading the response.
func HandleHTTPExport(reqPtr, reqLen uint32, handler func(HTTPRequest) HTTPResponse) uint64 {
	var req HTTPRequest
	if err := json.Unmarshal(ReadBytes(reqPtr, reqLen), &req); err != nil {
		return packResponse(HTTPResponse{Status: 500, Body: `{"error":"bad request"}`})
	}
	return packResponse(handler(req))
}

// packResponse JSON-encodes resp into a Malloc'd buffer and returns a packed
// uint64 (hi32=ptr, lo32=len) that the host can read and then free.
func packResponse(resp HTTPResponse) uint64 {
	b, _ := json.Marshal(resp)
	size := uint32(len(b))
	ptr := Malloc(size)
	WriteOutput(b, ptr, size)
	return uint64(ptr)<<32 | uint64(size)
}

// HandleEventExport decodes plugin_on_event ABI pointers and forwards them to handler.
func HandleEventExport(typePtr, typeLen, payloadPtr, payloadLen uint32, handler func(eventType string, payload []byte)) {
	handler(string(ReadBytes(typePtr, typeLen)), ReadBytes(payloadPtr, payloadLen))
}

// keep prevents the GC from collecting malloc'd slices whose pointers
// have been handed to the host.
var keep = map[uint32][]byte{}

// Malloc allocates size bytes in guest linear memory and returns the pointer.
// The actual WASM export "malloc" lives in package main as a shim to this function.
func Malloc(size uint32) uint32 {
	if size == 0 {
		return 0
	}
	b := make([]byte, size)
	ptr := ptrOf(b)
	keep[ptr] = b
	return ptr
}

// Free releases a previously malloc'd allocation.
func Free(ptr, _ uint32) {
	delete(keep, ptr)
}

// PublishEvent publishes an event back into the host's event bus.
// eventType should not shadow a native OOF_RL event type string.
// payload must be valid JSON.
func PublishEvent(c Certainty, eventType string, payload []byte) {
	tb := []byte(eventType)
	if len(tb) == 0 || len(payload) == 0 {
		return
	}
	hostPublishEvent(uint32(c), ptrOf(tb), uint32(len(tb)), ptrOf(payload), uint32(len(payload)))
}

// DBExec executes a SQL statement with the given args and returns rows affected.
// args is passed as a JSON array of strings. Returns -1 on error.
func DBExec(sql string, args []string) int64 {
	if len(sql) == 0 {
		return -1
	}
	sqlB, argsJSON := []byte(sql), encodeArgs(args)
	data := readResult(func(out, max uint32) uint32 {
		return hostDBExec(ptrOf(sqlB), uint32(len(sqlB)), ptrOf(argsJSON), uint32(len(argsJSON)), out, max)
	}, dbExecBufSize)
	if data == nil {
		return -1
	}
	var n int64
	if err := json.Unmarshal(data, &n); err != nil {
		return -1
	}
	return n
}

// DBQuery executes a SQL query with the given args and returns the result rows
// as a slice of maps (column→value). Returns nil on error.
func DBQuery(sql string, args []string) []map[string]any {
	if len(sql) == 0 {
		return nil
	}
	sqlB, argsJSON := []byte(sql), encodeArgs(args)
	data := readResult(func(out, max uint32) uint32 {
		return hostDBQuery(ptrOf(sqlB), uint32(len(sqlB)), ptrOf(argsJSON), uint32(len(argsJSON)), out, max)
	}, dbQueryBufSize)
	if data == nil {
		return nil
	}
	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil
	}
	return rows
}

// HTTPFetch performs an outbound HTTP request and returns the result.
// On network or serialization error, the returned HTTPFetchResult.Error is non-empty.
func HTTPFetch(req HTTPFetchRequest) HTTPFetchResult {
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return HTTPFetchResult{Error: "marshal request: " + err.Error()}
	}
	data := readResult(func(out, max uint32) uint32 {
		return hostHTTPFetch(ptrOf(reqJSON), uint32(len(reqJSON)), out, max)
	}, httpFetchBufSize)
	if data == nil {
		return HTTPFetchResult{Error: "fetch failed"}
	}
	var result HTTPFetchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return HTTPFetchResult{Error: "unmarshal response: " + err.Error()}
	}
	return result
}

// BroadcastWS sends msg to all connected WebSocket clients.
func BroadcastWS(msg []byte) {
	if len(msg) == 0 {
		return
	}
	hostBroadcastWS(ptrOf(msg), uint32(len(msg)))
}

// UploadFile streams a WASI-mounted file to url via multipart POST.
// The host reads the file directly from disk — no file bytes pass through WASM memory.
// fieldName is the multipart field name (e.g. "file"). headers are injected as-is.
func UploadFile(wasiPath, url, fieldName string, headers map[string]string) HTTPFetchResult {
	pathB := []byte(wasiPath)
	urlB := []byte(url)
	fieldB := []byte(fieldName)
	headersJSON, _ := json.Marshal(headers)
	data := readResult(func(out, max uint32) uint32 {
		return hostUploadFile(
			ptrOf(pathB), uint32(len(pathB)),
			ptrOf(urlB), uint32(len(urlB)),
			ptrOf(headersJSON), uint32(len(headersJSON)),
			ptrOf(fieldB), uint32(len(fieldB)),
			out, max,
		)
	}, uploadFileBufSize)
	if data == nil {
		return HTTPFetchResult{Error: "upload failed"}
	}
	var result HTTPFetchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return HTTPFetchResult{Error: "unmarshal response: " + err.Error()}
	}
	return result
}

// GetConfig returns the value of a config key, or "" if unknown or empty.
func GetConfig(key string) string {
	if len(key) == 0 {
		return ""
	}
	kb := []byte(key)
	data := readResult(func(out, max uint32) uint32 {
		return hostGetConfig(ptrOf(kb), uint32(len(kb)), out, max)
	}, getConfigBufSize)
	return string(data)
}

func encodeArgs(args []string) []byte {
	if len(args) == 0 {
		return []byte("[]")
	}
	b, _ := json.Marshal(args)
	return b
}

func ptrOf(b []byte) uint32 {
	return uint32(uintptr(unsafe.Pointer(&b[0])))
}
