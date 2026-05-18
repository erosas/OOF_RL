//go:build windows

package overlay

import (
	"strings"
	"sync"
	"time"

	webview2 "github.com/jchv/go-webview2"
	"golang.org/x/sys/windows"

	"OOF_RL/internal/config"
)

const defaultManualShellTitle = "OOF Overlay"

var (
	uiDispatcherMu sync.RWMutex
	uiDispatcher   func(func())
)

// SetUIDispatcher configures the app WebView dispatch function used by manual
// shells. Manual shell creation remains request-driven, but WebView2 windows
// must be created on the UI thread that owns the running message loop.
func SetUIDispatcher(dispatch func(func())) {
	uiDispatcherMu.Lock()
	defer uiDispatcherMu.Unlock()
	uiDispatcher = dispatch
}

func currentUIDispatcher() func(func()) {
	uiDispatcherMu.RLock()
	defer uiDispatcherMu.RUnlock()
	return uiDispatcher
}

// ManualShellOptions configure an explicitly launched overlay shell. This path
// intentionally avoids hotkey polling, persistence bindings, and lifecycle
// ownership; callers decide if and when to create or destroy the returned shell.
type ManualShellOptions struct {
	Title   string
	Visible bool
}

// ManualShell is a narrowly owned native shell handle for request-driven
// visibility control. It deliberately does not own hotkeys, polling, settings,
// or plugin lifecycle semantics.
type ManualShell struct {
	webview webview2.WebView
	hwnd    windows.HWND
}

// StartManualShell creates a native overlay WebView shell for a specific URL.
// It is request-driven only: no F9 listener, autostart behavior, render loop, or
// plugin lifecycle ownership is installed here.
func StartManualShell(url string, cfg *config.Config, opts ManualShellOptions) *ManualShell {
	url = strings.TrimSpace(url)
	if url == "" || cfg == nil {
		return nil
	}

	if dispatch := currentUIDispatcher(); dispatch != nil {
		result := make(chan *ManualShell, 1)
		dispatch(func() {
			result <- startManualShellOnCurrentThread(url, cfg, opts)
		})
		select {
		case shell := <-result:
			return shell
		case <-time.After(2 * time.Second):
			return nil
		}
	}

	return startManualShellOnCurrentThread(url, cfg, opts)
}

func startManualShellOnCurrentThread(url string, cfg *config.Config, opts ManualShellOptions) *ManualShell {
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
	return &ManualShell{webview: ov, hwnd: hwnd}
}

func (s *ManualShell) Show() {
	if s == nil || s.webview == nil || s.hwnd == 0 {
		return
	}
	s.webview.Dispatch(func() {
		procShowWindow.Call(uintptr(s.hwnd), swShowNA)
	})
}

func (s *ManualShell) Hide() {
	if s == nil || s.webview == nil || s.hwnd == 0 {
		return
	}
	s.webview.Dispatch(func() {
		procShowWindow.Call(uintptr(s.hwnd), swHide)
	})
}

func (s *ManualShell) Dormant() {
	s.Hide()
}
