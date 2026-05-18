//go:build !wasip1

package sdk

// Log is a no-op outside of WASM; plugins running in a host process use the host logger directly.
func Log(_ string) {}