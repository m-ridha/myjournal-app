// Package authsession adapts encrypted sessions to browser requests and cookies.
package authsession

import (
	"context"
	"net/http"
	"time"

	"github/mridha/myjournal-app/internal/journal"
	"github/mridha/myjournal-app/internal/provider"
	"github/mridha/myjournal-app/internal/session"
)

const MaxEncodedBytes = 3600
const MaxCookieBytes = 4096

// Read enforces the same session policy for HTML pages and API handlers.
func Read(r *http.Request, secret string, registry *provider.Registry, now time.Time) (session.Session, error) {
	cookie, err := r.Cookie(session.CookieName)
	if err != nil || len(cookie.Value) > MaxCookieBytes {
		return session.Session{}, journal.ErrAuthRequired
	}
	current, err := session.Decode(secret, cookie.Value)
	if err != nil || !current.ValidAt(now) {
		return session.Session{}, journal.ErrAuthRequired
	}
	if _, ok := registry.Get(current.ProviderID); !ok {
		return session.Session{}, journal.ErrAuthRequired
	}
	return current, nil
}

// Cookie builds an authenticated session cookie; an empty value clears it.
func Cookie(value string, issuedAt time.Time, secure bool) *http.Cookie {
	c := &http.Cookie{Name: session.CookieName, Value: value, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure, MaxAge: -1}
	if value != "" {
		c.Expires = issuedAt.Add(session.Lifetime)
		c.MaxAge = int(session.Lifetime.Seconds())
	}
	return c
}

// FolderURL queries optional provider navigation without creating journal paths.
func FolderURL(ctx context.Context, p provider.Provider, c provider.Connection) (string, error) {
	if locator, ok := p.(provider.FolderLocator); ok {
		return locator.FolderURL(ctx, c)
	}
	return "", nil
}
