package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github/mridha/myjournal-app/internal/provider"
	"github/mridha/myjournal-app/internal/session"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"
)

const cookieName = "mj_s7_session"
const loginCookieName = "mj_s7_oauth"
const lifetime = 30 * 24 * time.Hour // Experiment proposal, not a production setting.

var errRootTrashed = errors.New("root trashed")
var errRootNotFolder = errors.New("root is not a folder")

type credentials struct{ RefreshToken, RootID string }
type pending struct{ State, Verifier string }
type experiment struct {
	secret, rootID string
	secure         bool
	cfg            oauth2.Config
	now            func() time.Time
	mu             sync.Mutex
	tokens         map[[32]byte]*oauth2.Token
	driveCheck     func(context.Context, *oauth2.Token, string) error
}

func main() {
	root := flag.String("root-id", "", "existing S02 root folder ID (read-only checks)")
	flag.Parse()
	secret := os.Getenv("SESSION_KEY")
	key, err := base64.StdEncoding.DecodeString(secret)
	if err != nil || len(key) != 32 {
		log.Fatal("SESSION_KEY must be base64 encoding of 32 random bytes")
	}
	secure := false
	if value := os.Getenv("COOKIE_SECURE"); value != "" {
		secure, err = strconv.ParseBool(value)
		if err != nil {
			log.Fatal("invalid COOKIE_SECURE")
		}
	}
	if *root == "" {
		log.Fatal("pass -root-id with the existing S02 root folder ID")
	}
	if base := os.Getenv("APP_BASE_URL"); base != "" && base != "http://localhost:3030" {
		log.Fatal("S7 requires APP_BASE_URL=http://localhost:3030")
	}
	e := &experiment{secret: secret, rootID: *root, secure: secure, now: time.Now, tokens: make(map[[32]byte]*oauth2.Token),
		cfg: oauth2.Config{ClientID: os.Getenv("GOOGLE_CLIENT_ID"), ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL: "http://localhost:3030/auth/google/callback", Endpoint: google.Endpoint,
			Scopes: []string{"openid", "email", "profile", "https://www.googleapis.com/auth/drive.file"}}}
	if e.cfg.ClientID == "" || e.cfg.ClientSecret == "" {
		log.Fatal("set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET")
	}
	e.driveCheck = func(ctx context.Context, token *oauth2.Token, root string) error {
		ds, err := drive.NewService(ctx, option.WithTokenSource(oauth2.StaticTokenSource(token)))
		if err != nil {
			return err
		}
		f, err := ds.Files.Get(root).Fields("id,mimeType,trashed").Context(ctx).Do()
		if err != nil {
			return err
		}
		if f.Trashed {
			return errRootTrashed
		}
		if f.MimeType != "application/vnd.google-apps.folder" {
			return errRootNotFolder
		}
		return nil
	}
	server := &http.Server{Addr: "127.0.0.1:3030", Handler: e.routes(), ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 10 * time.Second, WriteTimeout: 40 * time.Second, IdleTimeout: 60 * time.Second}
	log.Print("S7 listening at http://localhost:3030; open / (no request-query logging)")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal("S7 server stopped; check port 3030 ownership")
	}
}

func (e *experiment) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", e.home)
	mux.HandleFunc("GET /login", e.login)
	mux.HandleFunc("GET /auth/google/callback", e.callback)
	mux.HandleFunc("GET /check", e.check)
	mux.HandleFunc("POST /tamper", e.tamper)
	mux.HandleFunc("POST /signout", func(w http.ResponseWriter, r *http.Request) {
		if s, err := e.readSession(r); err == nil {
			var c credentials
			if json.Unmarshal([]byte(s.Credentials), &c) == nil {
				e.forget(c.RefreshToken)
			}
		}
		e.clear(w, cookieName)
		http.Redirect(w, r, "/?result=signed_out", http.StatusSeeOther)
	})
	protected := new(http.CrossOriginProtection).Handler(mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		protected.ServeHTTP(w, r)
	})
}

func (e *experiment) home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	result := r.URL.Query().Get("result")
	switch result {
	case "session_rejected", "invalid_grant", "signed_out":
	default:
		result = "ready"
	}
	_, _ = fmt.Fprintf(w, `<h1>S7 cookie lifecycle</h1><p>result: %s</p><p>Session lifetime proposal: 30 days. Tokens and account values are never displayed.</p>
<p><a href="/login?force_consent=1">Sign in with consent</a> | <a href="/login">Sign in without forced consent</a></p>
<p><a href="/check">Check session and Drive</a> | <a href="/check?force_refresh=1">Simulate access-token expiry and check Drive</a></p>
<form method="post" action="/tamper"><button>Tamper with this spike's cookie</button></form>
<form method="post" action="/signout"><button>Clear this spike's session</button></form>`, result)
}

func (e *experiment) cookie(name, value string, age time.Duration) *http.Cookie {
	return &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: true, Secure: e.secure,
		SameSite: http.SameSiteLaxMode, MaxAge: int(age.Seconds()), Expires: e.now().Add(age)}
}
func (e *experiment) clear(w http.ResponseWriter, name string) {
	c := e.cookie(name, "", time.Second)
	c.MaxAge = -1
	c.Expires = time.Unix(1, 0)
	http.SetCookie(w, c)
}

func (e *experiment) login(w http.ResponseWriter, r *http.Request) {
	state := make([]byte, 32)
	if _, err := rand.Read(state); err != nil {
		http.Error(w, "login unavailable", http.StatusInternalServerError)
		return
	}
	p := pending{State: base64.RawURLEncoding.EncodeToString(state), Verifier: oauth2.GenerateVerifier()}
	b, err := json.Marshal(p)
	if err != nil {
		http.Error(w, "login unavailable", http.StatusInternalServerError)
		return
	}
	encoded, err := session.Encode(e.secret+":s7-oauth", session.Session{Connection: provider.Connection{Credentials: string(b)}, IssuedAt: e.now()})
	if err != nil {
		http.Error(w, "login unavailable", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, e.cookie(loginCookieName, encoded, 5*time.Minute))
	opts := []oauth2.AuthCodeOption{oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(p.Verifier)}
	if r.URL.Query().Get("force_consent") == "1" {
		opts = append(opts, oauth2.SetAuthURLParam("prompt", "consent"))
	}
	http.Redirect(w, r, e.cfg.AuthCodeURL(p.State, opts...), http.StatusFound)
}

func validAge(now, issued time.Time, age time.Duration) bool {
	return !issued.IsZero() && !issued.After(now) && now.Sub(issued) < age
}

func (e *experiment) callback(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(loginCookieName)
	e.clear(w, loginCookieName)
	if err != nil {
		http.Error(w, "OAuth state rejected", http.StatusBadRequest)
		return
	}
	s, err := session.Decode(e.secret+":s7-oauth", c.Value)
	var p pending
	if err != nil || !validAge(e.now(), s.IssuedAt, 5*time.Minute) || json.Unmarshal([]byte(s.Credentials), &p) != nil || p.State == "" || p.Verifier == "" || subtle.ConstantTimeCompare([]byte(p.State), []byte(r.URL.Query().Get("state"))) != 1 {
		http.Error(w, "OAuth state rejected", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" {
		http.Error(w, "authorization declined or incomplete", http.StatusUnauthorized)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	token, err := e.cfg.Exchange(ctx, r.URL.Query().Get("code"), oauth2.VerifierOption(p.Verifier))
	if err != nil {
		http.Error(w, "token exchange failed (details suppressed)", http.StatusUnauthorized)
		return
	}
	raw, _ := token.Extra("id_token").(string)
	payload, err := idtoken.Validate(ctx, raw, e.cfg.ClientID)
	if err != nil || payload.Subject == "" {
		http.Error(w, "ID token rejected", http.StatusUnauthorized)
		return
	}
	email, _ := payload.Claims["email"].(string)
	verified, _ := payload.Claims["email_verified"].(bool)
	if email == "" || !verified {
		http.Error(w, "verified email claim required", http.StatusUnauthorized)
		return
	}
	returned := token.RefreshToken != ""
	if !returned {
		old, err := e.readSession(r)
		if err == nil && old.AccountID == payload.Subject {
			var previous credentials
			if json.Unmarshal([]byte(old.Credentials), &previous) == nil {
				token.RefreshToken = previous.RefreshToken
			}
		}
	}
	if token.RefreshToken == "" {
		http.Error(w, "refresh token missing; retry /login?force_consent=1", http.StatusUnauthorized)
		return
	}
	if err := e.driveCheck(ctx, token, e.rootID); err != nil {
		driveFailure(w, err, false)
		return
	}
	b, err := json.Marshal(credentials{RefreshToken: token.RefreshToken, RootID: e.rootID})
	if err != nil {
		http.Error(w, "session unavailable", http.StatusInternalServerError)
		return
	}
	encoded, err := session.Encode(e.secret, session.Session{Connection: provider.Connection{ProviderID: "google", AccountID: payload.Subject, AccountEmail: email, Credentials: string(b)}, IssuedAt: e.now()})
	if err != nil {
		http.Error(w, "session unavailable", http.StatusInternalServerError)
		return
	}
	cookie := e.cookie(cookieName, encoded, lifetime)
	if len(cookie.String()) >= 3800 {
		http.Error(w, "session cookie exceeds experiment size budget", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, cookie)
	// Do not seed the cache: the first /check proves refresh from cookie credentials.
	e.forget(token.RefreshToken)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintf(w, "session_cookie_set: true\nrefresh_token_returned: %t\nexisting_refresh_token_preserved: %t\nid_token_valid: true\ndrive_root_read_succeeded: true\nset_cookie_bytes: %d\ncookie_under_3800_bytes: true\ncookie_httponly: true\ncookie_samesite_lax: true\ncookie_secure: %t\ncookie_lifetime_days: 30\nNext: open http://localhost:3030/check\n", returned, !returned, len(cookie.String()), e.secure)
}

func (e *experiment) readSession(r *http.Request) (session.Session, error) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return session.Session{}, errors.New("missing cookie")
	}
	s, err := session.Decode(e.secret, c.Value)
	if err != nil || s.ProviderID != "google" || s.AccountID == "" || s.AccountEmail == "" || !validAge(e.now(), s.IssuedAt, lifetime) {
		return session.Session{}, errors.New("invalid session")
	}
	return s, nil
}

func (e *experiment) forget(refresh string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.tokens, sha256.Sum256([]byte(refresh)))
}

func (e *experiment) accessToken(ctx context.Context, refresh string, force bool) (*oauth2.Token, bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	key := sha256.Sum256([]byte(refresh))
	if token := e.tokens[key]; !force && token != nil && token.AccessToken != "" && token.Expiry.After(e.now().Add(time.Minute)) {
		return token, true, nil
	}
	delete(e.tokens, key)
	token, err := e.cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: refresh, Expiry: e.now().Add(-time.Hour)}).Token()
	if err != nil {
		return nil, false, err
	}
	if token.AccessToken == "" {
		return nil, false, errors.New("empty access token")
	}
	if len(e.tokens) >= 128 {
		for old := range e.tokens {
			delete(e.tokens, old)
			break
		}
	}
	e.tokens[key] = token
	return token, false, nil
}

func (e *experiment) reject(w http.ResponseWriter, r *http.Request, reason string) {
	e.clear(w, cookieName)
	http.Redirect(w, r, "/?result="+reason, http.StatusSeeOther)
}

func (e *experiment) check(w http.ResponseWriter, r *http.Request) {
	s, err := e.readSession(r)
	if err != nil {
		e.reject(w, r, "session_rejected")
		return
	}
	var c credentials
	if json.Unmarshal([]byte(s.Credentials), &c) != nil || c.RefreshToken == "" || c.RootID == "" {
		e.reject(w, r, "session_rejected")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	force := r.URL.Query().Get("force_refresh") == "1"
	token, cached, err := e.accessToken(ctx, c.RefreshToken, force)
	if err != nil {
		var retrieve *oauth2.RetrieveError
		if errors.As(err, &retrieve) && retrieve.ErrorCode == "invalid_grant" {
			e.reject(w, r, "invalid_grant")
			return
		}
		http.Error(w, "token refresh unavailable; session retained; retry (details suppressed)", http.StatusBadGateway)
		return
	}
	if err := e.driveCheck(ctx, token, c.RootID); err != nil {
		driveFailure(w, err, true)
		return
	}
	if token.RefreshToken != "" && token.RefreshToken != c.RefreshToken {
		oldRefresh := c.RefreshToken
		c.RefreshToken = token.RefreshToken
		blob, marshalErr := json.Marshal(c)
		if marshalErr != nil {
			http.Error(w, "session update unavailable", http.StatusInternalServerError)
			return
		}
		s.Credentials = string(blob)
		encoded, encodeErr := session.Encode(e.secret, s)
		if encodeErr != nil {
			http.Error(w, "session update unavailable", http.StatusInternalServerError)
			return
		}
		cookie := e.cookie(cookieName, encoded, lifetime-e.now().Sub(s.IssuedAt))
		if len(cookie.String()) >= 3800 {
			http.Error(w, "session update exceeds size budget", http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, cookie)
		e.forget(oldRefresh)
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = fmt.Fprintf(w, "session_cookie_valid: true\naccess_token_cache_hit: %t\naccess_token_refreshed: %t\nsimulated_expiry: %t\ndrive_root_read_succeeded: true\n", cached, !cached, force)
}

// External error messages can contain private resource identifiers.
func driveFailure(w http.ResponseWriter, err error, retained bool) {
	category := "unclassified"
	status := 0
	var apiErr *googleapi.Error
	switch {
	case errors.Is(err, errRootTrashed):
		category = "root_trashed"
	case errors.Is(err, errRootNotFolder):
		category = "root_not_folder"
	case errors.Is(err, context.DeadlineExceeded):
		category = "timeout"
	case errors.As(err, &apiErr):
		status = apiErr.Code
		switch status {
		case http.StatusNotFound:
			category = "root_not_found_or_inaccessible"
		case http.StatusForbidden:
			category = "access_denied"
		case http.StatusUnauthorized:
			category = "access_token_rejected"
		case http.StatusTooManyRequests:
			category = "rate_limited"
		default:
			category = "google_api_error"
		}
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusBadGateway)
	_, _ = fmt.Fprintf(w, "drive_root_read_succeeded: false\ndrive_failure_category: %s\ngoogle_http_status: %d\nsession_retained: %t\n", category, status, retained)
}

func (e *experiment) tamper(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		e.reject(w, r, "session_rejected")
		return
	}
	b, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil || len(b) == 0 {
		e.reject(w, r, "session_rejected")
		return
	}
	b[len(b)-1] ^= 1
	http.SetCookie(w, e.cookie(cookieName, base64.RawURLEncoding.EncodeToString(b), lifetime))
	http.Redirect(w, r, "/check", http.StatusSeeOther)
}
