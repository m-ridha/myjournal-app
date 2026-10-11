package connection

import "context"

// Gateway is the outbound capability consumed by connection workflows.
// Storage construction belongs to the composition root's Provider bundle.
type Gateway interface {
	ID() string
	Name() string
	Begin(context.Context, LoginRequest) (string, error)
	Complete(context.Context, LoginCallback) (Connection, error)
	Revoke(context.Context, Connection) error
}

// Catalog resolves configured lifecycle gateways without exposing storage creation.
type Catalog interface{ Lookup(string) (Gateway, bool) }
