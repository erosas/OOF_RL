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
// Returns rows affected as int64 encoded in result[0], or -1 on error.
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

// keep is a global map that prevents the GC from collecting malloc'd slices
// whose pointers have been handed to the host.
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
	argsJSON := encodeArgs(args)
	outBuf := make([]byte, 32)
	out := ptrOf(outBuf)
	sqlB := []byte(sql)
	n := hostDBExec(ptrOf(sqlB), uint32(len(sqlB)), ptrOf(argsJSON), uint32(len(argsJSON)), out, uint32(len(outBuf)))
	if n == 0 {
		return -1
	}
	data, ok := readMem(out, n)
	if !ok {
		return -1
	}
	var result int64
	if err := json.Unmarshal(data, &result); err != nil {
		return -1
	}
	return result
}

// DBQuery executes a SQL query with the given args and returns the result rows
// as a slice of maps (column→value). Returns nil on error.
func DBQuery(sql string, args []string) []map[string]any {
	argsJSON := encodeArgs(args)
	outBuf := make([]byte, 256*1024) // 256 KB max result
	out := ptrOf(outBuf)
	sqlB := []byte(sql)
	n := hostDBQuery(ptrOf(sqlB), uint32(len(sqlB)), ptrOf(argsJSON), uint32(len(argsJSON)), out, uint32(len(outBuf)))
	if n == 0 {
		return nil
	}
	data, ok := readMem(out, n)
	if !ok {
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
	outBuf := make([]byte, 256*1024)
	out := ptrOf(outBuf)
	n := hostHTTPFetch(ptrOf(reqJSON), uint32(len(reqJSON)), out, uint32(len(outBuf)))
	if n == 0 {
		return HTTPFetchResult{Error: "fetch failed"}
	}
	data, ok := readMem(out, n)
	if !ok {
		return HTTPFetchResult{Error: "memory read failed"}
	}
	var result HTTPFetchResult
	json.Unmarshal(data, &result)
	return result
}

// BroadcastWS sends msg to all connected WebSocket clients.
func BroadcastWS(msg []byte) {
	if len(msg) == 0 {
		return
	}
	hostBroadcastWS(ptrOf(msg), uint32(len(msg)))
}

// GetConfig returns the value of a config key, or "" if unknown.
func GetConfig(key string) string {
	outBuf := make([]byte, 4096)
	out := ptrOf(outBuf)
	kb := []byte(key)
	n := hostGetConfig(ptrOf(kb), uint32(len(kb)), out, uint32(len(outBuf)))
	if n == 0 {
		return ""
	}
	data, ok := readMem(out, n)
	if !ok {
		return ""
	}
	return string(data)
}


// encodeArgs marshals a string slice as a compact JSON array.
func encodeArgs(args []string) []byte {
	if len(args) == 0 {
		return []byte("[]")
	}
	b, _ := json.Marshal(args)
	return b
}

func readMem(ptr, n uint32) ([]byte, bool) {
	if n == 0 {
		return nil, false
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), n), true
}

func ptrOf(b []byte) uint32 {
	return uint32(uintptr(unsafe.Pointer(&b[0])))
}
