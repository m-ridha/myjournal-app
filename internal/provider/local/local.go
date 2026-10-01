package local

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github/mridha/myjournal-app/internal/journal"
	"github/mridha/myjournal-app/internal/provider"
)

// Provider is the local dev-only storage provider used for UI work without OAuth.
type Provider struct {
	baseURL string
	dataDir string
}

func New(values ...string) *Provider {
	p := &Provider{}
	switch len(values) {
	case 0:
		p.baseURL = "http://localhost:3030"
	case 1:
		if strings.HasPrefix(values[0], "http://") || strings.HasPrefix(values[0], "https://") {
			p.baseURL = values[0]
		} else {
			p.dataDir = values[0]
		}
	default:
		p.baseURL = values[0]
		p.dataDir = values[1]
	}
	if p.baseURL == "" {
		p.baseURL = "http://localhost:3030"
	}
	if p.dataDir == "" {
		p.dataDir = os.Getenv("LOCAL_DATA_DIR")
	}
	if p.dataDir == "" {
		p.dataDir = "./.devdata"
	}
	return p
}

func (p *Provider) ID() string { return "local" }

func (p *Provider) Name() string { return "Local dev account" }

func (p *Provider) Begin(_ context.Context, _ string) (string, error) {
	if !strings.HasPrefix(p.baseURL, "http://localhost") && !strings.HasPrefix(p.baseURL, "http://127.0.0.1") {
		return "", fmt.Errorf("local provider requires APP_BASE_URL to point at localhost")
	}
	return p.baseURL + "/api/v1/auth/local/callback", nil
}

func (p *Provider) Complete(_ context.Context, _ string) (provider.Connection, error) {
	return provider.Connection{
		ProviderID:   "local",
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
