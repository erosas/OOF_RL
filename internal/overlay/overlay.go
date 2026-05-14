//go:build windows

package overlay

import (
	"time"
	"unsafe"

	webview2 "github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"

	"OOF_RL/internal/config"
)

var (
	user32                   = windows.NewLazySystemDLL("user32.dll")
	kernel32                 = windows.NewLazySystemDLL("kernel32.dll")
	procSetWindowLong        = user32.NewProc("SetWindowLongPtrW")
	procSetWindowPos         = user32.NewProc("SetWindowPos")
	procGetSystemMetrics     = user32.NewProc("GetSystemMetrics")
	procShowWindow           = user32.NewProc("ShowWindow")
	procGetAsyncKeyState     = user32.NewProc("GetAsyncKeyState")
	procSendMessage          = user32.NewProc("SendMessageW")
	procReleaseCapture       = user32.NewProc("ReleaseCapture")
	procGetWindowRect        = user32.NewProc("GetWindowRect")
	procSetLayeredWindowAttr = user32.NewProc("SetLayeredWindowAttributes")
	procSetClassLongPtr      = user32.NewProc("SetClassLongPtrW")
	procFreeConsole          = kernel32.NewProc("FreeConsole")
	procLoadImage            = user32.NewProc("LoadImageW")
	procGetModuleHandle      = kernel32.NewProc("GetModuleHandleW")
)

const (
	wmSetIcon   = uintptr(0x0080)     // WM_SETICON
	iconSmall   = uintptr(0)          // ICON_SMALL
	iconBig     = uintptr(1)          // ICON_BIG
	imageIcon   = uintptr(1)          // IMAGE_ICON
	lrShared    = uintptr(0x8000)     // LR_SHARED
	gclpHIcon   = uintptr(0xFFFFFFF2) // GCLP_HICON   = -14
	gclpHIconSm = uintptr(0xFFFFFFDE) // GCLP_HICONSM = -34

	// SetWindowPos flags for forcing non-client area repaint without moving/resizing
	swpNoMove   = uintptr(0x0002)
	swpNoSize   = uintptr(0x0001)
	swpNoZOrder = uintptr(0x0004)

	gwlStyle        = uintptr(0xFFFFFFF0) // GWL_STYLE   = -16
	gwlExStyle      = uintptr(0xFFFFFFEC) // GWL_EXSTYLE = -20
	wsPopup         = uintptr(0x80000000)
	wsVisible       = uintptr(0x10000000)
	wsExTopmost     = uintptr(0x00000008)
	wsExToolWindow  = uintptr(0x00000080)
	wsExNoActivate  = uintptr(0x08000000)
	wsExLayered     = uintptr(0x00080000)
	swpFrameChange  = uintptr(0x0020)
	swHide          = uintptr(0)
	swShowNA        = uintptr(8)
	wmNclbuttondown = uintptr(0x00A1)
	htCaption       = uintptr(2)
	htBottomRight   = uintptr(17)
	lwColorKey      = uintptr(1) // LWA_COLORKEY
	lwAlpha         = uintptr(2) // LWA_ALPHA
	vkLButton       = uintptr(0x01)
)

var hwndTopmost = ^uintptr(0)
var overlayColorKey = uintptr(0x00030201) // RGB(1,2,3), used by HUD mode as transparent chrome key.

var vkMap = map[string]uintptr{
	"F1": 0x70, "F2": 0x71, "F3": 0x72, "F4": 0x73,
	"F5": 0x74, "F6": 0x75, "F7": 0x76, "F8": 0x77,
	"F9": 0x78, "F10": 0x79, "F11": 0x7A, "F12": 0x7B,
	"Insert": 0x2D, "Delete": 0x2E,
	"Home": 0x24, "End": 0x23,
	"PageUp": 0x21, "PageDown": 0x22,
	"Pause": 0x13, "ScrollLock": 0x91,
}

type winRect struct{ Left, Top, Right, Bottom int32 }

// FreeConsole detaches the process from its console window so double-clicking
// the exe doesn't leave a blank terminal behind.
func FreeConsole() {
	procFreeConsole.Call()
}

// SetWindowIcon loads icon resource #1 from the running exe (embedded via rsrc.syso)
// and applies it to the window's title bar and taskbar button.
// hwnd is the value returned by webview2.WebView.Window().
//
// Must be called on the UI thread. WebView2 replaces the window icon with the
// page favicon after navigation — call this again via w.Dispatch to override it.
func SetWindowIcon(hwnd uintptr) {
	// GetModuleHandle(NULL) returns the current exe's HINSTANCE, which is
	// required for LoadImageW to find our embedded icon resource. Passing 0
	// directly looks up OEM/system resources instead.
	hInst, _, _ := procGetModuleHandle.Call(0)
	if hInst == 0 {
		return
	}

	// Load big (32×32) and small (16×16) separately so each is the right size.
	// MAKEINTRESOURCE(1) == uintptr(1) — rsrc uses ID 1 for the first icon.
	hBig, _, _ := procLoadImage.Call(hInst, 1, imageIcon, 32, 32, lrShared)
	hSm, _, _ := procLoadImage.Call(hInst, 1, imageIcon, 16, 16, lrShared)
	if hBig == 0 {
		return
	}
	if hSm == 0 {
		hSm = hBig
	}

	procSendMessage.Call(hwnd, wmSetIcon, iconBig, hBig)
	procSendMessage.Call(hwnd, wmSetIcon, iconSmall, hSm)
	procSetClassLongPtr.Call(hwnd, gclpHIcon, hBig)
	procSetClassLongPtr.Call(hwnd, gclpHIconSm, hSm)

	// Force the non-client area (title bar) to repaint with the new icon.
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoZOrder|swpFrameChange)
}

// Start creates and shows the overlay WebView window. Returns nil if WebView2
// is unavailable.
func Start(url string, cfg *config.Config) webview2.WebView {
	ov := webview2.NewWithOptions(webview2.WebViewOptions{Debug: false})
	if ov == nil {
		return nil
	}
	ov.SetTitle("OOF Overlay")
	ov.SetSize(cfg.OverlayWidth, cfg.OverlayHeight, webview2.HintNone)

	hwnd := windows.HWND(uintptr(ov.Window()))

	bindFunctions(ov, hwnd, cfg)

	ov.Navigate(overlayHUDURL(url))
	configureWindow(hwnd, cfg)
	SetWindowIcon(uintptr(hwnd))
	go listenHotkey(hwnd, cfg)
	return ov
}

func overlayHUDURL(baseURL string) string {
	return baseURL + "?overlay=1&view=overlay&hud=1&nativeHud=1&assetVersion=" + time.Now().Format("20060102150405.000000000")
}

func bindFunctions(ov webview2.WebView, hwnd windows.HWND, cfg *config.Config) {
	ov.Bind("overlayStartDrag", func() {
		procReleaseCapture.Call()
		procSendMessage.Call(uintptr(hwnd), wmNclbuttondown, htCaption, 0)
		go saveRectAfterInteraction(hwnd, cfg, false)
	})

	ov.Bind("overlayStartResize", func() {
		procReleaseCapture.Call()
		procSendMessage.Call(uintptr(hwnd), wmNclbuttondown, htBottomRight, 0)
		go saveRectAfterInteraction(hwnd, cfg, true)
	})

	ov.Bind("overlaySetOpacity", func(alpha int) {
		if alpha < 10 {
			alpha = 10
		}
		if alpha > 255 {
			alpha = 255
		}
		procSetLayeredWindowAttr.Call(uintptr(hwnd), overlayColorKey, uintptr(alpha), lwAlpha|lwColorKey)
		cfg.OverlayOpacity = float64(alpha) / 255.0
		_ = config.Save(config.ConfigPath(), *cfg)
	})
}

func saveRectAfterInteraction(hwnd windows.HWND, cfg *config.Config, saveSize bool) {
	for {
		state, _, _ := procGetAsyncKeyState.Call(vkLButton)
		if state&0x8000 == 0 {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	var r winRect
	procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&r)))
	cfg.OverlayX = int(r.Left)
	cfg.OverlayY = int(r.Top)
	if saveSize {
		cfg.OverlayWidth = int(r.Right - r.Left)
		cfg.OverlayHeight = int(r.Bottom - r.Top)
	}
	_ = config.Save(config.ConfigPath(), *cfg)
}

func configureWindow(hwnd windows.HWND, cfg *config.Config) {
	procSetWindowLong.Call(uintptr(hwnd), gwlStyle, wsPopup|wsVisible)
	procSetWindowLong.Call(uintptr(hwnd), gwlExStyle,
		wsExTopmost|wsExToolWindow|wsExNoActivate|wsExLayered)

	alpha := int(cfg.OverlayOpacity * 255)
	if alpha < 10 {
		alpha = 10
	}
	if alpha > 255 {
		alpha = 255
	}
	procSetLayeredWindowAttr.Call(uintptr(hwnd), overlayColorKey, uintptr(alpha), lwAlpha|lwColorKey)

	x, y := cfg.OverlayX, cfg.OverlayY
	if x < 0 || y < 0 {
		sw, _, _ := procGetSystemMetrics.Call(0)
		sh, _, _ := procGetSystemMetrics.Call(1)
		x = (int(sw) - cfg.OverlayWidth) / 2
		y = (int(sh) - cfg.OverlayHeight) / 2
	}

	procSetWindowPos.Call(
		uintptr(hwnd), hwndTopmost,
		uintptr(x), uintptr(y),
		uintptr(cfg.OverlayWidth), uintptr(cfg.OverlayHeight),
		swpFrameChange,
	)
	procShowWindow.Call(uintptr(hwnd), swHide)
}

func listenHotkey(hwnd windows.HWND, cfg *config.Config) {
	var prev bool
	visible := false
	// Keep F9 as a native window show/hide toggle only. Do not use this signal
	// to pause the HUD webview renderer; the overlay must keep its live state
	// simple and recoverable even when the native window is hidden.
	for range time.Tick(50 * time.Millisecond) {
		key := cfg.OverlayHotkey
		vk, ok := vkMap[key]
		if !ok {
			vk = vkMap["F9"]
		}
		state, _, _ := procGetAsyncKeyState.Call(vk)
		curr := state&0x8000 != 0

		if cfg.OverlayHoldMode {
			if curr && !visible {
				procShowWindow.Call(uintptr(hwnd), swShowNA)
				visible = true
			} else if !curr && visible {
				procShowWindow.Call(uintptr(hwnd), swHide)
				visible = false
			}
		} else {
			if curr && !prev {
				if visible {
					procShowWindow.Call(uintptr(hwnd), swHide)
				} else {
					procShowWindow.Call(uintptr(hwnd), swShowNA)
				}
				visible = !visible
			}
		}
		prev = curr
	}
}
