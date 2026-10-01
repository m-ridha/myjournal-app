package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/clock"
	"github/mridha/myjournal-app/internal/config"
	"github/mridha/myjournal-app/internal/provider"
	localProvider "github/mridha/myjournal-app/internal/provider/local"
)

type mockClock struct{ now time.Time }

func (m mockClock) Now() time.Time { return m.now }

func TestLocalLoginRedirect(t *testing.T) {
	registry := provider.NewRegistry(localProvider.New("http://localhost:3030"))
	h := NewRouter(config.Config{BaseURL: "http://localhost:3030", SessionKey: "0123456789abcdef0123456789abcdef"}, registry, clock.System{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/local/login", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusFound)
	}
	if got := res.Header().Get("Location"); got != "http://localhost:3030/api/v1/auth/local/callback" {
		t.Fatalf("Location = %q, want %q", got, "http://localhost:3030/api/v1/auth/local/callback")
	}
}

func TestLocalCallbackCreatesSessionCookie(t *testing.T) {
	registry := provider.NewRegistry(localProvider.New("http://localhost:3030"))
	h := NewRouter(config.Config{BaseURL: "http://localhost:3030", SessionKey: "0123456789abcdef0123456789abcdef"}, registry, clock.System{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/local/callback", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusFound)
	}
	if got := res.Result().Cookies(); len(got) == 0 {
		t.Fatal("expected a session cookie from local callback")
	}
}

func TestLandingPageRendersTemplate(t *testing.T) {
	cfg := config.Config{BaseURL: "http://localhost:3030", SessionKey: "0123456789abcdef0123456789abcdef"}
	h := NewRouter(cfg, provider.NewRegistry(localProvider.New("http://localhost:3030")), clock.System{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if got := res.Body.String(); !strings.Contains(got, "A quiet place to write") {
		t.Fatalf("landing page body = %q, want quiet writing copy", got)
	}
	if got := res.Body.String(); !strings.Contains(got, `data-messages="`) || !strings.Contains(got, "calendar.dayLabel") {
		t.Fatalf("landing page does not expose the locale catalog to scripts")
	}
}

func TestPrivacyPageRendersTemplate(t *testing.T) {
	cfg := config.Config{BaseURL: "http://localhost:3030", SessionKey: "0123456789abcdef0123456789abcdef"}
	h := NewRouter(cfg, provider.NewRegistry(localProvider.New("http://localhost:3030")), clock.System{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/privacy", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if got := res.Body.String(); !strings.Contains(got, "Privacy") {
		t.Fatalf("privacy page body = %q, want Privacy heading", got)
	}
}

func TestEntriesPageRendersTemplate(t *testing.T) {
	cfg := config.Config{BaseURL: "http://localhost:3030", SessionKey: "0123456789abcdef0123456789abcdef"}
	h := NewRouter(cfg, provider.NewRegistry(localProvider.New("http://localhost:3030")), clock.System{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/entries", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if got := res.Body.String(); !strings.Contains(got, "Entries") {
		t.Fatalf("entries page body = %q, want Entries heading", got)
	}
}

func TestCalendarPageRendersTemplate(t *testing.T) {
	cfg := config.Config{BaseURL: "http://localhost:3030", SessionKey: "0123456789abcdef0123456789abcdef"}
	h := NewRouter(cfg, provider.NewRegistry(localProvider.New("http://localhost:3030")), clock.System{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/calendar", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if got := res.Body.String(); !strings.Contains(got, "Calendar") {
		t.Fatalf("calendar page body = %q, want Calendar heading", got)
	}
}

func TestDateEntryPageRendersTemplate(t *testing.T) {
	cfg := config.Config{BaseURL: "http://localhost:3030", SessionKey: "0123456789abcdef0123456789abcdef"}
	h := NewRouter(cfg, provider.NewRegistry(localProvider.New("http://localhost:3030")), clock.System{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/entries/2026-09-20", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if got := res.Body.String(); !strings.Contains(got, "Sunday, 20 September 2026") {
		t.Fatalf("entry page body = %q, want the date heading", got)
	}
}

func TestFutureDateEntryPageIsRejected(t *testing.T) {
	cfg := config.Config{BaseURL: "http://localhost:3030", SessionKey: "0123456789abcdef0123456789abcdef"}
	h := NewRouter(cfg, provider.NewRegistry(localProvider.New("http://localhost:3030")), mockClock{now: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)}, nil)

	req := httptest.NewRequest(http.MethodGet, "/entries/2026-09-22", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusNotFound)
	}
	if got := res.Body.String(); !strings.Contains(got, "Page not found") {
		t.Fatalf("future date page body = %q, want branded 404 copy", got)
	}
}

func TestNotFoundPageRendersTemplate(t *testing.T) {
	cfg := config.Config{BaseURL: "http://localhost:3030", SessionKey: "0123456789abcdef0123456789abcdef"}
	h := NewRouter(cfg, provider.NewRegistry(localProvider.New("http://localhost:3030")), clock.System{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/definitely-missing", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusNotFound)
	}
	if got := res.Body.String(); !strings.Contains(got, "Page not found") {
		t.Fatalf("not found page body = %q, want not found copy", got)
	}
}

func TestSettingsPageRendersTemplate(t *testing.T) {
	cfg := config.Config{BaseURL: "http://localhost:3030", SessionKey: "0123456789abcdef0123456789abcdef"}
	h := NewRouter(cfg, provider.NewRegistry(localProvider.New("http://localhost:3030")), clock.System{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if got := res.Body.String(); !strings.Contains(got, "Settings") {
		t.Fatalf("settings page body = %q, want Settings heading", got)
	}
}
