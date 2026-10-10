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
	"github/mridha/myjournal-app/internal/session"
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
	return func(ctx context.Context, current session.Session) (*journal.Service, error) {
		p, ok := registry.Get(current.ProviderID)
		if !ok {
			return nil, fmt.Errorf("session provider is not configured")
		}
		store, err := p.Store(ctx, current.Connection)
		if err != nil {
			return nil, err
		}
		return journal.NewService(cache.Wrap(current.ProviderID+":"+current.AccountID, store), clk), nil
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

func newProviderRegistry(cfg config.Config) (*provider.Registry, error) {
	selected := cfg.ProviderIDs()
	if len(selected) == 0 {
		return nil, fmt.Errorf("PROVIDERS must contain at least one provider")
	}

	providers := make([]provider.Provider, 0, len(selected))
	seen := make(map[string]struct{}, len(selected))
	for _, name := range selected {
		switch name = strings.TrimSpace(name); name {
		case provider.LocalID:
			if _, ok := seen[name]; ok {
				return nil, fmt.Errorf("PROVIDERS contains duplicate provider %q", name)
			}
			if err := config.ValidateLoopbackURL(cfg.BaseURL); err != nil {
				return nil, err
			}
			providers = append(providers, localProvider.New(cfg.BaseURL, cfg.LocalDataDir))
			seen[name] = struct{}{}
		case provider.GoogleID:
			if _, ok := seen[name]; ok {
				return nil, fmt.Errorf("PROVIDERS contains duplicate provider %q", name)
			}
			if !cfg.GoogleTestMode {
				return nil, fmt.Errorf("Google is awaiting live acceptance; use GOOGLE_TEST_MODE=true only for localhost integration testing")
			}
			if err := config.ValidateLoopbackURL(cfg.BaseURL); err != nil {
				return nil, fmt.Errorf("Google test mode requires localhost: %w", err)
			}
			gc := googleProvider.Config{ClientID: cfg.GoogleClientID, ClientSecret: cfg.GoogleClientSecret, BaseURL: cfg.BaseURL}
			if gc.ClientID == "" || gc.ClientSecret == "" {
				return nil, fmt.Errorf("Google OAuth client configuration is missing")
			}
			providers = append(providers, googleProvider.New(gc))
			seen[name] = struct{}{}
		case "":
			return nil, fmt.Errorf("PROVIDERS contains an empty provider name")
		default:
			return nil, fmt.Errorf("PROVIDERS contains an unsupported provider")
		}
	}
	return provider.NewRegistry(providers...), nil
}

func newHTTPServer(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func newApp(logger *slog.Logger, registry *provider.Registry, server *http.Server) *App {
	return &App{
		Server:   server,
		Logger:   logger,
		Registry: registry,
	}
}
