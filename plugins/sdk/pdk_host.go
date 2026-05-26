//go:build !wasip1

package sdk

// Log is a no-op outside of WASM; plugins running in a host process use the host logger directly.
func Log(_ string) {}

func DBExec(_ string, _ []string) int64             { return -1 }
func DBQuery(_ string, _ []string) []map[string]any { return nil }
func HTTPFetch(_ HTTPFetchRequest) HTTPFetchResult  { return HTTPFetchResult{} }
func BroadcastWS(_ []byte)                          {}
func GetConfig(_ string) string { return "" }
