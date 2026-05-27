//go:build wasip1

// ranks is an OOF_RL WASM plugin. Compile with:
//
//	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ranks.wasm .
package main

import sdk "github.com/erosas/oof-plugin-sdk"

//go:wasmexport plugin_metadata
func pluginMetadata(outPtr, outMax uint32) uint32 {
	meta := sdk.PluginMeta{
		ID:     "ranks",
		NavTab: sdk.NavTabMeta{ID: "ranks", Label: "Ranks", Order: 15},
		Routes: []sdk.RouteMeta{{Path: "/api/ranks/players", Method: "GET"}},
		Events: []string{"state.updated", "match.destroyed"},
	}
	return sdk.WriteMetadata(meta, outPtr, outMax)
}

//go:wasmexport plugin_init
func pluginInit(cfgPtr, cfgLen uint32) uint32 { return 0 }

//go:wasmexport plugin_on_event
func pluginOnEvent(typePtr, typeLen, payloadPtr, payloadLen uint32) {
	sdk.HandleEventExport(typePtr, typeLen, payloadPtr, payloadLen, onEvent)
}

//go:wasmexport plugin_handle_http
func pluginHandleHTTP(reqPtr, reqLen, outPtr, outMax uint32) uint32 {
	return sdk.HandleHTTPExport(reqPtr, reqLen, outPtr, outMax, func(_ sdk.HTTPRequest) sdk.HTTPResponse {
		return handleHTTP()
	})
}

//go:wasmexport plugin_shutdown
func pluginShutdown() {}

//go:wasmexport malloc
func malloc(size uint32) uint32 { return sdk.Malloc(size) }

//go:wasmexport free
func free(ptr, size uint32) { sdk.Free(ptr, size) }

func main() {}
