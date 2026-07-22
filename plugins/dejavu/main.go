//go:build wasip1

// dejavu is an OOF_RL WASM plugin. Compile with:
//
//	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o dejavu.wasm .
package main

import sdk "github.com/erosas/oof-plugin-sdk"

//go:wasmexport plugin_metadata
func pluginMetadata(outPtr, outMax uint32) uint32 {
	meta := sdk.PluginMeta{
		ID:     "dejavu",
		NavTab: sdk.NavTabMeta{ID: "dejavu", Label: "Deja Vu", Order: 20},
		Routes: []sdk.RouteMeta{
			{Path: "/api/dejavu/recall", Method: "GET"},
		},
		Events: []string{"state.updated", "match.ended", "match.destroyed"},
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
