package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func TestRecovererOmitsPanicDetails(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	secret := "private-cookie private-code private-journal-text"
	h := chimiddleware.RequestID(Recoverer(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "generic failure", http.StatusInternalServerError)
	})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(secret) })))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries/2020-01-01?code=private-code", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusInternalServerError || res.Body.String() != "generic failure\n" {
		t.Fatalf("Recoverer(panic) = (%d, %q), want (500, generic failure)", res.Code, res.Body.String())
	}
	if strings.Contains(logs.String(), "private-") || !strings.Contains(logs.String(), `"msg":"request panic"`) || !strings.Contains(logs.String(), `"request_id":`) {
		t.Fatalf("Recoverer(panic) logs = %q, want generic event/request ID without private data", logs.String())
	}
}

func TestRecovererPreservesAbort(t *testing.T) {
	h := Recoverer(func(http.ResponseWriter, *http.Request) { t.Error("abort invoked responder") })(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }))
	defer func() {
		if got := recover(); got != http.ErrAbortHandler {
			t.Errorf("Recoverer(abort) panic = %v, want ErrAbortHandler", got)
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestRecovererAbortsCommittedResponse(t *testing.T) {
	h := Recoverer(func(http.ResponseWriter, *http.Request) { t.Error("committed response invoked responder") })(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("partial"))
		panic("private value")
	}))
	res := httptest.NewRecorder()
	defer func() {
		if got := recover(); got != http.ErrAbortHandler {
			t.Errorf("Recoverer(committed) panic = %v, want ErrAbortHandler", got)
		}
		if res.Body.String() != "partial" {
			t.Errorf("Recoverer(committed) body = %q, want partial", res.Body.String())
		}
	}()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))
}
