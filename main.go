package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"

	"github.com/jchv/go-webview2"

	"OOF_RL/internal/config"
	"OOF_RL/internal/db"
	"OOF_RL/internal/hub"
	"OOF_RL/internal/rl"
	"OOF_RL/internal/server"
)

//go:embed web/*
var webFS embed.FS

func main() {
	// Redirect logs to a file so they're not lost when the console is hidden,
	// then detach from the console window so double-clicking the exe doesn't
	// leave a terminal open. Check oof_rl.log for diagnostics.
	if f, err := os.OpenFile("oof_rl.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		log.SetOutput(f)
		defer f.Close()
		freeConsole()
	}

	cfg, err := config.Load("config.toml")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	h := hub.New()

	rlClient := rl.New(&cfg, database, h)
	go rlClient.Run()

	mux := http.NewServeMux()
	static := http.FileServer(http.FS(subFS(webFS, "web")))
	srv := server.New(&cfg, database, h, static, rlClient.Reconnect)
	srv.Register(mux)

	addr := fmt.Sprintf(":%d", cfg.AppPort)
	log.Printf("OOF RL running at http://localhost%s", addr)

	// Start HTTP server in background; shut it down when the window closes.
	httpSrv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	url := fmt.Sprintf("http://localhost%s", addr)

	if cfg.OpenInBrowser {
		// Browser mode: open system browser (DevTools available) and block on Ctrl+C.
		if err := exec.Command("cmd", "/c", "start", url).Start(); err != nil {
			log.Printf("could not open browser: %v — navigate to %s manually", err, url)
		}
		log.Printf("Browser mode active — open DevTools in your browser. Press Ctrl+C to stop.")
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)
		<-quit
		_ = httpSrv.Shutdown(context.Background())
		return
	}

	w := webview2.NewWithOptions(webview2.WebViewOptions{Debug: false})
	if w == nil {
		log.Fatal("failed to create webview (is WebView2 runtime installed?)")
	}
	defer w.Destroy()

	w.SetTitle("OOF RL")
	w.SetSize(1280, 800, webview2.HintNone)
	w.Navigate(url)

	// Overlay: hidden by default, press the configured hotkey to toggle.
	// Loads the main live-view page in an always-on-top borderless window.
	// Shares the same message loop as the main window — no extra thread needed.
	if ov := startOverlay(url, &cfg); ov != nil {
		defer ov.Destroy()
	}

	w.Run() // blocks until window is closed; also drives the overlay's message loop

	// Clean shutdown when the window is closed.
	_ = httpSrv.Shutdown(context.Background())
}