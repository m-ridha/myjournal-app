package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata" // Keep IANA locations available in minimal deployment images.

	"github/m-ridha/myjournal-app/internal/config"
)

// version is set from the validated release tag by the container build.
var version string

func main() {
	slog.SetDefault(newLogger(config.Config{}))
	if err := run(); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	app, cleanup, err := BuildApp(cfg)
	if err != nil {
		return err
	}
	defer cleanup()
	// Request middleware and failure paths share the configured logging policy.
	slog.SetDefault(app.Logger)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	listener, err := net.Listen("tcp", app.Server.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	app.Logger.Info("starting server", "addr", app.Server.Addr, "base_url", cfg.BaseURL, "providers", cfg.Providers)
	return serve(ctx, app.Server, listener)
}
