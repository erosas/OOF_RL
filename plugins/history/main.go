//go:build wasip1

// history is an OOF_RL WASM plugin. Compile with:
//
//	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o history.wasm .
//
// This plugin owns only the History nav tab and its frontend assets.
// All event recording and REST routes for history are handled by the host's
// internal/histstore package, which runs independently of this plugin.
package main

import sdk "github.com/erosas/oof-plugin-sdk"

//go:wasmexport plugin_metadata
func pluginMetadata(outPtr, outMax uint32) uint32 {
	meta := sdk.PluginMeta{
		ID:     "history",
		NavTab: sdk.NavTabMeta{ID: "history", Label: "History", Order: 20},
	}
	return sdk.WriteMetadata(meta, outPtr, outMax)
}

//go:wasmexport plugin_init
func pluginInit(cfgPtr, cfgLen uint32) uint32 { return 0 }

//go:wasmexport plugin_on_event
func pluginOnEvent(typePtr, typeLen, payloadPtr, payloadLen uint32) {}

//go:wasmexport plugin_handle_http
func pluginHandleHTTP(reqPtr, reqLen, outPtr, outMax uint32) uint32 {
	return sdk.HandleHTTPExport(reqPtr, reqLen, outPtr, outMax, func(_ sdk.HTTPRequest) sdk.HTTPResponse {
		return sdk.HTTPResponse{Status: 404, Body: `{"error":"not found"}`}
	})
}

//go:wasmexport plugin_shutdown
func pluginShutdown() {}

//go:wasmexport malloc
func malloc(size uint32) uint32 { return sdk.Malloc(size) }

//go:wasmexport free
func free(ptr, size uint32) { sdk.Free(ptr, size) }

func main() {}