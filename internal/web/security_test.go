package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/config"
	"github/mridha/myjournal-app/internal/provider"
	localProvider "github/mridha/myjournal-app/internal/provider/local"
	"github/mridha/myjournal-app/internal/session"
)

func TestExpiredSessionsRejectedByPagesAndAPI(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	cfg := config.Config{BaseURL: "http://localhost:3030", SessionKey: "test-key"}
	router := NewRouter(cfg, provider.NewRegistry(localProvider.New(cfg.BaseURL)), mockClock{now: now}, nil)
	for _, issued := range []time.Time{now.Add(-session.Lifetime), now.Add(time.Second), {}} {
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
	cfg := config.Config{BaseURL: "http://localhost:3030", SessionKey: "test-key"}
	router := NewRouter(cfg, provider.NewRegistry(localProvider.New(cfg.BaseURL)), mockClock{now: time.Now()}, nil)
	for _, tc := range []struct{ method, path string }{{"POST", "/api/v1/logout"}, {"PUT", "/api/v1/entries/2026-10-09"}, {"DELETE", "/api/v1/entries/2026-10-09"}} {
		for _, header := range []string{"Origin", "Sec-Fetch-Site"} {
			req := httptest.NewRequest(tc.method, "http://localhost:3030"+tc.path, nil)
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
		req := httptest.NewRequest(http.MethodPost, "http://localhost:3030/api/v1/logout", nil)
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
	cfg := config.Config{BaseURL: "http://localhost:3030", SessionKey: "test-key", CookieSecure: true}
	router := NewRouter(cfg, provider.NewRegistry(localProvider.New(cfg.BaseURL)), mockClock{now: now}, nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/auth/local/callback", nil))
	cookies := res.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies=%d", len(cookies))
	}
	cookie := cookies[0]
	if !cookie.Expires.Equal(now.Add(session.Lifetime)) || cookie.MaxAge != int(session.Lifetime.Seconds()) || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("incorrect cookie attributes")
	}
	decoded, err := session.Decode(cfg.SessionKey, cookie.Value)
	if err != nil || !decoded.IssuedAt.Equal(now) {
		t.Fatalf("issuance mismatch: %v", err)
	}
}
