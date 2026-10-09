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
	"github/mridha/myjournal-app/internal/session"
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

func TestPrivatePagesRedirectWithoutSession(t *testing.T) {
	cfg := config.Config{BaseURL: "http://localhost:3030", SessionKey: "0123456789abcdef0123456789abcdef"}
	h := NewRouter(cfg, provider.NewRegistry(localProvider.New("http://localhost:3030")), clock.System{}, nil)
	for _, path := range []string{"/entries", "/entries/today", "/entries/2026-09-20", "/calendar", "/settings"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			res := httptest.NewRecorder()
			h.ServeHTTP(res, req)
			if res.Code != http.StatusSeeOther || res.Header().Get("Location") != "/" {
				t.Fatalf("GET %s = %d, Location %q; want redirect to /", path, res.Code, res.Header().Get("Location"))
			}
		})
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
	addLocalSessionCookie(t, req, cfg.SessionKey)
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
	addLocalSessionCookie(t, req, cfg.SessionKey)
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
	addLocalSessionCookie(t, req, cfg.SessionKey)
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
	addLocalSessionCookie(t, req, cfg.SessionKey)
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
	addLocalSessionCookie(t, req, cfg.SessionKey)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
	if got := res.Body.String(); !strings.Contains(got, "Settings") {
		t.Fatalf("settings page body = %q, want Settings heading", got)
	}
	if !strings.Contains(res.Body.String(), "data-sign-out") || !strings.Contains(res.Body.String(), "data-draft-scope=") {
		t.Fatal("settings must expose account-scoped draft cleanup and sign-out controls")
	}
	if strings.Contains(res.Body.String(), "mj-segmented") || !strings.Contains(res.Body.String(), `href="/settings"`) || !strings.Contains(res.Body.String(), "data-theme-toggle") {
		t.Fatal("settings must use shared header navigation/theme controls without Appearance buttons")
	}
}

func TestEditorDraftScopeIsAccountAndProviderSpecific(t *testing.T) {
	a := session.Session{Connection: provider.Connection{ProviderID: "local", AccountID: "dev", Credentials: "private-credentials"}}
	b := a
	b.AccountID = "another-account"
	c := a
	c.ProviderID = "google"
	if draftScope(a) == draftScope(b) || draftScope(a) == draftScope(c) {
		t.Fatal("draft scopes must separate accounts and providers")
	}
	res := httptest.NewRecorder()
	clk := mockClock{now: time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)}
	renderEntryPage(res, clk.Now(), clk, a, time.UTC)
	body := res.Body.String()
	if !strings.Contains(body, `data-draft-scope="`+draftScope(a)+`"`) || strings.Contains(body, a.Credentials) {
		t.Fatal("editor must expose opaque draft scope without credentials")
	}
	b = a
	b.AccountEmail = "new-label@example.test"
	if draftScope(a) != draftScope(b) {
		t.Fatal("account-label changes must not orphan drafts")
	}
}

func addLocalSessionCookie(t *testing.T, req *http.Request, secret string) {
	t.Helper()
	encoded, err := session.Encode(secret, session.Session{
		Connection: provider.Connection{ProviderID: "local", AccountID: "dev", AccountEmail: "dev@localhost", Credentials: "local-dev-user"},
		IssuedAt:   time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("session.Encode() error = %v", err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: encoded})
}

func TestEntryPagesUseRequestTimeZone(t *testing.T) {
	cfg := config.Config{BaseURL: "http://localhost:3030", SessionKey: "0123456789abcdef0123456789abcdef"}
	router := NewRouter(cfg, provider.NewRegistry(localProvider.New(cfg.BaseURL)), mockClock{now: time.Date(2026, 10, 7, 19, 0, 0, 0, time.UTC)}, nil)
	for _, tc := range []struct {
		path, zone string
		status     int
		date       string
	}{
		{"/entries/today", "Asia%2FJakarta", 200, "2026-10-08"},
		{"/entries/2026-10-08", "Asia%2FJakarta", 200, "2026-10-08"},
		{"/entries/2026-10-09", "Asia%2FJakarta", 404, ""},
		{"/entries/2026-10-08", "America%2FLos_Angeles", 404, ""},
		{"/entries/today", "invalid", 200, "2026-10-07"},
	} {
		t.Run(tc.path+tc.zone, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			addLocalSessionCookie(t, req, cfg.SessionKey)
			req.AddCookie(&http.Cookie{Name: "mj_tz", Value: tc.zone})
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != tc.status {
				t.Fatalf("status=%d want %d", res.Code, tc.status)
			}
			if tc.date != "" && (!strings.Contains(res.Body.String(), `data-entry-date="`+tc.date+`"`) || !strings.Contains(res.Body.String(), `data-today="`+tc.date+`"`)) {
				t.Fatal("editor date and today disagree with request timezone")
			}
		})
	}
}
