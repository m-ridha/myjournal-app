package provider

import (
	"context"

	"github/mridha/myjournal-app/internal/journal"
)

// Provider is the storage-provider boundary used by the application to
// establish and revoke a provider connection.
type Provider interface {
	ID() string
	Name() string
	Begin(ctx context.Context, request LoginRequest) (string, error)
	Complete(ctx context.Context, callback LoginCallback) (Connection, error)
	Store(ctx context.Context, c Connection) (journal.Store, error)
	Revoke(ctx context.Context, c Connection) error
}

// LoginRequest carries independent browser state, PKCE verifier, and OIDC nonce.
type LoginRequest struct {
	State, Verifier, Nonce string
	ForceConsent           bool
}

// LoginCallback carries the validated browser handoff to the provider exchange.
type LoginCallback struct {
	Code, Verifier, Nonce string
	Previous              *Connection
}

// Connection is a provider-specific connection blob stored in the session.
type Connection struct {
	ProviderID   string
	AccountID    string
	AccountEmail string
	Credentials  string
}

// Registry maps provider IDs to provider implementations.
type Registry struct {
	providers map[string]Provider
}

func NewRegistry(providers ...Provider) *Registry {
	r := &Registry{providers: make(map[string]Provider, len(providers))}
	for _, p := range providers {
		if p != nil {
			r.providers[p.ID()] = p
		}
	}
	return r
}

func (r *Registry) Get(id string) (Provider, bool) {
	p, ok := r.providers[id]
	return p, ok
}

func (r *Registry) List() []Provider {
	items := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		items = append(items, p)
	}
	return items
}

// FolderLocator supplies an existing provider folder link without creating paths.
type FolderLocator interface {
	FolderURL(context.Context, Connection) (string, error)
}
