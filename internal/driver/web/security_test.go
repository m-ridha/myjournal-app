package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/config"
	provider "github/mridha/myjournal-app/internal/core/connection"
	localProvider "github/mridha/myjournal-app/internal/driven/provider/local"
	"github/mridha/myjournal-app/internal/driven/session"
	"github/mridha/myjournal-app/internal/testutil"
)

func TestExpiredSessionsRejectedByPagesAndAPI(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "test-key"}
	router := newTestRouter(cfg, provider.NewRegistry(localProvider.New(cfg.BaseURL, "")), testutil.StubClock{Time: now}, nil)
	for _, issued := range []time.Time{now.Add(-provider.Lifetime), now.Add(time.Second), {}} {
		for _, tc := range []struct {
			path   string
			status int
		}{{"/entries/today", 303}, {"/settings", 303}, {"/api/v1/entries/2026-10-09", 401}} {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			addLocalSessionCookie(t, req, cfg.SessionKey, issued)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != tc.status {
				t.Fatalf("issued=%v path=%s status=%d", issued, tc.path, res.Code)
			}
		}
	}
}

func TestCrossOriginMutationsRejectedBeforeCookieChanges(t *testing.T) {
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "test-key"}
	router := newTestRouter(cfg, provider.NewRegistry(localProvider.New(cfg.BaseURL, "")), testutil.StubClock{Time: time.Now()}, nil)
	for _, tc := range []struct{ method, path string }{{"POST", "/api/v1/logout"}, {"PUT", "/api/v1/entries/2026-10-09"}, {"DELETE", "/api/v1/entries/2026-10-09"}} {
		for _, header := range []string{"Origin", "Sec-Fetch-Site"} {
			req := httptest.NewRequest(tc.method, testutil.LocalBaseURL+tc.path, nil)
			if header == "Origin" {
				req.Header.Set(header, "https://other.example")
			} else {
				req.Header.Set(header, "cross-site")
			}
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != 403 || len(res.Result().Cookies()) != 0 {
				t.Fatalf("%s %s status=%d", tc.method, header, res.Code)
			}
		}
	}
	for _, site := range []string{"same-origin", ""} {
		req := httptest.NewRequest(http.MethodPost, testutil.LocalBaseURL+"/api/v1/logout", nil)
		req.Header.Set("Sec-Fetch-Site", site)
		req.Header.Set("Origin", cfg.BaseURL)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != 302 {
			t.Fatalf("same-origin logout status=%d", res.Code)
		}
	}
}

func TestLoginCookieHasFixedExpiry(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "test-key", CookieSecure: true}
	router := newTestRouter(cfg, provider.NewRegistry(localProvider.New(cfg.BaseURL, "")), testutil.StubClock{Time: now}, nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, localProvider.CallbackPath, nil))
	cookies := res.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies=%d", len(cookies))
	}
	cookie := cookies[0]
	if !cookie.Expires.Equal(now.Add(provider.Lifetime)) || cookie.MaxAge != int(provider.Lifetime.Seconds()) || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("incorrect cookie attributes")
	}
	decoded, err := session.Decode(cfg.SessionKey, cookie.Value)
	if err != nil || !decoded.IssuedAt.Equal(now) {
		t.Fatalf("issuance mismatch: %v", err)
	}
}

func TestSecurityHeadersAndExternalScripts(t *testing.T) {
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "test-key"}
	router := newTestRouter(cfg, provider.NewRegistry(localProvider.New(cfg.BaseURL, "")), testutil.StubClock{Time: time.Now()}, nil)
	for _, path := range []string{"/", "/settings", "/missing", "/api/v1/settings", "/static/js/theme-init.js"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		policy := res.Header().Get("Content-Security-Policy")
		for _, directive := range []string{"script-src 'self'", "style-src 'self'", "object-src 'none'", "base-uri 'none'", "frame-ancestors 'none'", "form-action 'self'"} {
			if !strings.Contains(policy, directive) {
				t.Fatalf("%s missing %s", path, directive)
			}
		}
		if strings.Contains(policy, "unsafe-inline") || strings.Contains(policy, "unsafe-eval") {
			t.Fatal("unsafe CSP")
		}
		if res.Header().Get("X-Content-Type-Options") != "nosniff" || res.Header().Get("X-Frame-Options") != "DENY" || res.Header().Get("Referrer-Policy") != "no-referrer" {
			t.Fatalf("missing security headers for %s", path)
		}
		if !strings.HasPrefix(path, "/static/") && res.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("cache allowed for %s", path)
		}
		if path == "/" {
			body := res.Body.String()
			if strings.Contains(body, "<script>") || !strings.Contains(body, `src="/static/js/theme-init.js"`) {
				t.Fatal("theme bootstrap must be external")
			}
			if strings.Index(body, "/static/js/theme-init.js") > strings.Index(body, "/static/css/app.css") {
				t.Fatal("theme must initialize before stylesheet")
			}
		}
		if strings.HasPrefix(path, "/static/") && res.Code != 200 {
			t.Fatal("theme bootstrap asset unavailable")
		}
	}
	req := httptest.NewRequest(http.MethodPost, testutil.LocalBaseURL+"/api/v1/logout", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != 403 || res.Header().Get("Content-Security-Policy") == "" || res.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("cross-origin rejection missing headers")
	}
}

func TestHSTSFollowsConfiguredOrigin(t *testing.T) {
	for _, tc := range []struct {
		name           string
		baseURL        string
		forwardedProto string
		want           string
	}{
		{name: "https behind proxy", baseURL: "https://journal.example", forwardedProto: "http", want: "max-age=31536000"},
		{name: "localhost ignores forwarded https", baseURL: testutil.LocalBaseURL, forwardedProto: "https"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Config{BaseURL: tc.baseURL, SessionKey: "test-key"}
			router := newTestRouter(cfg, provider.NewRegistry(), testutil.StubClock{Time: time.Now()}, nil)
			for _, path := range []string{"/", "/missing", "/api/v1/settings", "/static/js/theme-init.js"} {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				req.Header.Set("X-Forwarded-Proto", tc.forwardedProto)
				res := httptest.NewRecorder()
				router.ServeHTTP(res, req)
				if got := res.Header().Get("Strict-Transport-Security"); got != tc.want {
					t.Errorf("%s HSTS = %q, want %q", path, got, tc.want)
				}
			}
		})
	}
}
