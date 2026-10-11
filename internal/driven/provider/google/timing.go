package google

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// timingTransport records only bounded categories, never request URLs or errors.
// It wraps OAuth so token acquisition is included in time to response headers.
// Body completion includes decoding/consumer time; this is not pure network time.
type timingTransport struct {
	next   http.RoundTripper
	logger *slog.Logger
}

func (t timingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	logger := t.logger
	if logger == nil {
		logger = slog.Default()
	}
	if !logger.Enabled(r.Context(), slog.LevelDebug) {
		return t.next.RoundTrip(r)
	}
	category := "other"
	switch {
	case strings.HasPrefix(r.URL.Path, "/upload/drive/v3/files"):
		category = "upload"
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/drive/v3/files"):
		switch {
		case r.URL.Query().Get("alt") == "media":
			category = "download"
		case r.URL.Path == "/drive/v3/files":
			category = "list"
		default:
			category = "metadata"
		}
	}
	start := time.Now()
	response, err := t.next.RoundTrip(r)
	headers := time.Since(start)
	status := 0
	if response != nil {
		status = response.StatusCode
	}
	var once sync.Once
	finish := func(outcome string) {
		once.Do(func() {
			logger.DebugContext(r.Context(), "google_request", "category", category, "status", status, "outcome", outcome, "headers_duration", headers, "duration", time.Since(start))
		})
	}
	if err != nil {
		finish("transport_error")
		return response, err
	}
	if response == nil || response.Body == nil {
		finish("complete")
		return response, nil
	}
	response.Body = &timingBody{ReadCloser: response.Body, finish: finish}
	return response, nil
}

type timingBody struct {
	io.ReadCloser
	finish func(string)
}

func (b *timingBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err == io.EOF {
		b.finish("complete")
	} else if err != nil {
		b.finish("body_error")
	}
	return n, err
}

func (b *timingBody) Close() error {
	err := b.ReadCloser.Close()
	if err != nil {
		b.finish("body_error")
	} else {
		b.finish("closed")
	}
	return err
}
