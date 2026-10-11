package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/config"
	provider "github/mridha/myjournal-app/internal/core/connection"
	"github/mridha/myjournal-app/internal/core/journal"
	"github/mridha/myjournal-app/internal/driven/clock"
	googleProvider "github/mridha/myjournal-app/internal/driven/provider/google"
	localProvider "github/mridha/myjournal-app/internal/driven/provider/local"
	"github/mridha/myjournal-app/internal/driven/session"
	"github/mridha/myjournal-app/internal/driver/web/authsession"
	"github/mridha/myjournal-app/internal/testutil"
)

func TestLocalLoginRedirect(t *testing.T) {
	registry := provider.NewRegistry(localProvider.New(testutil.LocalBaseURL, ""))
	h := newTestRouter(config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}, registry, clock.System{}, nil)

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
	registry := provider.NewRegistry(localProvider.New(testutil.LocalBaseURL, ""))
	h := newTestRouter(config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}, registry, clock.System{}, nil)

	req := httptest.NewRequest(http.MethodGet, localProvider.CallbackPath, nil)
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
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	h := newTestRouter(cfg, provider.NewRegistry(localProvider.New(testutil.LocalBaseURL, "")), clock.System{}, nil)

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

func TestHomepageIncludesOwnershipVerification(t *testing.T) {
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	h := newTestRouter(cfg, provider.NewRegistry(), clock.System{}, nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))
	body := res.Body.String()
	end := strings.Index(body, "</head>")
	if end < 0 || !strings.Contains(body[:end], `<meta name="google-site-verification" content="cZ5rwhKdK0RE8qM3-0wr6T4V96RMU5mjUUCys4EhIdk" />`) {
		t.Fatal("homepage head missing Search Console verification tag")
	}
}

func TestPrivatePagesRedirectWithoutSession(t *testing.T) {
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	h := newTestRouter(cfg, provider.NewRegistry(localProvider.New(testutil.LocalBaseURL, "")), clock.System{}, nil)
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
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	h := newTestRouter(cfg, provider.NewRegistry(localProvider.New(testutil.LocalBaseURL, "")), clock.System{}, nil)

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
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	h := newTestRouter(cfg, provider.NewRegistry(localProvider.New(testutil.LocalBaseURL, "")), clock.System{}, nil)

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
	if got := res.Body.String(); !strings.Contains(got, `data-entries-status>Loading…</p>`) {
		t.Fatal("initial loading message must belong to the shared live status region")
	}
	if got := strings.Count(res.Body.String(), ">Loading…</p>"); got != 1 {
		t.Fatalf("loading message count = %d, want 1", got)
	}

}

func TestCalendarPageRendersTemplate(t *testing.T) {
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	h := newTestRouter(cfg, provider.NewRegistry(localProvider.New(testutil.LocalBaseURL, "")), clock.System{}, nil)

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
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	h := newTestRouter(cfg, provider.NewRegistry(localProvider.New(testutil.LocalBaseURL, "")), clock.System{}, nil)

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
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	h := newTestRouter(cfg, provider.NewRegistry(localProvider.New(testutil.LocalBaseURL, "")), testutil.StubClock{Time: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)}, nil)

	req := httptest.NewRequest(http.MethodGet, "/entries/2026-09-22", nil)
	addLocalSessionCookie(t, req, cfg.SessionKey, time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC))
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
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	h := newTestRouter(cfg, provider.NewRegistry(localProvider.New(testutil.LocalBaseURL, "")), clock.System{}, nil)

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
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	h := newTestRouter(cfg, provider.NewRegistry(localProvider.New(testutil.LocalBaseURL, "")), clock.System{}, nil)

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
	a := provider.Session{Connection: provider.Connection{ProviderID: "local", AccountID: "dev", Credentials: "private-credentials"}}
	b := a
	b.AccountID = "another-account"
	c := a
	c.ProviderID = "google"
	if draftScope(a) == draftScope(b) || draftScope(a) == draftScope(c) {
		t.Fatal("draft scopes must separate accounts and providers")
	}
	res := httptest.NewRecorder()
	clk := testutil.StubClock{Time: time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)}
	testRenderer(t).renderEntryPage(res, clk.Now(), clk, a, time.UTC)
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

func addLocalSessionCookie(t *testing.T, req *http.Request, secret string, issued ...time.Time) {
	when := time.Now().UTC()
	if len(issued) > 0 {
		when = issued[0]
	}
	t.Helper()
	encoded, err := session.Encode(secret, provider.Session{
		Connection: provider.Connection{ProviderID: "local", AccountID: "dev", AccountEmail: "dev@localhost", Credentials: "local-dev-user"},
		IssuedAt:   when,
	})
	if err != nil {
		t.Fatalf("session.Encode() error = %v", err)
	}
	req.AddCookie(&http.Cookie{Name: authsession.CookieName, Value: encoded})
}

func TestEntryPagesUseRequestTimeZone(t *testing.T) {
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	router := newTestRouter(cfg, provider.NewRegistry(localProvider.New(cfg.BaseURL, "")), testutil.StubClock{Time: time.Date(2026, 10, 7, 19, 0, 0, 0, time.UTC)}, nil)
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
			addLocalSessionCookie(t, req, cfg.SessionKey, time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC))
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

func TestLandingOffersOnlyRegisteredProviders(t *testing.T) {
	local := localProvider.New(testutil.LocalBaseURL, "")
	google := googleProvider.New(googleProvider.Config{ClientID: "test-client", ClientSecret: "test-secret", BaseURL: testutil.LocalBaseURL})
	for _, tc := range []struct {
		name          string
		providers     []provider.Provider
		local, google bool
	}{
		{"local", []provider.Provider{local}, true, false},
		{"google", []provider.Provider{google}, false, true},
		{"both", []provider.Provider{local, google}, true, true},
		{"none", nil, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestRouter(config.Config{SessionKey: "test-key"}, provider.NewRegistry(tc.providers...), testutil.StubClock{Time: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)}, nil)
			response := httptest.NewRecorder()
			h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
			body := response.Body.String()
			if got := strings.Contains(body, `href="/api/v1/auth/local/login"`); got != tc.local {
				t.Errorf("local login offered=%v want=%v", got, tc.local)
			}
			if got := strings.Contains(body, `href="/api/v1/auth/google/login"`); got != tc.google {
				t.Errorf("Google login offered=%v want=%v", got, tc.google)
			}
			if tc.google && !strings.Contains(body, ">Continue with Google</a>") {
				t.Error("missing localized Google button")
			}
		})
	}
}

func TestAPIPanicReturnsGenericEnvelope(t *testing.T) {
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	h := newTestRouter(cfg, provider.NewRegistry(localProvider.New(testutil.LocalBaseURL, "")), clock.System{}, func(context.Context, provider.Session) (*journal.Service, error) {
		panic("private provider details")
	})
	login := httptest.NewRecorder()
	h.ServeHTTP(login, httptest.NewRequest(http.MethodGet, localProvider.CallbackPath, nil))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries/2020-01-01", nil)
	for _, cookie := range login.Result().Cookies() {
		req.AddCookie(cookie)
	}
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	var body map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("GET panicking API decode = %v, want JSON envelope", err)
	}
	if res.Code != http.StatusInternalServerError || body["message"] != "Something went wrong. Please try again." || len(body) != 1 || res.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("GET panicking API = (%d, %v, %q), want generic JSON 500", res.Code, body, res.Header().Get("Content-Type"))
	}
}

func TestPublicPagesSettingsLinkFollowsSession(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "public-page-test-key"}
	registry := provider.NewRegistry(localProvider.New(cfg.BaseURL, ""))
	router := newTestRouter(cfg, registry, testutil.StubClock{Time: now}, nil)
	for _, tc := range []struct {
		name         string
		issuedAt     time.Time
		invalid      bool
		wantSettings bool
	}{
		{name: "signed out"},
		{name: "valid session", issuedAt: now, wantSettings: true},
		{name: "expired session", issuedAt: now.Add(-provider.Lifetime)},
		{name: "invalid cookie", invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, path := range []string{"/privacy", "/help", "/terms"} {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				if !tc.issuedAt.IsZero() {
					encoded, err := session.Encode(cfg.SessionKey, provider.Session{
						Connection: provider.Connection{ProviderID: provider.LocalID, AccountID: "dev"},
						IssuedAt:   tc.issuedAt,
					})
					if err != nil {
						t.Fatal(err)
					}
					req.AddCookie(&http.Cookie{Name: authsession.CookieName, Value: encoded})
				} else if tc.invalid {
					req.AddCookie(&http.Cookie{Name: authsession.CookieName, Value: "invalid"})
				}
				res := httptest.NewRecorder()
				router.ServeHTTP(res, req)
				if res.Code != http.StatusOK {
					t.Fatalf("%s status = %d", path, res.Code)
				}
				if got := strings.Contains(res.Body.String(), `href="/settings"`); got != tc.wantSettings {
					t.Errorf("%s Settings link = %v, want %v", path, got, tc.wantSettings)
				}
			}
		})
	}
}

func TestEditorRejectsYearZero(t *testing.T) {
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "synthetic-test-key"}
	router := newTestRouter(cfg, provider.NewRegistry(localProvider.New(cfg.BaseURL, "")), clock.System{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/entries/0000-01-01", nil)
	addLocalSessionCookie(t, req, cfg.SessionKey)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound || !strings.Contains(res.Body.String(), "Page not found") {
		t.Fatal("year zero did not use styled 404")
	}
}

func TestPublicBrandingDisclosuresWithoutSession(t *testing.T) {
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	h := newTestRouter(cfg, provider.NewRegistry(), clock.System{}, nil)
	for path, required := range map[string][]string{
		"/":        {"myjournal", "one plain-text entry per day", `href="/privacy"`, "drive.file"},
		"/terms":   {"Terms of Service", "Your writing", "Availability", "support@myjournal.ink", `href="/privacy"`},
		"/help":    {"Saving and recovery", "Conflicting changes", "Disconnect", "support@myjournal.ink"},
		"/privacy": {"Privacy policy", "Data we access", "How we use", "30 days", "Railway", "Limited Use", "Disconnect", "support@myjournal.ink"},
	} {
		t.Run(path, func(t *testing.T) {
			res := httptest.NewRecorder()
			h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
			if res.Code != http.StatusOK {
				t.Fatalf("status = %d", res.Code)
			}
			body := res.Body.String()
			start := strings.Index(body, `<main`)
			end := strings.Index(body, `</main>`)
			if start < 0 || end < start {
				t.Fatal("missing main content")
			}
			body = body[start:end]
			for _, want := range required {
				if !strings.Contains(body, want) {
					t.Errorf("public content missing %q", want)
				}
			}
		})
	}
}
