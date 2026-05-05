package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	webview2 "github.com/jchv/go-webview2"

	"OOF_RL/internal/config"
	"OOF_RL/internal/core"
	"OOF_RL/internal/db"
	"OOF_RL/internal/hub"
	"OOF_RL/internal/logging"
	"OOF_RL/internal/mmr"
	"OOF_RL/internal/mmr/rlstats"
	"OOF_RL/internal/mmr/trackergg"
	"OOF_RL/internal/overlay"
	"OOF_RL/internal/plugins/ballchasing"
	"OOF_RL/internal/plugins/history"
	"OOF_RL/internal/plugins/live"
	"OOF_RL/internal/plugins/ranks"
	"OOF_RL/internal/plugins/session"
	"OOF_RL/internal/rl"
	"OOF_RL/internal/singleinstance"
)

//go:embed web/*
var webFS embed.FS

func main() {
	cfgPath := config.ConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Log to data dir before acquiring the instance lock so duplicate launches
	// leave an audit trail in oof_rl.log before exiting.
	if err := logging.Rotate(cfg.LogPath(), logging.RotateOptions{Retain: 5}); err != nil {
		log.Printf("log rotation: %v", err)
	}
	if f, err := os.OpenFile(cfg.LogPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		log.SetOutput(f)
		defer f.Close()
		overlay.FreeConsole()
	}

	appLock, err := singleinstance.Acquire(cfg.DataDir)
	if err != nil {
		if errors.Is(err, singleinstance.ErrAlreadyRunning) {
			log.Printf("[core] duplicate instance rejected; lock already held: %s", singleinstance.LockPath(cfg.DataDir))
			return
		}
		log.Fatalf("single instance lock: %v", err)
	}
	defer appLock.Release()
	log.Printf("[core] single-instance lock acquired: %s", appLock.Path())

	database, err := db.Open(cfg.DBPath())
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	h := hub.New()

	trnProvider := mmr.NewFallbackProvider(trackergg.New(), rlstats.New())

	webSub, _ := fs.Sub(webFS, "web")
	mux := http.NewServeMux()

	var rlReconnect func()
	srv := core.NewServer(cfgPath, &cfg, database, h, http.FileServer(http.FS(webSub)), func() {
		if rlReconnect != nil {
			rlReconnect()
		}
	}, trnProvider)
	srv.Use(live.New())
	srv.Use(ranks.New())
	srv.Use(history.New(&cfg, database))
	srv.Use(session.New(database))
	srv.Use(ballchasing.New(&cfg, database, h))
	srv.Register(mux)

	rlClient := rl.New(&cfg, h)
	rlReconnect = rlClient.Reconnect
	rlClient.SetDispatch(srv.DispatchEvent)
	go rlClient.Run()

	ln, port := bindAvailablePort(cfg.AppPort)
	url := fmt.Sprintf("http://localhost:%d", port)
	log.Printf("OOF RL running at %s", url)

	httpSrv := &http.Server{Handler: mux}
	go func() {
		if err := httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	w := webview2.NewWithOptions(webview2.WebViewOptions{Debug: false})
	if w == nil {
		log.Fatal("failed to create webview (is WebView2 runtime installed?)")
	}
	defer w.Destroy()
	w.SetTitle("OOF RL")
	w.SetSize(1280, 800, webview2.HintNone)
	w.Navigate(url)

	// Set once immediately, then again after WebView2 finishes loading because
	// it replaces the window icon with the page favicon after navigation.
	hwnd := uintptr(w.Window())
	overlay.SetWindowIcon(hwnd)
	go func() {
		time.Sleep(800 * time.Millisecond)
		w.Dispatch(func() { overlay.SetWindowIcon(hwnd) })
	}()

	if ov := overlay.Start(url, &cfg); ov != nil {
		defer ov.Destroy()
	}

	w.Run()
	_ = httpSrv.Shutdown(context.Background())
}

func bindAvailablePort(start int) (net.Listener, int) {
	if start <= 0 {
		start = 8080
	}
	for port := start; port < start+20; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			return ln, port
		}
	}
	log.Fatalf("could not bind any port in range %d–%d", start, start+19)
	return nil, 0
}
