package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/journal"
	"github/mridha/myjournal-app/internal/provider"
	"github/mridha/myjournal-app/internal/testutil"

	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
)

func TestExchangeAndRefresh(t *testing.T) {
	refreshes := 0
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Form.Get("grant_type") == "refresh_token" {
			refreshes++
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "renewed", "token_type": "Bearer", "expires_in": 3600})
			return
		}
		if r.Form.Get("code_verifier") != "verifier" {
			t.Error("PKCE verifier missing")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access", "refresh_token": "refresh", "id_token": "signed-token", "token_type": "Bearer", "expires_in": 3600, "scope": driveScope})
	}))
	defer endpoint.Close()
	p := New(Config{ClientID: "client", ClientSecret: "secret", BaseURL: testutil.LocalBaseURL})
	p.endpoint = oauth2.Endpoint{TokenURL: endpoint.URL, AuthStyle: oauth2.AuthStyleInParams}
	p.validate = func(_ context.Context, token, audience string) (*idtoken.Payload, error) {
		if token != "signed-token" || audience != "client" {
			t.Error("validator arguments")
		}
		return &idtoken.Payload{Issuer: "https://accounts.google.com", Audience: "client", Subject: "account", Expires: time.Now().Add(time.Hour).Unix(), IssuedAt: time.Now().Unix(), Claims: map[string]any{"nonce": "nonce", "email": "test@example.com", "email_verified": true}}, nil
	}
	conn, err := p.Complete(context.Background(), provider.LoginCallback{Code: "code", Verifier: "verifier", Nonce: "nonce"})
	if err != nil {
		t.Fatal(err)
	}
	if conn.AccountID != "account" {
		t.Fatal("wrong account")
	}
	creds, err := decodeCredentials(conn)
	if err != nil {
		t.Fatal(err)
	}
	p.dropToken(creds.Refresh)
	for i := 0; i < 2; i++ {
		if _, err := p.accessToken(context.Background(), creds.Refresh); err != nil {
			t.Fatal(err)
		}
	}
	if refreshes != 1 {
		t.Fatalf("refresh requests %d", refreshes)
	}
	p.validate = func(context.Context, string, string) (*idtoken.Payload, error) {
		return nil, errors.New("signature rejected")
	}
	if _, err := p.Complete(context.Background(), provider.LoginCallback{Code: "code", Verifier: "verifier", Nonce: "nonce"}); !errors.Is(err, journal.ErrAuthRequired) {
		t.Fatal("invalid identity accepted")
	}
}

func TestIdentityAndCredentialRejections(t *testing.T) {
	ctx := context.Background()
	refresh := "new-refresh"
	scope := driveScope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access", "refresh_token": refresh, "id_token": "signed", "token_type": "Bearer", "expires_in": 3600, "scope": scope})
	}))
	defer server.Close()
	p := New(Config{ClientID: "client", ClientSecret: "secret"})
	p.endpoint = oauth2.Endpoint{TokenURL: server.URL, AuthStyle: oauth2.AuthStyleInParams}
	valid := func() *idtoken.Payload {
		return &idtoken.Payload{Issuer: "accounts.google.com", Audience: "client", Subject: "account", IssuedAt: time.Now().Unix(), Expires: time.Now().Add(time.Hour).Unix(), Claims: map[string]any{"nonce": "nonce", "email": "user@example.com", "email_verified": true}}
	}
	payload := valid()
	p.validate = func(context.Context, string, string) (*idtoken.Payload, error) { return payload, nil }
	old, _ := json.Marshal(credentials{Version: 1, Subject: "account", Refresh: "old-refresh"})
	previous := provider.Connection{ProviderID: "google", AccountID: "account", Credentials: string(old)}
	callback := provider.LoginCallback{Code: "code", Verifier: "verifier", Nonce: "nonce", Previous: &previous}
	refresh = ""
	conn, err := p.Complete(ctx, callback)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := decodeCredentials(conn)
	if err != nil || stored.Refresh != "old-refresh" {
		t.Fatal("omitted refresh token lost")
	}
	cases := []struct {
		name   string
		change func()
	}{
		{"issuer", func() { payload.Issuer = "other" }},
		{"audience", func() { payload.Audience = "other" }},
		{"expiry", func() { payload.Expires = time.Now().Unix() - 1 }},
		{"future issue", func() { payload.IssuedAt = time.Now().Add(time.Hour).Unix() }},
		{"nonce", func() { payload.Claims["nonce"] = "wrong" }},
		{"verified email", func() { payload.Claims["email_verified"] = false }},
		{"authorized party", func() { payload.Claims["azp"] = "other" }},
		{"account switch", func() { payload.Subject = "other" }},
		{"missing grant", func() { scope = "openid email profile" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload = valid()
			scope = driveScope
			tc.change()
			if _, err := p.Complete(ctx, callback); !errors.Is(err, journal.ErrAuthRequired) {
				t.Fatal("invalid grant or identity accepted")
			}
		})
	}
	payload = valid()
	scope = driveScope
	callback.Previous = nil
	if _, err := p.Complete(ctx, callback); !errors.Is(err, journal.ErrAuthRequired) {
		t.Fatal("new account accepted without refresh token")
	}
	// Cached access tokens are bounded even across many connections.
	for i := 0; i < 200; i++ {
		p.cacheToken(fmt.Sprint(i), &oauth2.Token{AccessToken: "token", Expiry: time.Now().Add(time.Hour)})
	}
	if len(p.tokens) > 128 {
		t.Fatal("unbounded token cache")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestRevocationAndRefreshFailure(t *testing.T) {
	p := New(Config{ClientID: "client", ClientSecret: "secret"})
	stored, _ := json.Marshal(credentials{Version: 1, Subject: "account", Refresh: "refresh"})
	conn := provider.Connection{ProviderID: "google", AccountID: "account", Credentials: string(stored)}
	p.cacheToken("refresh", &oauth2.Token{AccessToken: "cached", Expiry: time.Now().Add(time.Hour)})
	status := 503
	p.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://oauth2.googleapis.com/revoke" || r.Method != http.MethodPost {
			t.Error("unexpected revocation request")
		}
		if err := r.ParseForm(); err != nil || r.Form.Get("token") != "refresh" {
			t.Error("missing revocation credential")
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
	})}
	if err := p.Revoke(context.Background(), conn); !errors.Is(err, journal.ErrUnavailable) {
		t.Fatal("failed revocation reported success")
	}
	if len(p.tokens) != 1 {
		t.Fatal("failed revocation discarded cache")
	}
	status = 200
	if err := p.Revoke(context.Background(), conn); err != nil || len(p.tokens) != 0 {
		t.Fatal("successful revocation retained token")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = io.WriteString(w, `{"error":"invalid_grant"}`)
	}))
	defer server.Close()
	p.client = server.Client()
	p.endpoint = oauth2.Endpoint{TokenURL: server.URL, AuthStyle: oauth2.AuthStyleInParams}
	if _, err := p.accessToken(context.Background(), "revoked"); !errors.Is(err, journal.ErrAuthRequired) {
		t.Fatal("revoked token did not require reconnect")
	}
}

func TestTokenReplacementAtCapacityPreservesOtherEntries(t *testing.T) {
	p := New(Config{})
	for i := range 128 {
		p.cacheToken(fmt.Sprint(i), &oauth2.Token{AccessToken: "test"})
	}
	p.cacheToken("0", &oauth2.Token{AccessToken: "updated"})
	if len(p.tokens) != 128 {
		t.Fatalf("replacement evicted another token: %d", len(p.tokens))
	}
}
