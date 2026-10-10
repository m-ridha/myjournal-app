package google

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"testing"

	"github/mridha/myjournal-app/internal/provider"
	"github/mridha/myjournal-app/internal/testutil"
)

func TestAuthorizationHandoff(t *testing.T) {
	p := New(Config{ClientID: "client", ClientSecret: "secret", BaseURL: testutil.LocalBaseURL, Scopes: []string{"openid", "email", "profile", "https://www.googleapis.com/auth/drive.file"}})
	request := provider.LoginRequest{State: "browser-state", Verifier: "persisted-verifier", Nonce: "browser-nonce"}
	raw, err := p.Begin(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	sum := sha256.Sum256([]byte(request.Verifier))
	for key, want := range map[string]string{
		"state": request.State, "nonce": request.Nonce,
		"code_challenge": base64.RawURLEncoding.EncodeToString(sum[:]), "code_challenge_method": "S256",
		"redirect_uri": "http://localhost:3030/api/v1/auth/google/callback",
		"scope":        "openid email profile https://www.googleapis.com/auth/drive.file",
		"access_type":  "offline", "response_type": "code",
	} {
		if q.Get(key) != want {
			t.Errorf("%s mismatch", key)
		}
	}
	request.ForceConsent = true
	forced, err := p.Begin(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	forcedURL, _ := url.Parse(forced)
	if q.Get("prompt") != "" || forcedURL.Query().Get("prompt") != "consent" {
		t.Fatal("consent mode ignored")
	}
	if _, err := p.Begin(context.Background(), provider.LoginRequest{}); err == nil {
		t.Fatal("accepted absent handoff")
	}
	if conn, err := p.Complete(context.Background(), provider.LoginCallback{Code: "code"}); err == nil || conn.AccountID != "" {
		t.Fatal("invalid login handoff manufactured a connection")
	}
	if err := p.Revoke(context.Background(), provider.Connection{}); err == nil {
		t.Fatal("invalid credentials reported revocation success")
	}
}
