package main

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github/mridha/myjournal-app/internal/config"
)

func TestHTTPServerLogOmitsRawDetails(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	server := newHTTPServer(config.Config{Port: 3030}, http.NotFoundHandler())
	if server.ErrorLog == nil {
		t.Fatal("newHTTPServer.ErrorLog = nil, want sanitized logger")
	}
	server.ErrorLog.Print("panic serving private-address: private-cookie private-journal-text")
	got := logs.String()
	if strings.Contains(got, "private-") || !strings.Contains(got, `"msg":"HTTP server error"`) {
		t.Fatalf("HTTP server log = %q, want generic event without raw details", got)
	}
}
