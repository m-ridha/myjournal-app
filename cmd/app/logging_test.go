package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github/mridha/myjournal-app/internal/config"
	middleware "github/mridha/myjournal-app/internal/driver/web/middleware"
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

func TestLoggerBuildVersion(t *testing.T) {
	originalVersion, originalOutput := version, os.Stdout
	t.Cleanup(func() { version, os.Stdout = originalVersion, originalOutput })
	for _, buildVersion := range []string{"", "v1.2.3"} {
		t.Run(buildVersion, func(t *testing.T) {
			output, err := os.CreateTemp(t.TempDir(), "logs")
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := output.Close(); err != nil {
					t.Error(err)
				}
			}()
			version, os.Stdout = buildVersion, output
			logger := newLogger(config.Config{})
			logger.With("request_id", "synthetic").Info("request completed")
			logger.Error("application failed")
			if _, err := output.Seek(0, 0); err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(output)
			if err != nil {
				t.Fatal(err)
			}
			for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				if strings.Contains(line, "version=") != (buildVersion != "") {
					t.Fatalf("unexpected version field: %s", line)
				}
				if buildVersion != "" && !strings.Contains(line, "version=v1.2.3") {
					t.Fatalf("missing release version: %s", line)
				}
			}
		})
	}
}
