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
	_ "net/http/pprof"

	"os"
	"time"

	"github.com/arl/statsviz"

	webview2 "github.com/jchv/go-webview2"

	"OOF_RL/internal/config"
	"OOF_RL/internal/core"
	"OOF_RL/internal/db"
	"OOF_RL/internal/hub"
	"OOF_RL/internal/logging"
	"OOF_RL/internal/mmr"
	"OOF_RL/internal/mmr/rlstats"
	"OOF_RL/internal/mmr/trackergg"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/overlay"
	"OOF_RL/internal/rl"
	"OOF_RL/internal/singleinstance"
	"OOF_RL/internal/update"
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
		logging.RedirectStderr(f)
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

	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		log.Fatalf("event bus: %v", err)
	}

	// Short TTL: the cache only collapses bursts (a match's players all looked
	// up at once, or rapid re-opens of a view) into one upstream fetch. MMR
	// changes every match, so we effectively re-fetch on any real revisit.
	const trackerCacheTTL = 5 * time.Second
	trnProvider := mmr.NewCachedProvider(
		mmr.NewFallbackProvider(trackergg.New(), rlstats.New()),
		database,
		trackerCacheTTL,
	)

	webSub, _ := fs.Sub(webFS, "web")
	mux := http.NewServeMux()

	var rlReconnect func()
	srv := core.NewServer(cfgPath, &cfg, database, h, http.FileServer(http.FS(webSub)), func() {
		if rlReconnect != nil {
			rlReconnect()
		}
	}, trnProvider, bus)
	if err := srv.LoadPlugins(); err != nil {
		log.Fatalf("plugin load: %v", err)
	}
	if err := seedBundledWASMPlugins(cfg.PluginsDir()); err != nil {
		log.Printf("[core] bundled wasm plugins: %v", err)
	}
	log.Printf("[core] wasm plugins dir: %s", cfg.PluginsDir())
	if err := srv.LoadWASMPlugins(cfg.PluginsDir()); err != nil {
		log.Fatalf("wasm plugin load: %v", err)
	}
	updates := update.New(config.AppVersion)
	srv.SetUpdateChecker(updates)
	go updates.RunPeriodic(context.Background(), 24*time.Hour)
	srv.Register(mux)
	if cfg.DevMode {
		mux.Handle("/debug/pprof/", http.DefaultServeMux)
		if err := statsviz.Register(mux); err != nil {
			log.Printf("statsviz: %v", err)
		}
	}

	if err := srv.InitPlugins(); err != nil {
		log.Fatalf("plugin init: %v", err)
	}

	rlClient := rl.New(&cfg, h)
	rlReconnect = rlClient.Reconnect
	rlClient.SetDispatch(srv.DispatchEvent)
	go rlClient.Run()

	ln, port := bindAvailablePort(cfg.AppPort)
	url := fmt.Sprintf("http://localhost:%d", port)
	log.Printf("OOF RL running at %s", url)
	log.Printf("pprof     at %s/debug/pprof/", url)
	log.Printf("statsviz  at %s/debug/statsviz/", url)

	writeTimeout := 30 * time.Second
	if cfg.DevMode {
		// pprof CPU profiles stream for up to 30s; give the write enough
		// headroom so the timeout doesn't race the response.
		writeTimeout = 90 * time.Second
	}
	httpSrv := &http.Server{
		Handler:           core.LocalhostGuard(mux),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       120 * time.Second,
	}
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
	overlay.SetUIDispatcher(w.Dispatch)
	defer overlay.SetUIDispatcher(nil)
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
	overlay.SetOpacityHook(func(opacity float64) { srv.BroadcastOpacity(opacity) })

	w.Run()
	srv.ShutdownPlugins()
	bus.Stop()
	_ = httpSrv.Shutdown(context.Background())
}

func bindAvailablePort(start int) (net.Listener, int) {
	if start <= 0 {
		start = 8080
	}
	for port := start; port < start+20; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			return ln, port
		}
	}
	log.Fatalf("could not bind any port in range %d–%d", start, start+19)
	return nil, 0
}
