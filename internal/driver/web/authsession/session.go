// Package authsession adapts encrypted sessions to browser requests and cookies.
package authsession

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	provider "github/m-ridha/myjournal-app/internal/core/connection"
	"github/m-ridha/myjournal-app/internal/core/journal"
)

// Codec is the cookie serialization capability consumed by web delivery.
// Implementations preserve the authenticated session and login-flow purposes.
type Codec interface {
	Encode(provider.Session) (string, error)
	Decode(string) (provider.Session, error)
	EncodeLoginFlow(provider.LoginFlow) (string, error)
	DecodeLoginFlow(string) (provider.LoginFlow, error)
}

const CookieName = "mj_session"
const LoginCookieName = "mj_oauth"
const LoginCookiePath = "/api/v1/auth/"

const MaxEncodedBytes = 3600
const MaxCookieBytes = 4096

// Read enforces the same session policy for HTML pages and API handlers.
func Read(r *http.Request, codec Codec, registry provider.Catalog, now time.Time) (provider.Session, error) {
	cookie, err := r.Cookie(CookieName)
	if err != nil || len(cookie.Value) > MaxCookieBytes {
		return provider.Session{}, journal.ErrAuthRequired
	}
	current, err := codec.Decode(cookie.Value)
	if err != nil || !current.ValidAt(now) {
		return provider.Session{}, journal.ErrAuthRequired
	}
	if _, ok := registry.Lookup(current.ProviderID); !ok {
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

// DraftScope identifies a provider/account independently of credentials and labels.
func DraftScope(current provider.Session) string {
	identity, _ := json.Marshal([2]string{current.ProviderID, current.AccountID})
	sum := sha256.Sum256(identity)
	return hex.EncodeToString(sum[:])
}
