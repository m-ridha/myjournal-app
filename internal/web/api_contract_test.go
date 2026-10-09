package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/config"
	"github/mridha/myjournal-app/internal/journal"
	"github/mridha/myjournal-app/internal/provider"
	"github/mridha/myjournal-app/internal/provider/local"
	"github/mridha/myjournal-app/internal/session"
	"github/mridha/myjournal-app/internal/web/api"
)

func TestAPIRouterErrorEnvelopes(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	cfg := config.Config{BaseURL: "http://localhost:3030", SessionKey: "test-key"}
	registry := provider.NewRegistry(local.New(cfg.BaseURL, t.TempDir()))
	router := NewRouter(cfg, registry, mockClock{now: now}, nil)
	for _, tc := range []struct {
		method, path, site string
		status             int
	}{
		{"GET", "/api/v1/entries/invalid", "", 400},
		{"GET", "/api/v1/entries?page=no", "", 400},
		{"GET", "/api/v1/calendar", "", 400},
		{"GET", "/api/v1/settings", "", 401},
		{"GET", "/api/v1/unknown", "", 404},
		{"PATCH", "/api/v1/entries/2026-10-09", "", 405},
		{"PUT", "/api/v1/entries/2026-10-09", "cross-site", 403},
		{"GET", "/api/v1/auth/google/login", "", 404},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)
			r.Header.Set("Sec-Fetch-Site", tc.site)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			checkAPIError(t, w, tc.status)
		})
	}
}

func TestAPIRouterWrappedProviderFailures(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	cfg := config.Config{BaseURL: "http://localhost:3030", SessionKey: "test-key"}
	for _, tc := range []struct {
		err    error
		status int
	}{{journal.ErrAuthRequired, 401}, {journal.ErrUnavailable, 503}, {fmt.Errorf("private adapter failure"), 500}} {
		factory := func(context.Context, session.Session) (*journal.Service, error) {
			return nil, fmt.Errorf("sensitive context: %w", tc.err)
		}
		router := NewRouter(cfg, provider.NewRegistry(local.New(cfg.BaseURL, t.TempDir())), mockClock{now: now}, factory)
		for _, route := range []struct{ method, path string }{{"GET", "/api/v1/entries"}, {"GET", "/api/v1/entries/2026-10-09"}, {"PUT", "/api/v1/entries/2026-10-09"}, {"DELETE", "/api/v1/entries/2026-10-09"}, {"GET", "/api/v1/calendar?month=2026-10"}} {
			r := httptest.NewRequest(route.method, route.path, strings.NewReader(`{"text":"","baseVersion":""}`))
			addLocalSessionCookie(t, r, cfg.SessionKey, now)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			checkAPIError(t, w, tc.status)
			if strings.Contains(w.Body.String(), "sensitive") || strings.Contains(w.Body.String(), "private") {
				t.Fatal("adapter details exposed")
			}
		}
	}
}

func checkAPIError(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	if w.Code != status || w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("status=%d type=%q body=%s", w.Code, w.Header().Get("Content-Type"), w.Body.String())
	}
	var body api.Error
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Message == "" {
		t.Fatalf("invalid error envelope: %s", w.Body.String())
	}
}
