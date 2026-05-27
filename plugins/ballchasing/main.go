//go:build wasip1

// ballchasing is an OOF_RL WASM plugin. Compile with:
//
//	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o ballchasing.wasm .
package main

import sdk "github.com/erosas/oof-plugin-sdk"

//go:wasmexport plugin_metadata
func pluginMetadata(outPtr, outMax uint32) uint32 {
	meta := sdk.PluginMeta{
		ID:     "ballchasing",
		NavTab: sdk.NavTabMeta{ID: "bc", Label: "Ballchasing", Order: 40},
		Routes: []sdk.RouteMeta{
			{Path: "/api/ballchasing/ping", Method: "GET"},
			{Path: "/api/ballchasing/local-replays/purge", Method: "POST"},
			{Path: "/api/ballchasing/matches", Method: "GET"},
			{Path: "/api/ballchasing/sync", Method: "POST"},
			{Path: "/api/ballchasing/replays", Method: "GET"},
			{Path: "/api/ballchasing/groups", Method: "GET"},
			{Path: "/api/ballchasing/upload", Method: "POST"},
		},
		Events: []string{"match.ended"},
		Settings: []sdk.SettingSchema{
			{Key: "ballchasing_api_key", Label: "API key", Description: "Your Ballchasing.com API key. Get one at ballchasing.com/upload.", Secret: true},
			{Key: "ballchasing_delete_after_upload", Label: "Delete after upload", Description: "Automatically delete the local .replay file after successful upload.", Type: "checkbox"},
		},
	}
	return sdk.WriteMetadata(meta, outPtr, outMax)
}

//go:wasmexport plugin_init
func pluginInit(cfgPtr, cfgLen uint32) uint32 {
	return initPlugin()
}

//go:wasmexport plugin_apply_settings
func pluginApplySettings(cfgPtr, cfgLen uint32) uint32 {
	data := sdk.ReadBytes(cfgPtr, cfgLen)
	return applySettings(data)
}

//go:wasmexport plugin_on_event
func pluginOnEvent(typePtr, typeLen, payloadPtr, payloadLen uint32) {
	sdk.HandleEventExport(typePtr, typeLen, payloadPtr, payloadLen, onEvent)
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
