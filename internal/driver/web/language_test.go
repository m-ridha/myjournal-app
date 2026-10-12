package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github/mridha/myjournal-app/internal/config"
	provider "github/mridha/myjournal-app/internal/core/connection"
	"github/mridha/myjournal-app/internal/driven/clock"
	localProvider "github/mridha/myjournal-app/internal/driven/provider/local"
	"github/mridha/myjournal-app/internal/testutil"
)

func TestLanguageSelection(t *testing.T) {
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	h := newTestRouter(cfg, provider.NewRegistry(localProvider.New(testutil.LocalBaseURL, "")), clock.System{}, nil)
	for _, tc := range []struct{ locale, lang, title string }{
		{"", "en", "Help"}, {"en", "en", "Help"}, {"id", "id", "Bantuan"}, {"fr", "en", "Help"},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/help", nil)
			req.AddCookie(&http.Cookie{Name: "mj_language", Value: tc.locale})
			res := httptest.NewRecorder()
			h.ServeHTTP(res, req)
			if res.Code != 200 || !strings.Contains(res.Body.String(), `<html lang="`+tc.lang+`"`) || !strings.Contains(res.Body.String(), "<h1>"+tc.title+"</h1>") {
				t.Fatalf("GET /help language=%q: status=%d, missing localized HTML", tc.locale, res.Code)
			}
		})
	}
	for _, locale := range []string{"en", "id", "fr"} {
		req := httptest.NewRequest(http.MethodPost, "/settings/language", strings.NewReader(url.Values{"language": {locale}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		addLocalSessionCookie(t, req, cfg.SessionKey)
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		if locale == "fr" {
			if res.Code != 400 || len(res.Result().Cookies()) != 0 {
				t.Fatalf("invalid language: status=%d cookies=%v", res.Code, res.Result().Cookies())
			}
			continue
		}
		if res.Code != 303 || res.Header().Get("Location") != "/settings" {
			t.Fatalf("language submission: status=%d location=%q", res.Code, res.Header().Get("Location"))
		}
		cookies := res.Result().Cookies()
		if len(cookies) != 1 || cookies[0].Name != "mj_language" || cookies[0].Value != locale || cookies[0].Path != "/" || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode || cookies[0].MaxAge <= 0 {
			t.Fatalf("language cookie = %v", cookies)
		}
		req = httptest.NewRequest(http.MethodGet, "/settings", nil)
		addLocalSessionCookie(t, req, cfg.SessionKey)
		req.AddCookie(cookies[0])
		res = httptest.NewRecorder()
		h.ServeHTTP(res, req)
		if !strings.Contains(res.Body.String(), `value="`+locale+`" selected`) || !strings.Contains(res.Body.String(), "mj-reading-page") {
			t.Fatal("settings missing selected language or reading layout")
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/settings/language", strings.NewReader("language=id"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != 303 || res.Header().Get("Location") != "/" || len(res.Result().Cookies()) != 0 {
		t.Fatal("signed-out language update must require a session")
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	req.AddCookie(&http.Cookie{Name: "mj_language", Value: "id"})
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != 404 || !strings.Contains(res.Body.String(), "Permintaan tidak valid.") {
		t.Fatal("API error was not localized")
	}
}

func TestLanguagePreferenceSafety(t *testing.T) {
	cfg := config.Config{BaseURL: "https://example.com", SessionKey: "0123456789abcdef0123456789abcdef", CookieSecure: true}
	h := newTestRouter(cfg, provider.NewRegistry(localProvider.New(testutil.LocalBaseURL, "")), clock.System{}, nil)
	for _, tc := range []struct {
		name, body, origin string
		status             int
	}{
		{"valid", "language=id", "", 303},
		{"cross origin", "language=id", "https://other.example", 403},
		{"duplicate", "language=en&language=id", "", 400},
		{"missing", "", "", 400},
		{"malformed", "language=%zz", "", 400},
		{"oversized", "language=" + strings.Repeat("a", 2048), "", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "https://example.com/settings/language", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			addLocalSessionCookie(t, req, cfg.SessionKey)
			res := httptest.NewRecorder()
			h.ServeHTTP(res, req)
			if res.Code != tc.status {
				t.Fatalf("POST language %s: status=%d, want %d", tc.name, res.Code, tc.status)
			}
			if tc.status == 303 {
				cookies := res.Result().Cookies()
				if len(cookies) != 1 || !cookies[0].Secure {
					t.Fatal("HTTPS preference cookie must be Secure")
				}
			} else if len(res.Result().Cookies()) != 0 {
				t.Fatal("rejected request set a preference cookie")
			}
		})
	}
}

func TestLanguageRequestsRemainIsolated(t *testing.T) {
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "0123456789abcdef0123456789abcdef"}
	h := newTestRouter(cfg, provider.NewRegistry(localProvider.New(testutil.LocalBaseURL, "")), clock.System{}, nil)
	for i := 0; i < 20; i++ {
		for _, locale := range []string{"en", "id"} {
			t.Run(fmt.Sprintf("%s-%d", locale, i), func(t *testing.T) {
				t.Parallel()
				req := httptest.NewRequest(http.MethodGet, "/help", nil)
				req.AddCookie(&http.Cookie{Name: "mj_language", Value: locale})
				res := httptest.NewRecorder()
				h.ServeHTTP(res, req)
				if !strings.Contains(res.Body.String(), `<html lang="`+locale+`"`) {
					t.Fatalf("request language %s leaked another catalog", locale)
				}
			})
		}
	}
}
