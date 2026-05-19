//go:build wasip1

// session is an OOF_RL WASM plugin. Compile with:
//
//	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o session.wasm .
package main

import (
	"encoding/json"

	sdk "github.com/erosas/oof-plugin-sdk"
)

//go:wasmexport plugin_metadata
func pluginMetadata(outPtr, outMax uint32) uint32 {
	meta := sdk.PluginMeta{
		ID:       "session",
		NavTab:   sdk.NavTabMeta{ID: "session", Label: "Session", Order: 25},
		Requires: []string{"history"},
		Routes: []string{
			"/api/session/stats",
			"/api/session/start",
			"/api/session/new",
			"/api/session/suggest-player",
			"/api/session/history/",
			"/api/session/history",
		},
		Events: []string{"match.started"},
	}
	b, _ := json.Marshal(meta)
	return sdk.WriteOutput(b, outPtr, outMax)
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
	var req sdk.HTTPRequest
	if err := json.Unmarshal(sdk.ReadBytes(reqPtr, reqLen), &req); err != nil {
		resp := sdk.HTTPResponse{Status: 500, Body: `{"error":"bad request"}`}
		b, _ := json.Marshal(resp)
		return sdk.WriteOutput(b, outPtr, outMax)
	}
	resp := handleHTTP(req)
	b, _ := json.Marshal(resp)
	return sdk.WriteOutput(b, outPtr, outMax)
}

//go:wasmexport plugin_shutdown
func pluginShutdown() {}

//go:wasmexport malloc
func malloc(size uint32) uint32 { return sdk.Malloc(size) }

//go:wasmexport free
func free(ptr, size uint32) { sdk.Free(ptr, size) }

func main() {}