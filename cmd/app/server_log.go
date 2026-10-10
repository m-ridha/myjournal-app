package main

import "log/slog"

// serverLogWriter discards net/http diagnostics, which may contain raw panic
// values, peer addresses or protocol data. Keep only a generic failure signal.
type serverLogWriter struct{}

func (serverLogWriter) Write(p []byte) (int, error) {
	slog.Error("HTTP server error")
	return len(p), nil
}
