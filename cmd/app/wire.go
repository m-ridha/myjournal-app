//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/wire"

	"github/mridha/myjournal-app/internal/cachestore"
	"github/mridha/myjournal-app/internal/clock"
	"github/mridha/myjournal-app/internal/config"
	"github/mridha/myjournal-app/internal/journal"
	"github/mridha/myjournal-app/internal/provider"
	googleProvider "github/mridha/myjournal-app/internal/provider/google"
	localProvider "github/mridha/myjournal-app/internal/provider/local"
	"github/mridha/myjournal-app/internal/web"
	webHandler "github/mridha/myjournal-app/internal/web/handler"
)

// App is the runtime composition built in the cmd/app layer.
type App struct {
	Server   *http.Server
	Logger   *slog.Logger
	Registry *provider.Registry
}

func BuildApp(cfg config.Config) (*App, func(), error) {
	panic(wire.Build(
		newLogger,
		newProviderRegistry,
		newClock,
		newJournalCache,
		newServiceFactory,
		web.NewRouter,
		newHTTPServer,
		newApp,
	))
}

func newClock() journal.Clock {
	return clock.System{}
}

func newJournalCache() *cachestore.Cache {
	return cachestore.New(2 * time.Minute)
}

func newServiceFactory(registry *provider.Registry, cache *cachestore.Cache, clk journal.Clock) webHandler.ServiceFactory {
	return func(ctx context.Context) (*journal.Service, error) {
		p, ok := registry.Get("local")
		if !ok {
			return nil, fmt.Errorf("local provider is not configured")
		}
		store, err := p.Store(ctx, provider.Connection{ProviderID: "local", AccountID: "dev", AccountEmail: "dev@localhost", Credentials: "local-dev-user"})
		if err != nil {
			return nil, err
		}
		return journal.NewService(cache.Wrap("local:dev", store), clk), nil
	}
}

func newLogger(cfg config.Config) *slog.Logger {
	level := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

func newProviderRegistry(cfg config.Config) *provider.Registry {
	selected := strings.FieldsFunc(cfg.Providers, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';'
	})
	if len(selected) == 0 {
		selected = []string{"local"}
	}

	providers := make([]provider.Provider, 0, len(selected))
	for _, name := range selected {
		switch strings.TrimSpace(name) {
		case "google":
			providers = append(providers, googleProvider.New(googleProvider.ConfigFromEnv()))
		case "local":
			providers = append(providers, localProvider.New(cfg.BaseURL, cfg.LocalDataDir))
		case "":
			continue
		default:
			providers = append(providers, localProvider.New(cfg.BaseURL, cfg.LocalDataDir))
		}
	}
	if len(providers) == 0 {
		providers = append(providers, localProvider.New(cfg.BaseURL, cfg.LocalDataDir))
	}

	return provider.NewRegistry(providers...)
}

func newHTTPServer(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func newApp(logger *slog.Logger, registry *provider.Registry, server *http.Server) *App {
	return &App{
		Server:   server,
		Logger:   logger,
		Registry: registry,
	}
}
