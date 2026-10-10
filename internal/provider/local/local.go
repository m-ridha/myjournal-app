package local

import (
	"context"
	"strings"

	"github/mridha/myjournal-app/internal/config"
	"github/mridha/myjournal-app/internal/journal"
	"github/mridha/myjournal-app/internal/provider"
)

// Provider is the local dev-only storage provider used for UI work without OAuth.
type Provider struct {
	baseURL string
	dataDir string
}

// New constructs the development adapter from explicit startup configuration.
func New(baseURL, dataDir string) *Provider {
	return &Provider{baseURL: baseURL, dataDir: dataDir}
}

func (p *Provider) ID() string { return provider.LocalID }

func (p *Provider) Name() string { return "Local dev account" }

func (p *Provider) Begin(_ context.Context, _ provider.LoginRequest) (string, error) {
	if err := config.ValidateLoopbackURL(p.baseURL); err != nil {
		return "", err
	}
	return strings.TrimRight(p.baseURL, "/") + "/api/v1/auth/local/callback", nil
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

var _ provider.Provider = (*Provider)(nil)
