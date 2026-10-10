package authsession

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/provider"
	"github/mridha/myjournal-app/internal/session"
)

type stubProvider struct{ provider.Provider }

func (stubProvider) ID() string { return provider.LocalID }

func TestSessionPolicyAndCookieLifecycle(t *testing.T) {
	now := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	secret := "synthetic-key"
	current := session.Session{Connection: provider.Connection{ProviderID: provider.LocalID, AccountID: "test"}, IssuedAt: now}
	value, err := session.Encode(secret, current)
	if err != nil {
		t.Fatal(err)
	}
	registry := provider.NewRegistry(stubProvider{})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(Cookie(value, now, true))
	if _, err := Read(r, secret, registry, now); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(r, secret, registry, now.Add(session.Lifetime)); err == nil {
		t.Fatal("expired session accepted")
	}
	if _, err := Read(r, secret, provider.NewRegistry(), now); err == nil {
		t.Fatal("disabled provider accepted")
	}
	for _, v := range []string{value, ""} {
		c := Cookie(v, now, true)
		if !c.Secure || !c.HttpOnly || c.Path != "/" || c.SameSite != http.SameSiteLaxMode {
			t.Fatal("cookie policy drift")
		}
		if v == "" && c.MaxAge != -1 {
			t.Fatal("clear cookie did not expire")
		}
	}
}
