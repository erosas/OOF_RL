//go:build wasip1

// autoupdate is an OOF_RL WASM plugin. Compile with:
//
//	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o autoupdate.wasm .
package main

import sdk "github.com/erosas/oof-plugin-sdk"

//go:wasmexport plugin_metadata
func pluginMetadata(outPtr, outMax uint32) uint32 {
	meta := sdk.PluginMeta{
		ID:     "autoupdate",
		NavTab: sdk.NavTabMeta{Label: "Auto Update"},
		Routes: []sdk.RouteMeta{
			{Path: "/api/autoupdate/status", Method: "GET"},
			{Path: "/api/autoupdate/check", Method: "POST"},
			{Path: "/api/autoupdate/download", Method: "POST"},
		},
		Settings: []sdk.SettingSchema{
			{
				Key:          "autoupdate_check_url",
				Label:        "Check for updates",
				Description:  "Checks the unsigned GitHub release manifest and verifies downloaded zip files with SHA256. Milestone 1 does not install updates.",
				Type:         "action",
				Default:      defaultManifestURL,
				Placeholder:  "Check",
				ActionPath:   "/api/autoupdate/check",
				ActionMethod: "POST",
				StatusPath:   "/api/autoupdate/status",
				DownloadPath: "/api/autoupdate/download",
			},
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
