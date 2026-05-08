// Astrolabe panel backend entrypoint.
//
// Startup: parse flags, load config, open store and services, register JSON-RPC
// over HTTP, spin up the SSE hub, listen, shut down on SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/pflag"

	"astrolabe/internal/adapter"
	_ "astrolabe/internal/adapter/docker"
	_ "astrolabe/internal/adapter/local"
	_ "astrolabe/internal/adapter/netdata"
	"astrolabe/internal/api"
	"astrolabe/internal/config"
	"astrolabe/internal/core/datasource"
	"astrolabe/internal/core/iconify"
	"astrolabe/internal/core/metric"
	"astrolabe/internal/core/upload"
	"astrolabe/internal/events"
	"astrolabe/internal/pkg/logging"
	"astrolabe/internal/probe"
	"astrolabe/internal/rpc"
	"astrolabe/internal/rpc/handlers"
	"astrolabe/internal/store"
)

// String set via -ldflags.
var (
	version = "dev"
	commit  = "none"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "astrolabe: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	flags := pflag.NewFlagSet("astrolabe", pflag.ContinueOnError)
	configPath := flags.String("config", "", "Path to config.json (default: ~/.astrolabe_panel/config.json)")
	showVersion := flags.Bool("version", false, "Print version and exit")
	if err := flags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return nil
		}
		return err
	}
	if *showVersion {
		fmt.Printf("astrolabe %s (%s)\n", version, commit)
		return nil
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger, err := logging.Init(logging.Config{Level: cfg.Log.Level, Format: cfg.Log.Format})
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}

	logger.Info("astrolabe starting",
		"version", version,
		"commit", commit,
		"config_path", cfg.SourcePath,
		"data_dir", cfg.Paths.DataDir,
		"db_path", cfg.Paths.DBPath,
		"listen", cfg.Addr(),
		"locale", cfg.I18n.DefaultLocale,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, store.Options{DBPath: cfg.Paths.DBPath})
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer func() {
		if err := st.Close(); err != nil {
			logger.Warn("store close failed", "err", err)
		}
	}()

	// Event hub: must outlive RPC handlers and the probe scheduler so both can
	// publish without racing on startup.
	hub := events.NewHub()
	defer hub.Close()

	scheduler := probe.NewScheduler(st, probe.Options{
		DefaultIntervalSec: cfg.Probe.DefaultIntervalSec,
		DefaultTimeoutSec:  cfg.Probe.DefaultTimeoutSec,
	})
	scheduler.SetHub(hub)

	dsManager := datasource.NewManager(st, adapter.DefaultRegistry)
	defer dsManager.Close()
	pipeline := metric.New(dsManager, st)
	pipeline.SetHub(hub)
	cleaner := metric.NewCleaner(st, cfg.Metric.RetentionMinutes, cfg.Metric.CleanupIntervalMinutes)

	iconifyProxy := iconify.New(cfg.Iconify.Mirror, filepath.Join(cfg.Paths.DataDir, "iconify_cache"), logger)
	uploader, err := upload.New(cfg.Paths.UploadDir)
	if err != nil {
		return fmt.Errorf("init uploader: %w", err)
	}

	registry := rpc.NewRegistry()
	handlers.RegisterSystem(registry)
	handlers.RegisterBoard(registry, st, hub)
	handlers.RegisterWidget(registry, st, hub)
	handlers.RegisterProbe(registry, scheduler)
	handlers.RegisterDataSource(registry, st, dsManager, hub)
	handlers.RegisterMetric(registry, pipeline)
	handlers.RegisterIconify(registry, iconifyProxy)

	logger.Info("rpc methods registered", "methods", registry.Methods())
	logger.Info("data source adapters registered", "types", adapter.DefaultRegistry.Types())

	router, err := api.New(api.Options{
		Logger:    logger,
		Registry:  registry,
		Events:    hub,
		Build:     api.BuildInfo{Version: version, Commit: commit},
		UploadDir: cfg.Paths.UploadDir,
		Uploader:  uploader,
		Store:     st,
		DSManager: dsManager,
	})
	if err != nil {
		return fmt.Errorf("build router: %w", err)
	}

	httpServer := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	bgCtx, stopBg := context.WithCancel(context.Background())
	defer stopBg()
	probeDone := make(chan struct{})
	go func() {
		defer close(probeDone)
		scheduler.Run(bgCtx)
	}()
	cleanerDone := make(chan struct{})
	go func() {
		defer close(cleanerDone)
		cleaner.Run(bgCtx)
	}()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "addr", cfg.Addr())
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Warn("http shutdown failed", "err", err)
	}

	stopBg()
	select {
	case <-probeDone:
	case <-time.After(5 * time.Second):
		logger.Warn("probe scheduler did not stop within 5s")
	}
	select {
	case <-cleanerDone:
	case <-time.After(5 * time.Second):
		logger.Warn("metric cleaner did not stop within 5s")
	}

	if err := <-serverErr; err != nil {
		return fmt.Errorf("http server: %w", err)
	}
	logger.Info("astrolabe stopped")
	return nil
}
