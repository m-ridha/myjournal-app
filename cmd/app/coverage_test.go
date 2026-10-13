package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"testing"

	"github/m-ridha/myjournal-app/internal/config"
	provider "github/m-ridha/myjournal-app/internal/core/connection"
	"github/m-ridha/myjournal-app/internal/core/journal"
	"github/m-ridha/myjournal-app/internal/driven/clock"
	"github/m-ridha/myjournal-app/internal/driven/provider/local"
)

type failingStoreProvider struct{ provider.Provider }

func (failingStoreProvider) ID() string { return "unavailable" }
func (failingStoreProvider) Store(context.Context, provider.Connection) (journal.Store, error) {
	return nil, journal.ErrUnavailable
}

func TestServiceFactoryBuildsAndRejectsConnections(t *testing.T) {
	cache, cleanup, err := newJournalCache()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	registry := provider.NewRegistry(local.New("http://localhost:3030", t.TempDir()), failingStoreProvider{})
	factory := newServiceFactory(registry, cache, clock.System{})
	if _, err := factory(context.Background(), provider.Session{Connection: provider.Connection{ProviderID: "missing"}}); err == nil {
		t.Fatal("unknown provider accepted")
	}
	if _, err := factory(context.Background(), provider.Session{Connection: provider.Connection{ProviderID: "unavailable"}}); !errors.Is(err, journal.ErrUnavailable) {
		t.Fatal(err)
	}
	service, err := factory(context.Background(), provider.Session{Connection: provider.Connection{ProviderID: "local", AccountID: "dev"}})
	if err != nil {
		t.Fatal(err)
	}
	date := journal.Date{Year: 2020, Month: 1, Day: 1}
	if _, err := service.Save(context.Background(), date, "synthetic", ""); err != nil {
		t.Fatal(err)
	}
	if entry, err := service.Open(context.Background(), date); err != nil || entry.Text != "synthetic" {
		t.Fatal("injected provider store failed")
	}
}
func TestLoggerHonorsConfiguredThreshold(t *testing.T) {
	for _, tc := range []struct {
		level    string
		enabled  slog.Level
		disabled slog.Level
	}{{"debug", slog.LevelDebug, slog.LevelDebug - 1}, {"warn", slog.LevelWarn, slog.LevelInfo}, {"error", slog.LevelError, slog.LevelWarn}} {
		logger := newLogger(config.Config{LogLevel: tc.level})
		if !logger.Enabled(context.Background(), tc.enabled) || logger.Enabled(context.Background(), tc.disabled) {
			t.Fatal("logging threshold not applied")
		}
	}
}
func TestRunRejectsInvalidConfigurationAndOccupiedListener(t *testing.T) {
	defer slog.SetDefault(slog.Default())
	for _, key := range []string{"APP_BASE_URL", "PORT", "LOG_LEVEL", "SESSION_KEY", "COOKIE_SECURE", "PROVIDERS", "LOCAL_DATA_DIR", "GOOGLE_TEST_MODE", "GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET"} {
		t.Setenv(key, "")
	}
	t.Setenv("PORT", "invalid")
	if err := run(); err == nil {
		t.Fatal("invalid port accepted")
	}
	t.Setenv("PORT", "3030")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("GOOGLE_TEST_MODE", "false")
	if err := run(); err == nil {
		t.Fatal("invalid app configuration accepted")
	}
	t.Setenv("APP_BASE_URL", "http://localhost:3030")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("SESSION_KEY", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	t.Setenv("PROVIDERS", "google")
	if err := run(); err == nil {
		t.Fatal("incomplete Google setup accepted")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	port := listener.Addr().(*net.TCPAddr).Port
	t.Setenv("PORT", strconv.Itoa(port))
	t.Setenv("APP_BASE_URL", "http://127.0.0.1:"+strconv.Itoa(port))
	t.Setenv("PROVIDERS", "local")
	t.Setenv("LOCAL_DATA_DIR", t.TempDir())
	if err := run(); err == nil {
		t.Fatal("occupied listener accepted")
	}
}

type shutdownErrorListener struct {
	net.Listener
	started chan struct{}
	once    sync.Once
}

func (l *shutdownErrorListener) Accept() (net.Conn, error) {
	l.once.Do(func() { close(l.started) })
	return l.Listener.Accept()
}
func (l *shutdownErrorListener) Close() error { _ = l.Listener.Close(); return errListenerClose }

var errListenerClose = errors.New("synthetic listener close failure")

func TestServeHandlesAlreadyClosedServerAndShutdownFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := newHTTPServer(config.Config{}, nil)
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	if err := serve(context.Background(), server, listener); err != nil {
		t.Fatal("already closed server returned failure")
	}
	base, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = base.Close() }()
	broken := &shutdownErrorListener{Listener: base, started: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- serve(ctx, newHTTPServer(config.Config{}, nil), broken) }()
	<-broken.started
	cancel()
	if err := <-done; !errors.Is(err, errListenerClose) {
		t.Fatalf("shutdown error identity lost: %v", err)
	}
}
