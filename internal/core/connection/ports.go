package connection

import "context"

// Gateway is the outbound capability consumed by connection workflows.
// Storage construction belongs to the composition root's Provider bundle.
type Gateway interface {
	ID() string
	Name() string
	Begin(ctx context.Context, input LoginRequest) (string, error)
	Complete(ctx context.Context, input LoginCallback) (Connection, error)
	Revoke(ctx context.Context, c Connection) error
}

// Catalog resolves configured lifecycle gateways without exposing storage creation.
type Catalog interface{ Lookup(string) (Gateway, bool) }
