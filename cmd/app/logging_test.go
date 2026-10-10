package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github/mridha/myjournal-app/internal/config"
	middleware "github/mridha/myjournal-app/internal/web/middleware"
)

type captureHandler struct {
	calls int
	level slog.Level
}

func (h *captureHandler) Enabled(_ context.Context, level slog.Level) bool { return level >= h.level }

func (h *captureHandler) Handle(_ context.Context, _ slog.Record) error { h.calls++; return nil }

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *captureHandler) WithGroup(string) slog.Handler { return h }

func TestConfiguredLoggerFiltersRequestLogs(t *testing.T) {
	old := slog.Default()
	defer slog.SetDefault(old)
	logger := newLogger(config.Config{LogLevel: "error"})
	if logger.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("info enabled at error level")
	}
	captured := &captureHandler{level: slog.LevelError}
	slog.SetDefault(slog.New(captured))
	middleware.RequestLog(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if captured.calls != 0 {
		t.Fatal("request info escaped configured level")
	}
}
