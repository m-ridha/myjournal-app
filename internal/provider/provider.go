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
	Begin(ctx context.Context, state string) (string, error)
	Complete(ctx context.Context, code string) (Connection, error)
	Store(ctx context.Context, c Connection) (journal.Store, error)
	Revoke(ctx context.Context, c Connection) error
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
