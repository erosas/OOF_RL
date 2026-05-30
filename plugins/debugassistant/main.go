//go:build wasip1

// debugassistant is an OOF_RL WASM plugin. Compile with:
//
//	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o debugassistant.wasm .
//
// After compilation, copy view.html and view.js into a debugassistant/ subdirectory
// next to the .wasm file so the host can serve them as plugin assets.
package main

import sdk "github.com/erosas/oof-plugin-sdk"

//go:wasmexport plugin_metadata
func pluginMetadata(outPtr, outMax uint32) uint32 {
	meta := sdk.PluginMeta{
		ID:     "debugassistant",
		NavTab: sdk.NavTabMeta{ID: "debugassistant", Label: "Debug", Order: 90},
		Routes: []sdk.RouteMeta{
			{Path: "/api/debug-assistant/events", Method: "GET"},
			{Path: "/api/debug-assistant/context", Method: "GET"},
			{Path: "/api/debug-assistant/screenshots", Method: "GET"},
			{Path: "/api/debug-assistant/export-report", Method: "POST"},
			{Path: "/api/debug-assistant/reset", Method: "POST"},
		},
		Events: []string{
			"match.started",
			"state.updated",
			"goal.scored",
			"stat.feed",
			"clock.updated",
			"match.ended",
			"match.destroyed",
		},
	}
	return sdk.WriteMetadata(meta, outPtr, outMax)
}

//go:wasmexport plugin_init
func pluginInit(_, _ uint32) uint32 {
	return initPlugin()
}

//go:wasmexport plugin_on_event
func pluginOnEvent(typePtr, typeLen, payloadPtr, payloadLen uint32) {
	onEvent(
		string(sdk.ReadBytes(typePtr, typeLen)),
		sdk.ReadBytes(payloadPtr, payloadLen),
	)
}

//go:wasmexport plugin_handle_http
func pluginHandleHTTP(reqPtr, reqLen uint32) uint64 {
	return sdk.HandleHTTPExport(reqPtr, reqLen, handleHTTP)
}

//go:wasmexport plugin_shutdown
func pluginShutdown() {}

//go:wasmexport malloc
func malloc(size uint32) uint32 { return sdk.Malloc(size) }

//go:wasmexport free
func free(ptr, size uint32) { sdk.Free(ptr, size) }

func main() {}