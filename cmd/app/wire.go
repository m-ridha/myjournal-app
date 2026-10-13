//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/google/wire"

	"github/mridha/myjournal-app/internal/config"
	provider "github/mridha/myjournal-app/internal/core/connection"
	"github/mridha/myjournal-app/internal/core/journal"
	"github/mridha/myjournal-app/internal/driven/cache/journal"
	cacheAdapter "github/mridha/myjournal-app/internal/driven/cache/local/bigcache"
	"github/mridha/myjournal-app/internal/driven/clock"
	googleProvider "github/mridha/myjournal-app/internal/driven/provider/google"
	localProvider "github/mridha/myjournal-app/internal/driven/provider/local"
	"github/mridha/myjournal-app/internal/driver/web"
	webHandler "github/mridha/myjournal-app/internal/driver/web/handler"
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
		newRenderer,
		newWebOptions,
		newConnectionService,
		newHTTPServer,
		newApp,
	))
}

func newRenderer() (*web.Renderer, error) {
	return web.NewRendererWithVersion(version)
}

func newClock() journal.Clock {
	return clock.System{}
}

func newJournalCache() (*cachestore.Cache, func(), error) {
	const ttl = 2 * time.Minute
	backend, err := cacheAdapter.New(ttl)
	if err != nil {
		return nil, nil, fmt.Errorf("initialize journal cache: %w", err)
	}
	cache, stopWarming := cachestore.NewWithWarming(backend, ttl)
	return cache, func() { stopWarming(); _ = backend.Close() }, nil
}

func newServiceFactory(registry *provider.Registry, cache *cachestore.Cache, clk journal.Clock) webHandler.ServiceFactory {
	return func(ctx context.Context, current provider.Session) (webHandler.JournalService, error) {
		p, ok := registry.Get(current.ProviderID)
		if !ok {
			return nil, fmt.Errorf("session provider is not configured")
		}
		store, err := p.Store(ctx, current.Connection)
		if err != nil {
			return nil, err
		}
		return journal.NewService(cache.WrapWithFactory(current.ProviderID+":"+current.AccountID, store, func(ctx context.Context) (journal.Store, error) {
			return p.Store(ctx, current.Connection)
		}), clk), nil
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

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	if version != "" {
		logger = logger.With("version", version)
	}
	return logger
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
			if cfg.GoogleTestMode {
				if err := config.ValidateLoopbackURL(cfg.BaseURL); err != nil {
					return nil, fmt.Errorf("Google test mode requires localhost: %w", err)
				}
			} else {
				origin, err := url.Parse(cfg.BaseURL)
				if err != nil || origin.Scheme != "https" || origin.Hostname() == "" || origin.User != nil || origin.Path != "" || origin.RawQuery != "" || origin.ForceQuery || origin.Fragment != "" {
					return nil, fmt.Errorf("Google requires an HTTPS origin without credentials, path, query or fragment")
				}
				if !cfg.CookieSecure {
					return nil, fmt.Errorf("Google HTTPS deployment requires COOKIE_SECURE=true")
				}
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
	addr := fmt.Sprintf(":%d", cfg.Port)
	if cfg.GoogleTestMode || slices.Contains(cfg.ProviderIDs(), provider.LocalID) {
		host := "127.0.0.1"
		origin, _ := url.Parse(cfg.BaseURL)
		if origin != nil {
			if ip := net.ParseIP(origin.Hostname()); ip != nil && ip.IsLoopback() {
				host = ip.String()
			}
		}
		addr = net.JoinHostPort(host, fmt.Sprint(cfg.Port))
	}
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ErrorLog:          log.New(serverLogWriter{}, "", 0),
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

func newConnectionService(registry *provider.Registry, clk journal.Clock) *provider.Service {
	return provider.NewService(registry, clk, rand.Reader)
}
func newWebOptions(cfg config.Config) web.Options {
	origin, _ := url.Parse(cfg.BaseURL)
	return web.Options{BaseURL: cfg.BaseURL, Session: webHandler.Options{SessionKey: cfg.SessionKey, CookieSecure: cfg.CookieSecure}, HTTPS: origin != nil && origin.Scheme == "https"}
}
