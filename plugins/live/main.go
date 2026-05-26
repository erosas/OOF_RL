//go:build wasip1

// live is an OOF_RL WASM plugin. Compile with:
//
//	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o live.wasm .
package main

import (
	"encoding/json"

	sdk "github.com/erosas/oof-plugin-sdk"
)

//go:wasmexport plugin_metadata
func pluginMetadata(outPtr, outMax uint32) uint32 {
	meta := sdk.PluginMeta{
		ID:     "live",
		NavTab: sdk.NavTabMeta{ID: "live", Label: "Live", Order: 10},
		Routes: []sdk.RouteMeta{{Path: "/api/live/state", Method: "GET"}},
		Events: []string{"state.updated", "match.destroyed"},
	}
	b, _ := json.Marshal(meta)
	return sdk.WriteOutput(b, outPtr, outMax)
}

//go:wasmexport plugin_init
func pluginInit(cfgPtr, cfgLen uint32) uint32 { return 0 }

//go:wasmexport plugin_on_event
func pluginOnEvent(typePtr, typeLen, payloadPtr, payloadLen uint32) {
	onEvent(
		string(sdk.ReadBytes(typePtr, typeLen)),
		sdk.ReadBytes(payloadPtr, payloadLen),
	)
}

//go:wasmexport plugin_handle_http
func pluginHandleHTTP(reqPtr, reqLen, outPtr, outMax uint32) uint32 {
	resp := handleHTTP()
	b, _ := json.Marshal(resp)
	return sdk.WriteOutput(b, outPtr, outMax)
}

//go:wasmexport plugin_shutdown
func pluginShutdown() {}

// malloc and free are exported so the host can allocate/free guest memory for
// passing data in and out of the plugin. The GC-protection map lives in the SDK.
//
//go:wasmexport malloc
func malloc(size uint32) uint32 { return sdk.Malloc(size) }

//go:wasmexport free
func free(ptr, size uint32) { sdk.Free(ptr, size) }

func main() {}