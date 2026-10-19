package local

import (
	"context"
	"strings"

	provider "myjournal-app/internal/core/connection"
	"myjournal-app/internal/core/journal"
)

// CallbackPath is the local development sign-in callback route.
const CallbackPath = "/api/v1/auth/local/callback"

// Provider is the local dev-only storage provider used for UI work without OAuth.
type Provider struct {
	baseURL string
	dataDir string
}

var _ provider.Provider = (*Provider)(nil)

// New constructs the development adapter from explicit startup configuration.
func New(baseURL, dataDir string) *Provider {
	return &Provider{baseURL: baseURL, dataDir: dataDir}
}

func (p *Provider) ID() string { return provider.LocalID }

func (p *Provider) Name() string { return "Local dev account" }

func (p *Provider) Begin(_ context.Context, _ provider.LoginRequest) (string, error) {
	if err := provider.ValidateLocalOrigin(p.baseURL); err != nil {
		return "", err
	}
	return strings.TrimRight(p.baseURL, "/") + CallbackPath, nil
}

func (p *Provider) Complete(_ context.Context, _ provider.LoginCallback) (provider.Connection, error) {
	return provider.Connection{
		ProviderID:   provider.LocalID,
		AccountID:    "dev",
		AccountEmail: "dev@localhost",
		Credentials:  "local-dev-user",
	}, nil
}

func (p *Provider) Store(_ context.Context, _ provider.Connection) (journal.Store, error) {
	return NewStore(p.dataDir), nil
}

func (p *Provider) Revoke(_ context.Context, _ provider.Connection) error {
	return nil
}

func (*Provider) RequiresLoginFlow() bool { return false }
