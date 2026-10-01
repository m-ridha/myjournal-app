package google

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"

	"github/mridha/myjournal-app/internal/journal"
	"github/mridha/myjournal-app/internal/provider"

	"golang.org/x/oauth2"
	goauth "golang.org/x/oauth2/google"
)

// Config carries the environment-derived Google OAuth configuration.
type Config struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
	Scopes       []string
}

func ConfigFromEnv() Config {
	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3030"
	}
	return Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		BaseURL:      baseURL,
		Scopes: []string{
			"openid",
			"email",
			"profile",
			"https://www.googleapis.com/auth/drive.file",
		},
	}
}

// Provider is the Google-backed provider used by the real application.
type Provider struct {
	cfg Config
}

func New(cfg Config) *Provider {
	return &Provider{cfg: cfg}
}

func (p *Provider) ID() string { return "google" }

func (p *Provider) Name() string { return "Google" }

func (p *Provider) Begin(_ context.Context, state string) (string, error) {
	if p.cfg.ClientID == "" || p.cfg.ClientSecret == "" {
		return "", fmt.Errorf("google OAuth client configuration is missing")
	}

	cfg := &oauth2.Config{
		ClientID:     p.cfg.ClientID,
		ClientSecret: p.cfg.ClientSecret,
		RedirectURL:  p.cfg.BaseURL + "/auth/google/callback",
		Scopes:       p.cfg.Scopes,
		Endpoint:     goauth.Endpoint,
	}

	verifier, err := randomVerifier()
	if err != nil {
		return "", err
	}
	challenge := codeChallenge(verifier)

	url := cfg.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
	return url, nil
}

func (p *Provider) Complete(_ context.Context, _ string) (provider.Connection, error) {
	return provider.Connection{
		ProviderID:   "google",
		AccountID:    "google-user",
		AccountEmail: "placeholder@google.local",
		Credentials:  "google-oauth-credentials",
	}, nil
}

func (p *Provider) Store(_ context.Context, _ provider.Connection) (journal.Store, error) {
	return nil, fmt.Errorf("google provider store is not implemented yet")
}

func (p *Provider) Revoke(_ context.Context, _ provider.Connection) error {
	return nil
}

func randomVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func codeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
