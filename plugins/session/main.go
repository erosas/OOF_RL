//go:build wasip1

// session is an OOF_RL WASM plugin. Compile with:
//
//	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o session.wasm .
package main

import sdk "github.com/erosas/oof-plugin-sdk"

//go:wasmexport plugin_metadata
func pluginMetadata(outPtr, outMax uint32) uint32 {
	meta := sdk.PluginMeta{
		ID:       "session",
		NavTab:   sdk.NavTabMeta{ID: "session", Label: "Session", Order: 25},
		Routes: []sdk.RouteMeta{
			{Path: "/api/session/stats", Method: "GET"},
			{Path: "/api/session/start"},
			{Path: "/api/session/new", Method: "POST"},
			{Path: "/api/session/suggest-player", Method: "GET"},
			{Path: "/api/session/history/"},
			{Path: "/api/session/history", Method: "GET"},
		},
		Events: []string{"match.started"},
	}
	return sdk.WriteMetadata(meta, outPtr, outMax)
}

//go:wasmexport plugin_init
func pluginInit(cfgPtr, cfgLen uint32) uint32 {
	return initPlugin()
}

//go:wasmexport plugin_on_event
func pluginOnEvent(typePtr, typeLen, payloadPtr, payloadLen uint32) {
	onEvent(string(sdk.ReadBytes(typePtr, typeLen)))
}

//go:wasmexport plugin_handle_http
func pluginHandleHTTP(reqPtr, reqLen, outPtr, outMax uint32) uint32 {
	return sdk.HandleHTTPExport(reqPtr, reqLen, outPtr, outMax, handleHTTP)
}

//go:wasmexport plugin_shutdown
func pluginShutdown() {}

//go:wasmexport malloc
func malloc(size uint32) uint32 { return sdk.Malloc(size) }

//go:wasmexport free
func free(ptr, size uint32) { sdk.Free(ptr, size) }

func main() {}