package google

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"

	provider "myjournal-app/internal/core/connection"
	"myjournal-app/internal/core/journal"
)

const (
	driveScope              = "https://www.googleapis.com/auth/drive.file"
	credentialsVersion      = 1
	maxCredentialBytes      = 2400
	maxRefreshTokenBytes    = 1800
	maxSubjectBytes         = 255
	maxEmailBytes           = 320
	revocationResponseBytes = 4096
)

type credentials struct {
	Version  int    `json:"v"`
	Subject  string `json:"s"`
	Refresh  string `json:"r"`
	IssuedAt int64  `json:"i,omitempty"`
}

func decodeCredentials(c provider.Connection) (credentials, error) {
	var stored credentials
	if c.ProviderID != provider.GoogleID || len(c.Credentials) > maxCredentialBytes || json.Unmarshal([]byte(c.Credentials), &stored) != nil || stored.Version != credentialsVersion || stored.Subject == "" || stored.Subject != c.AccountID || stored.Refresh == "" {
		return stored, journal.ErrAuthRequired
	}
	return stored, nil
}

func (p *Provider) oauthConfig() *oauth2.Config {
	return &oauth2.Config{ClientID: p.cfg.ClientID, ClientSecret: p.cfg.ClientSecret, RedirectURL: strings.TrimRight(p.cfg.BaseURL, "/") + "/api/v1/auth/google/callback", Scopes: p.cfg.Scopes, Endpoint: p.endpoint}
}

func (p *Provider) Complete(ctx context.Context, callback provider.LoginCallback) (provider.Connection, error) {
	if callback.Code == "" || callback.Verifier == "" || callback.Nonce == "" {
		return provider.Connection{}, journal.ErrAuthRequired
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.client)
	token, err := p.oauthConfig().Exchange(ctx, callback.Code, oauth2.VerifierOption(callback.Verifier))
	if err != nil {
		return provider.Connection{}, &provider.LoginError{Stage: provider.LoginExchange, Err: authError(err)}
	}
	raw, _ := token.Extra("id_token").(string)
	claims, err := p.validate(ctx, raw, p.cfg.ClientID)
	if err != nil || claims == nil {
		return provider.Connection{}, loginRejected(provider.LoginClaims)
	}
	now := time.Now()
	if !validIdentity(claims, p.cfg.ClientID, callback.Nonce, now) {
		return provider.Connection{}, loginRejected(provider.LoginClaims)
	}
	email, _ := claims.Claims["email"].(string)
	scope, _ := token.Extra("scope").(string)
	granted := false
	for _, item := range strings.Fields(scope) {
		if item == driveScope {
			granted = true
		}
	}
	if !granted || token.AccessToken == "" || token.Expiry.IsZero() || !token.Expiry.After(now) {
		return provider.Connection{}, loginRejected(provider.LoginGrant)
	}
	if callback.Previous != nil && (callback.Previous.ProviderID != provider.GoogleID || callback.Previous.AccountID != claims.Subject) {
		return provider.Connection{}, loginRejected(provider.LoginAccount)
	}
	if token.RefreshToken == "" && callback.Previous != nil {
		old, err := decodeCredentials(*callback.Previous)
		if err == nil && old.Subject == claims.Subject {
			token.RefreshToken = old.Refresh
		}
	}
	if token.RefreshToken == "" || len(token.RefreshToken) > maxRefreshTokenBytes {
		return provider.Connection{}, loginRejected(provider.LoginRefresh)
	}
	stored := credentials{Version: credentialsVersion, Subject: claims.Subject, Refresh: token.RefreshToken, IssuedAt: now.UnixNano()}
	encoded, err := json.Marshal(stored)
	if err != nil {
		return provider.Connection{}, journal.ErrAuthRequired
	}
	p.cacheToken(stored.Refresh, token)
	return provider.Connection{ProviderID: provider.GoogleID, AccountID: claims.Subject, AccountEmail: email, Credentials: string(encoded)}, nil
}

func authError(err error) error {
	var e *oauth2.RetrieveError
	if errors.As(err, &e) && (e.ErrorCode == "invalid_grant" || e.ErrorCode == "invalid_token" || e.ErrorCode == "access_denied") {
		return journal.ErrAuthRequired
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return journal.ErrUnavailable
}

func (p *Provider) lock(ctx context.Context, key string) (func(), error) {
	hash := sha256.Sum256([]byte(key))
	lock := p.locks[int(hash[0])%len(p.locks)]
	select {
	case lock <- struct{}{}:
		return func() { <-lock }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *Provider) cacheToken(refresh string, token *oauth2.Token) {
	key := sha256.Sum256([]byte(refresh))
	copyToken := oauth2.Token{AccessToken: token.AccessToken, TokenType: token.TokenType, Expiry: token.Expiry}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.tokens[key]; !exists && len(p.tokens) >= maxCachedTokens {
		for k := range p.tokens {
			delete(p.tokens, k)
			break
		}
	}
	p.tokens[key] = &copyToken
}

func (p *Provider) dropToken(refresh string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.tokens, sha256.Sum256([]byte(refresh)))
}

func (p *Provider) accessToken(ctx context.Context, refresh string) (*oauth2.Token, error) {
	unlock, err := p.lock(ctx, "token:"+refresh)
	if err != nil {
		return nil, err
	}
	defer unlock()
	p.mu.Lock()
	cached := p.tokens[sha256.Sum256([]byte(refresh))]
	p.mu.Unlock()
	if cached != nil && cached.Valid() {
		copyToken := *cached
		return &copyToken, nil
	}
	return p.refreshToken(ctx, refresh)
}

// refreshToken must run under the credential lock, including authorization checks.
func (p *Provider) refreshToken(ctx context.Context, refresh string) (*oauth2.Token, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.client)
	token, err := p.oauthConfig().TokenSource(ctx, &oauth2.Token{RefreshToken: refresh}).Token()
	if err != nil {
		p.dropToken(refresh)
		return nil, authError(err)
	}
	// Google refresh tokens are stable. Fail closed if a future response rotates it;
	// persisting a replacement requires an acknowledged session-cookie update.
	if token.RefreshToken != "" && token.RefreshToken != refresh {
		return nil, journal.ErrAuthRequired
	}
	if token.AccessToken == "" || token.Expiry.IsZero() || !token.Expiry.After(time.Now()) {
		return nil, journal.ErrAuthRequired
	}
	p.cacheToken(refresh, token)
	return token, nil
}

type tokenSource struct {
	p      *Provider
	ctx    context.Context
	stored credentials
}

func (s tokenSource) Token() (*oauth2.Token, error) {
	if err := s.p.checkConnection(s.ctx, s.stored); err != nil {
		return nil, err
	}
	return s.p.accessToken(s.ctx, s.stored.Refresh)
}

func (p *Provider) Revoke(ctx context.Context, c provider.Connection) error {
	stored, err := decodeCredentials(c)
	if err != nil {
		return err
	}
	if err := p.checkConnection(ctx, stored); err != nil {
		return err
	}
	unlock, err := p.lock(ctx, "token:"+stored.Refresh)
	if err != nil {
		return err
	}
	defer unlock()
	if err := p.checkConnection(ctx, stored); err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/revoke", strings.NewReader(url.Values{"token": {stored.Refresh}}.Encode()))
	if err != nil {
		return journal.ErrUnavailable
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := p.client.Do(request)
	if err != nil {
		return authError(err)
	}
	defer func() { _ = response.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, revocationResponseBytes))
	if response.StatusCode != http.StatusOK {
		return journal.ErrUnavailable
	}
	// Keep a monotonic cutoff even if concurrent revocations finish out of order.
	cutoff := p.revocationCutoff(stored.Subject)
	revokedAt := time.Now().UnixNano()
	for previous := cutoff.Load(); previous < revokedAt; previous = cutoff.Load() {
		if cutoff.CompareAndSwap(previous, revokedAt) {
			break
		}
	}
	p.dropToken(stored.Refresh)
	return nil
}

func loginRejected(stage provider.LoginStage) error {
	return &provider.LoginError{Stage: stage, Err: journal.ErrAuthRequired}
}

func validIdentity(claims *idtoken.Payload, audience, expectedNonce string, now time.Time) bool {
	if claims == nil {
		return false
	}
	if claims.Issuer != "https://accounts.google.com" && claims.Issuer != "accounts.google.com" {
		return false
	}
	if claims.Audience != audience || claims.Subject == "" || len(claims.Subject) > maxSubjectBytes {
		return false
	}
	email, _ := claims.Claims["email"].(string)
	verified, _ := claims.Claims["email_verified"].(bool)
	if email == "" || len(email) > maxEmailBytes || !verified {
		return false
	}
	if claims.IssuedAt <= 0 || claims.Expires <= now.Unix() || claims.IssuedAt > now.Add(time.Minute).Unix() {
		return false
	}
	nonce, _ := claims.Claims["nonce"].(string)
	if subtle.ConstantTimeCompare([]byte(nonce), []byte(expectedNonce)) != 1 {
		return false
	}
	if value, present := claims.Claims["azp"]; present {
		azp, ok := value.(string)
		if !ok || azp != audience {
			return false
		}
	}
	return true
}
