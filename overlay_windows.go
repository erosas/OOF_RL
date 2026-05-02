//go:build windows

package main

import (
	"time"

	webview2 "github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"

	"OOF_RL/internal/config"
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	kernel32             = windows.NewLazySystemDLL("kernel32.dll")
	procSetWindowLong    = user32.NewProc("SetWindowLongPtrW")
	procSetWindowPos     = user32.NewProc("SetWindowPos")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
	procShowWindow       = user32.NewProc("ShowWindow")
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
	procFreeConsole      = kernel32.NewProc("FreeConsole")
)

const (
	gwlStyle       = uintptr(0xFFFFFFF0) // GWL_STYLE   = -16
	gwlExStyle     = uintptr(0xFFFFFFEC) // GWL_EXSTYLE = -20
	wsPopup        = uintptr(0x80000000)
	wsVisible      = uintptr(0x10000000)
	wsExTopmost    = uintptr(0x00000008)
	wsExToolWindow = uintptr(0x00000080)
	wsExNoActivate = uintptr(0x08000000)
	swpFrameChange = uintptr(0x0020)
	swHide         = uintptr(0) // SW_HIDE
	swShowNA       = uintptr(8) // SW_SHOWNA — show without stealing focus

	overlayW = 860
	overlayH = 620
)

// HWND_TOPMOST = -1 cast to uintptr
var hwndTopmost = ^uintptr(0)

// vkMap maps user-facing key names (stored in config) to Windows virtual-key codes.
// Only keys unlikely to conflict with in-game bindings are included.
var vkMap = map[string]uintptr{
	"F1": 0x70, "F2": 0x71, "F3": 0x72, "F4": 0x73,
	"F5": 0x74, "F6": 0x75, "F7": 0x76, "F8": 0x77,
	"F9": 0x78, "F10": 0x79, "F11": 0x7A, "F12": 0x7B,
	"Insert":     0x2D,
	"Delete":     0x2E,
	"Home":       0x24,
	"End":        0x23,
	"PageUp":     0x21,
	"PageDown":   0x22,
	"Pause":      0x13,
	"ScrollLock": 0x91,
}

// startOverlay creates a hidden, borderless, always-on-top WebView2 window
// that loads the main dashboard URL. The returned WebView must be kept alive
// (defer Destroy) by the caller; it is driven by the main window's Run() loop
// because both windows share the same OS thread and Win32 message pump.
// Returns nil if WebView2 is unavailable.
func startOverlay(url string, cfg *config.Config) webview2.WebView {
	ov := webview2.NewWithOptions(webview2.WebViewOptions{Debug: false})
	if ov == nil {
		return nil
	}
	ov.SetTitle("OOF Overlay")
	ov.SetSize(overlayW, overlayH, webview2.HintFixed)
	ov.Navigate(url)

	hwnd := windows.HWND(uintptr(ov.Window()))
	configureOverlayWindow(hwnd)
	go listenOverlayHotkey(hwnd, cfg)
	return ov
}

func configureOverlayWindow(hwnd windows.HWND) {
	// Borderless popup — no title bar, no resize handles
	procSetWindowLong.Call(uintptr(hwnd), gwlStyle, wsPopup|wsVisible)
	// Always-on-top + not in taskbar/alt-tab + never steal game focus
	procSetWindowLong.Call(uintptr(hwnd), gwlExStyle, wsExTopmost|wsExToolWindow|wsExNoActivate)

	// Center on primary monitor
	screenW, _, _ := procGetSystemMetrics.Call(0) // SM_CXSCREEN
	screenH, _, _ := procGetSystemMetrics.Call(1) // SM_CYSCREEN
	x := (int(screenW) - overlayW) / 2
	y := (int(screenH) - overlayH) / 2

	procSetWindowPos.Call(
		uintptr(hwnd),
		hwndTopmost,
		uintptr(x), uintptr(y),
		uintptr(overlayW), uintptr(overlayH),
		swpFrameChange, // no SWP_SHOWWINDOW — stays hidden until hotkey
	)

	// Start hidden; the configured hotkey reveals it
	procShowWindow.Call(uintptr(hwnd), swHide)
}

// listenOverlayHotkey polls GetAsyncKeyState and toggles overlay visibility.
// It reads cfg.OverlayHotkey on every tick so hotkey changes saved in Settings
// take effect immediately without a restart.
func listenOverlayHotkey(hwnd windows.HWND, cfg *config.Config) {
	var prev bool
	visible := false
	for range time.Tick(50 * time.Millisecond) {
		key := cfg.OverlayHotkey
		vk, ok := vkMap[key]
		if !ok {
			vk = vkMap["F9"]
		}
		state, _, _ := procGetAsyncKeyState.Call(vk)
		curr := state&0x8000 != 0
		if curr && !prev { // rising edge — key just pressed
			if visible {
				procShowWindow.Call(uintptr(hwnd), swHide)
			} else {
				procShowWindow.Call(uintptr(hwnd), swShowNA)
			}
			visible = !visible
		}
		prev = curr
	}
}

// freeConsole detaches from the inherited console window so double-clicking
// the exe doesn't leave a terminal open behind the game.
func freeConsole() {
	procFreeConsole.Call()
}