package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/provider"
	"github/mridha/myjournal-app/internal/session"
	"github/mridha/myjournal-app/internal/testutil"

	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"
)

func fixture(t *testing.T) (*experiment, *http.Cookie, *int, *bool) {
	t.Helper()
	calls := new(int)
	revoked := new(bool)
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*calls++
		if err := r.ParseForm(); err != nil {
			t.Error("parse token request")
		}
		if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "synthetic-refresh" {
			t.Error("unexpected refresh request")
		}
		w.Header().Set("Content-Type", "application/json")
		if *revoked {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"PRIVATE"}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"synthetic-access","token_type":"Bearer","expires_in":3600}`))
	}))
	t.Cleanup(tokenServer.Close)
	e := &experiment{secret: base64.StdEncoding.EncodeToString(make([]byte, 32)), rootID: "synthetic-root", now: time.Now,
		tokens: make(map[[32]byte]*oauth2.Token), cfg: oauth2.Config{ClientID: "synthetic-client", ClientSecret: "synthetic-secret",
			Endpoint: oauth2.Endpoint{TokenURL: tokenServer.URL, AuthStyle: oauth2.AuthStyleInParams}}}
	e.driveCheck = func(_ context.Context, token *oauth2.Token, root string) error {
		if token.AccessToken != "synthetic-access" || root != "synthetic-root" {
			t.Error("invalid Drive credentials")
		}
		return nil
	}
	b, err := json.Marshal(credentials{RefreshToken: "synthetic-refresh", RootID: "synthetic-root"})
	if err != nil {
		t.Fatal(err)
	}
	v, err := session.Encode(e.secret, session.Session{Connection: provider.Connection{ProviderID: "google", AccountID: "synthetic-sub", AccountEmail: "synthetic@example.test", Credentials: string(b)}, IssuedAt: e.now()})
	if err != nil {
		t.Fatal(err)
	}
	return e, e.cookie(cookieName, v, lifetime), calls, revoked
}

func request(e *experiment, method, path string, c *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, testutil.LocalBaseURL+path, nil)
	if c != nil {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	e.routes().ServeHTTP(w, r)
	return w
}

func TestRefreshCacheExpiryRestartAndRevocation(t *testing.T) {
	e, cookie, calls, revoked := fixture(t)
	first := request(e, "GET", "/check", cookie)
	if first.Code != 200 || !strings.Contains(first.Body.String(), "access_token_refreshed: true") || *calls != 1 {
		t.Fatal("cold cache must refresh successfully")
	}
	second := request(e, "GET", "/check", cookie)
	if second.Code != 200 || !strings.Contains(second.Body.String(), "access_token_cache_hit: true") || *calls != 1 {
		t.Fatal("warm cache must reuse access token")
	}
	for _, token := range e.tokens {
		token.Expiry = time.Now().Add(-time.Hour)
	}
	if w := request(e, "GET", "/check", cookie); w.Code != 200 || *calls != 2 {
		t.Fatal("expired cache must refresh")
	}
	// A new instance holds no token/session state, but accepts the browser cookie.
	restarted := &experiment{secret: e.secret, rootID: e.rootID, now: e.now, cfg: e.cfg, tokens: make(map[[32]byte]*oauth2.Token), driveCheck: e.driveCheck}
	if w := request(restarted, "GET", "/check", cookie); w.Code != 200 || *calls != 3 {
		t.Fatal("restart must work from cookie alone")
	}
	*revoked = true
	w := request(restarted, "GET", "/check?force_refresh=1", cookie)
	if w.Code != 303 || w.Header().Get("Location") != "/?result=invalid_grant" {
		t.Fatal("revocation must redirect cleanly")
	}
	cleared := w.Result().Cookies()
	if len(cleared) != 1 || cleared[0].Name != cookieName || cleared[0].MaxAge != -1 {
		t.Fatal("revocation must clear cookie")
	}
	if strings.Contains(w.Body.String(), "PRIVATE") || strings.Contains(w.Body.String(), "synthetic-refresh") {
		t.Fatal("private error details exposed")
	}
}

func TestTamperRotationAndAge(t *testing.T) {
	e, c, calls, _ := fixture(t)
	if len(c.String()) >= 3800 || !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Secure || c.MaxAge != int(lifetime.Seconds()) {
		t.Fatal("invalid cookie attributes/size")
	}
	e.secure = true
	if !e.cookie(cookieName, "synthetic", lifetime).Secure {
		t.Fatal("Secure toggle ignored")
	}
	e.secure = false
	w := request(e, "POST", "/tamper", c)
	if w.Code != 303 || len(w.Result().Cookies()) != 1 {
		t.Fatal("tamper setup failed")
	}
	w = request(e, "GET", "/check", w.Result().Cookies()[0])
	if w.Code != 303 || w.Header().Get("Location") != "/?result=session_rejected" {
		t.Fatal("tampered cookie accepted")
	}
	e.secret = "rotated synthetic key"
	if w := request(e, "GET", "/check", c); w.Code != 303 {
		t.Fatal("old key cookie accepted")
	}
	if *calls != 0 {
		t.Fatal("invalid sessions must not contact Google")
	}
	now := time.Now()
	if validAge(now, now.Add(-lifetime), lifetime) || validAge(now, now.Add(time.Second), lifetime) || validAge(now, time.Time{}, lifetime) || !validAge(now, now.Add(-time.Minute), lifetime) {
		t.Fatal("invalid server-side age boundary")
	}
}

func TestDriveFailureDiagnosticsSuppressPrivateDetails(t *testing.T) {
	w := httptest.NewRecorder()
	driveFailure(w, &googleapi.Error{Code: 404, Message: "PRIVATE RESOURCE ID"}, false)
	if w.Code != 502 || !strings.Contains(w.Body.String(), "google_http_status: 404") || !strings.Contains(w.Body.String(), "root_not_found_or_inaccessible") || strings.Contains(w.Body.String(), "PRIVATE") {
		t.Fatal("unsafe or missing diagnostics")
	}
	for _, tc := range []struct {
		err      error
		category string
	}{{errRootTrashed, "root_trashed"}, {errRootNotFolder, "root_not_folder"}, {context.DeadlineExceeded, "timeout"}} {
		w := httptest.NewRecorder()
		driveFailure(w, tc.err, true)
		if !strings.Contains(w.Body.String(), tc.category) || !strings.Contains(w.Body.String(), "session_retained: true") {
			t.Fatal("incorrect diagnostics")
		}
	}
}

func TestLoginCookieAndCrossOriginProtection(t *testing.T) {
	e, c, _, _ := fixture(t)
	e.cfg.Endpoint.AuthURL = "https://accounts.example.test/auth"
	e.cfg.RedirectURL = testutil.LocalBaseURL + "/auth/google/callback"
	w := request(e, "GET", "/login", nil)
	if w.Code != 302 || len(w.Result().Cookies()) != 1 {
		t.Fatal("login setup failed")
	}
	pendingCookie := w.Result().Cookies()[0]
	if pendingCookie.Name != loginCookieName || !pendingCookie.HttpOnly || pendingCookie.MaxAge != 300 {
		t.Fatal("invalid pending login attributes")
	}
	s, err := session.Decode(e.secret+":s7-oauth", pendingCookie.Value)
	if err != nil {
		t.Fatal("pending login not encrypted")
	}
	var p pending
	if json.Unmarshal([]byte(s.Credentials), &p) != nil || p.State == "" || p.Verifier == "" {
		t.Fatal("state/verifier absent")
	}
	if w := request(e, "GET", "/auth/google/callback?state=wrong&code=synthetic", pendingCookie); w.Code != 400 {
		t.Fatal("state mismatch accepted")
	}
	r := httptest.NewRequest("POST", testutil.LocalBaseURL+"/tamper", nil)
	r.AddCookie(c)
	r.Header.Set("Origin", "https://elsewhere.example.test")
	w = httptest.NewRecorder()
	e.routes().ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross-origin tampering accepted")
	}
}
