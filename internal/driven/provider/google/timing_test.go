package google

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

func TestRequestTimingsAreSanitized(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	transport := timingTransport{next: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("private body"))}, nil
	}), logger: logger}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://www.googleapis.com/drive/v3/files/private-id?alt=media&secret=private-query", nil)
	req.Header.Set("Authorization", "private-token")
	response, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if logs.Len() != 0 {
		t.Fatal("timing must wait for body consumption")
	}
	_, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if strings.Count(logs.String(), "google_request") != 1 {
		t.Fatalf("expected one timing: %s", logs.String())
	}
	for _, secret := range []string{"private-id", "private-query", "private-token", "private body", "googleapis.com"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("timing leaked %s", secret)
		}
	}
	if !strings.Contains(logs.String(), `"category":"download"`) {
		t.Fatal(logs.String())
	}
}

func TestTimingTransportPreservesFailure(t *testing.T) {
	var logs bytes.Buffer
	failure := errors.New("private upstream error")
	transport := timingTransport{next: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, failure }), logger: slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPatch, "https://www.googleapis.com/upload/drive/v3/files/private-id", nil)
	_, err := transport.RoundTrip(req)
	if !errors.Is(err, failure) {
		t.Fatal("error identity changed")
	}
	if strings.Contains(logs.String(), failure.Error()) || !strings.Contains(logs.String(), `"outcome":"transport_error"`) {
		t.Fatal(logs.String())
	}
}

func TestTimingDisabledPreservesBody(t *testing.T) {
	body := io.NopCloser(strings.NewReader("private body"))
	transport := timingTransport{next: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
	}), logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://www.googleapis.com/drive/v3/files", nil)
	response, err := transport.RoundTrip(req)
	if err != nil || response.Body != body {
		t.Fatal("disabled timings must leave response unchanged")
	}
	_ = response.Body.Close()
}
