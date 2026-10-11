// Package authsession adapts encrypted sessions to browser requests and cookies.
package authsession

import (
	"net/http"
	"time"

	provider "github/mridha/myjournal-app/internal/core/connection"
	"github/mridha/myjournal-app/internal/core/journal"
	"github/mridha/myjournal-app/internal/driven/session"
)

const CookieName = "mj_session"
const LoginCookieName = "mj_oauth"
const LoginCookiePath = "/api/v1/auth/"

const MaxEncodedBytes = 3600
const MaxCookieBytes = 4096

// Read enforces the same session policy for HTML pages and API handlers.
func Read(r *http.Request, secret string, registry *provider.Registry, now time.Time) (provider.Session, error) {
	cookie, err := r.Cookie(CookieName)
	if err != nil || len(cookie.Value) > MaxCookieBytes {
		return provider.Session{}, journal.ErrAuthRequired
	}
	current, err := session.Decode(secret, cookie.Value)
	if err != nil || !current.ValidAt(now) {
		return provider.Session{}, journal.ErrAuthRequired
	}
	if _, ok := registry.Get(current.ProviderID); !ok {
		return provider.Session{}, journal.ErrAuthRequired
	}
	return current, nil
}

// Cookie builds an authenticated session cookie; an empty value clears it.
func Cookie(value string, issuedAt time.Time, secure bool) *http.Cookie {
	c := &http.Cookie{Name: CookieName, Value: value, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure, MaxAge: -1}
	if value != "" {
		c.Expires = issuedAt.Add(provider.Lifetime)
		c.MaxAge = int(provider.Lifetime.Seconds())
	}
	return c
}
