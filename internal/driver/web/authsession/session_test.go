package authsession

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	provider "github/mridha/myjournal-app/internal/core/connection"
	"github/mridha/myjournal-app/internal/core/journal"
	"github/mridha/myjournal-app/internal/driven/session"
)

func TestSessionPolicyAndCookieLifecycle(t *testing.T) {
	now := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	secret := "synthetic-key"
	current := provider.Session{Connection: provider.Connection{ProviderID: provider.LocalID, AccountID: "test"}, IssuedAt: now}
	value, err := session.Encode(secret, current)
	require.NoError(t, err)
	p := newMockProvider(t)
	p.EXPECT().ID().Return(provider.LocalID)
	registry := provider.NewRegistry(p)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(Cookie(value, now, true))
	_, err = Read(r, session.NewCodec(secret), registry, now)
	require.NoError(t, err)
	if _, err := Read(r, session.NewCodec(secret), registry, now.Add(provider.Lifetime)); err == nil {
		t.Fatal("expired session accepted")
	}
	if _, err := Read(r, session.NewCodec(secret), provider.NewRegistry(), now); err == nil {
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

func TestReadMapsInjectedCodecFailureToAuthenticationRequired(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: CookieName, Value: "synthetic"})
	codec := newMockCodec(t)
	codec.EXPECT().Decode("synthetic").Return(provider.Session{}, errors.New("synthetic decoding failure")).Once()
	_, err := Read(request, codec, provider.NewRegistry(), time.Now())
	if !errors.Is(err, journal.ErrAuthRequired) {
		t.Fatalf("error = %v, want authentication required", err)
	}
}
