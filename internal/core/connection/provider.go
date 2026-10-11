package connection

import (
	"context"

	"github/mridha/myjournal-app/internal/core/journal"
)

// Stable provider identifiers are shared by configuration, adapters, and delivery.
const (
	LocalID  = "local"
	GoogleID = "google"
)

// Provider is the storage-provider boundary used by the application to
// establish and revoke a provider connection.
type Provider interface {
	Gateway
	Store(ctx context.Context, c Connection) (journal.Store, error)
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

var _ Catalog = (*Registry)(nil)

// NewRegistry registers the configured provider implementations by stable ID.
func NewRegistry(providers ...Provider) *Registry {
	r := &Registry{providers: make(map[string]Provider, len(providers))}
	for _, p := range providers {
		if p != nil {
			r.providers[p.ID()] = p
		}
	}
	return r
}

// Get looks up a registered provider.
func (r *Registry) Get(id string) (Provider, bool) {
	p, ok := r.providers[id]
	return p, ok
}

// List returns registered providers in unspecified order.
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

// LoginStage identifies a bounded, non-sensitive authentication rejection category.
type LoginStage string

const (
	LoginTransaction LoginStage = "transaction"
	LoginExchange    LoginStage = "exchange"
	LoginClaims      LoginStage = "claims"
	LoginGrant       LoginStage = "grant"
	LoginAccount     LoginStage = "account"
	LoginRefresh     LoginStage = "refresh"
)

// LoginError preserves a domain error while exposing only a safe diagnostic stage.
type LoginError struct {
	Stage LoginStage
	Err   error
}

func (e *LoginError) Error() string { return "sign-in rejected" }
func (e *LoginError) Unwrap() error { return e.Err }

// Lookup supplies only lifecycle capabilities to application consumers.
func (r *Registry) Lookup(id string) (Gateway, bool) { p, ok := r.Get(id); return p, ok }
