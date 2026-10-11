package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	provider "github/mridha/myjournal-app/internal/core/connection"
	"github/mridha/myjournal-app/internal/core/journal"
	"github/mridha/myjournal-app/internal/driven/session"
	"github/mridha/myjournal-app/internal/driver/web/api"
	"github/mridha/myjournal-app/internal/driver/web/authsession"
)

type loginProvider struct {
	provider.Provider
	request  provider.LoginRequest
	callback provider.LoginCallback
	calls    int
}

func (p *loginProvider) ID() string { return "google" }
func (p *loginProvider) Begin(_ context.Context, r provider.LoginRequest) (string, error) {
	p.request = r
	return "https://example.com/authorize", nil
}
func (p *loginProvider) Complete(_ context.Context, c provider.LoginCallback) (provider.Connection, error) {
	p.callback = c
	p.calls++
	return provider.Connection{ProviderID: "google", AccountID: "test"}, nil
}

func TestBrowserLoginHandoff(t *testing.T) {
	now := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	p := &loginProvider{}
	s := &Server{cfg: Options{SessionKey: "test-key", CookieSecure: true}, registry: provider.NewRegistry(p), now: func() time.Time { return now }}
	s.connections = provider.NewService(s.registry, testClock(s.now), rand.Reader)
	w := httptest.NewRecorder()
	s.StartProviderLogin(w, httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/login", nil), "google", api.StartProviderLoginParams{})
	if w.Code != 302 {
		t.Fatalf("login status %d", w.Code)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("missing pending cookie")
	}
	cookie := cookies[0]
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != authsession.LoginCookiePath || cookie.MaxAge != 600 {
		t.Fatal("incorrect pending cookie flags")
	}
	flow, err := session.DecodeLoginFlow("test-key", cookie.Value)
	if err != nil || flow.State != p.request.State || flow.Verifier != p.request.Verifier || flow.Nonce != p.request.Nonce {
		t.Fatal("provider handoff lost pending secrets")
	}
	otherFlow := flow
	otherFlow.ProviderID = "other"
	otherValue, err := session.EncodeLoginFlow("test-key", otherFlow)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, query string
		cookie      *http.Cookie
		clock       time.Time
		want        int
	}{
		{"valid", "code=test&state=" + flow.State, cookie, now, 302},
		{"missing cookie", "code=test&state=" + flow.State, nil, now, 401},
		{"wrong state", "code=test&state=wrong", cookie, now, 401},
		{"missing code", "state=" + flow.State, cookie, now, 401},
		{"duplicate state", "code=test&state=" + flow.State + "&state=" + flow.State, cookie, now, 401},
		{"duplicate code", "code=test&code=test&state=" + flow.State, cookie, now, 401},
		{"denied", "code=test&state=" + flow.State + "&error=access_denied", cookie, now, 401},
		{"expired", "code=test&state=" + flow.State, cookie, now.Add(provider.LoginLifetime), 401},
		{"future issuance", "code=test&state=" + flow.State, cookie, now.Add(-time.Second), 401},
		{"wrong provider", "code=test&state=" + flow.State, &http.Cookie{Name: authsession.LoginCookieName, Value: otherValue}, now, 401},
		{"tampered", "code=test&state=" + flow.State, &http.Cookie{Name: authsession.LoginCookieName, Value: "invalid"}, now, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s.now = func() time.Time { return tc.clock }
			s.connections = provider.NewService(s.registry, testClock(s.now), rand.Reader)
			r := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?"+tc.query, nil)
			if tc.cookie != nil {
				r.AddCookie(tc.cookie)
			}
			q, _ := url.ParseQuery(tc.query)
			code, state := q.Get("code"), q.Get("state")
			before := p.calls
			w := httptest.NewRecorder()
			s.connections = provider.NewService(s.registry, testClock(s.now), rand.Reader)
			s.CompleteProviderLogin(w, r, "google", api.CompleteProviderLoginParams{Code: &code, State: &state})
			if w.Code != tc.want {
				t.Fatalf("status %d want %d", w.Code, tc.want)
			}
			if tc.want == 302 {
				if p.calls != before+1 || p.callback.Code != "test" || p.callback.Verifier != flow.Verifier || p.callback.Nonce != flow.Nonce {
					t.Fatal("incorrect exchange handoff")
				}
			} else if p.calls != before {
				t.Fatal("invalid callback reached exchange")
			}
			cleared := false
			for _, c := range w.Result().Cookies() {
				if c.Name == authsession.LoginCookieName && c.MaxAge < 0 {
					cleared = true
				}
				if tc.want != 302 && c.Name == authsession.CookieName {
					t.Fatal("failed login changed session")
				}
			}
			if !cleared {
				t.Fatal("pending cookie not consumed")
			}
		})
	}
}

type unavailableLoginProvider struct{ loginProvider }

func (p *unavailableLoginProvider) Complete(context.Context, provider.LoginCallback) (provider.Connection, error) {
	return provider.Connection{}, journal.ErrUnavailable
}
func TestLoginExchangeOutagePreservesSession(t *testing.T) {
	now := time.Now().UTC()
	flow, err := provider.NewLoginFlow("google", now, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	value, err := session.EncodeLoginFlow("key", flow)
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{cfg: Options{SessionKey: "key"}, registry: provider.NewRegistry(&unavailableLoginProvider{}), now: func() time.Time { return now }}
	code, state := "code", flow.State
	r := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google/callback?code=code&state="+state, nil)
	r.AddCookie(&http.Cookie{Name: authsession.LoginCookieName, Value: value})
	w := httptest.NewRecorder()
	s.connections = provider.NewService(s.registry, testClock(s.now), rand.Reader)
	s.CompleteProviderLogin(w, r, "google", api.CompleteProviderLoginParams{Code: &code, State: &state})
	if w.Code != http.StatusBadGateway {
		t.Fatalf("outage status %d", w.Code)
	}
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == authsession.CookieName {
			t.Fatal("outage changed existing session")
		}
	}
}

func TestLoginDiagnosticsAllowOnlyKnownStages(t *testing.T) {
	old := slog.Default()
	defer slog.SetDefault(old)
	var output bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	logLoginRejection(provider.LoginStage("synthetic-sensitive-value"))
	if output.Len() != 0 {
		t.Fatal("unrecognized diagnostic logged")
	}
	logLoginRejection(provider.LoginClaims)
	if !strings.Contains(output.String(), `"stage":"claims"`) || strings.Contains(output.String(), "synthetic-sensitive-value") {
		t.Fatal("unexpected diagnostic")
	}
}
