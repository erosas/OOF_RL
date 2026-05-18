//go:build wasip1

package sdk

import "unsafe"

// host_log is exported by the host into every plugin module.
//
//go:wasmimport env host_log
func hostLog(level uint32, ptr uint32, length uint32)

//go:wasmimport env host_publish_event
func hostPublishEvent(certainty, typePtr, typeLen, payloadPtr, payloadLen uint32)

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

func ptrOf(b []byte) uint32 {
	return uint32(uintptr(unsafe.Pointer(&b[0])))
}