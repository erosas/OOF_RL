//go:build windows

package overlay

import (
	"strings"

	webview2 "github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"

	"OOF_RL/internal/config"
)

const defaultManualShellTitle = "OOF Overlay"

// ManualShellOptions configure an explicitly launched overlay shell. This path
// intentionally avoids hotkey polling, persistence bindings, and lifecycle
// ownership; callers decide if and when to create or destroy the returned shell.
type ManualShellOptions struct {
	Title   string
	Visible bool
}

// StartManualShell creates a native overlay WebView shell for a specific URL.
// It is request-driven only: no F9 listener, autostart behavior, render loop, or
// plugin lifecycle ownership is installed here.
func StartManualShell(url string, cfg *config.Config, opts ManualShellOptions) webview2.WebView {
	url = strings.TrimSpace(url)
	if url == "" || cfg == nil {
		return nil
	}

	ov := webview2.NewWithOptions(webview2.WebViewOptions{Debug: false})
	if ov == nil {
		return nil
	}

	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = defaultManualShellTitle
	}

	ov.SetTitle(title)
	ov.SetSize(cfg.OverlayWidth, cfg.OverlayHeight, webview2.HintNone)
	ov.Navigate(url)

	hwnd := windows.HWND(uintptr(ov.Window()))
	configureWindow(hwnd, cfg)
	if opts.Visible {
		procShowWindow.Call(uintptr(hwnd), swShowNA)
	}
	SetWindowIcon(uintptr(hwnd))
	return ov
}
