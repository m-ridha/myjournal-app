package web

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/config"
	provider "github/mridha/myjournal-app/internal/core/connection"
	"github/mridha/myjournal-app/internal/core/journal"
	"github/mridha/myjournal-app/internal/driver/web/authsession"
	"github/mridha/myjournal-app/internal/testutil"
)

type disconnectProvider struct {
	provider.Provider
	err        error
	calls      int
	connection provider.Connection
}

func (*disconnectProvider) ID() string { return "local" }
func (p *disconnectProvider) Revoke(_ context.Context, c provider.Connection) error {
	p.calls++
	p.connection = c
	return p.err
}

func TestDisconnectRevokesBeforeClearingSession(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{"success", nil, 204},
		{"unavailable", fmt.Errorf("private adapter context: %w", journal.ErrUnavailable), 503},
		{"reconnect", fmt.Errorf("private adapter context: %w", journal.ErrAuthRequired), 401},
		{"unexpected", fmt.Errorf("private adapter failure"), 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &disconnectProvider{err: tc.err}
			cfg := config.Config{SessionKey: "test-key", CookieSecure: true}
			router := newTestRouter(cfg, provider.NewRegistry(p), testutil.StubClock{Time: now}, nil)
			r := httptest.NewRequest(http.MethodPost, "/api/v1/disconnect", nil)
			addLocalSessionCookie(t, r, cfg.SessionKey, now)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != tc.status || p.calls != 1 || p.connection.AccountID != "dev" {
				t.Fatalf("status=%d revoke calls=%d", w.Code, p.calls)
			}
			cookies := w.Result().Cookies()
			if tc.err != nil {
				checkAPIError(t, w, tc.status)
				if len(cookies) != 0 || strings.Contains(w.Body.String(), "private") {
					t.Fatal("failure cleared session or exposed adapter details")
				}
			} else {
				if w.Body.Len() != 0 || len(cookies) != 1 {
					t.Fatal("disconnect must return empty 204 and clear session")
				}
				cookie := cookies[0]
				if cookie.Name != authsession.CookieName || cookie.MaxAge != -1 || cookie.Value != "" || cookie.Path != "/" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
					t.Fatal("incorrect cookie removal attributes")
				}
			}
		})
	}
}

func TestDisconnectRejectsUnauthenticatedAndCrossOriginRequests(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name                string
		cookie, crossOrigin bool
		issued              time.Time
		status              int
	}{
		{"missing session", false, false, now, 401},
		{"expired session", true, false, now.Add(-provider.Lifetime), 401},
		{"cross-origin", true, true, now, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &disconnectProvider{}
			cfg := config.Config{SessionKey: "test-key"}
			router := newTestRouter(cfg, provider.NewRegistry(p), testutil.StubClock{Time: now}, nil)
			r := httptest.NewRequest(http.MethodPost, "/api/v1/disconnect", nil)
			if tc.cookie {
				addLocalSessionCookie(t, r, cfg.SessionKey, tc.issued)
			}
			if tc.crossOrigin {
				r.Header.Set("Sec-Fetch-Site", "cross-site")
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			checkAPIError(t, w, tc.status)
			if p.calls != 0 || len(w.Result().Cookies()) != 0 {
				t.Fatal("rejected request mutated provider/session")
			}
		})
	}
}
