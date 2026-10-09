package journal

import (
	"context"
	"time"
)

// Store is implemented by each storage provider and by cache adapters.
type Store interface {
	ListYears(ctx context.Context) ([]int, error)
	ListYear(ctx context.Context, year int) ([]EntryMeta, error)
	Get(ctx context.Context, d Date) (Entry, error)
	// Previews returns UTF-8-safe first-meaningful-line prefixes, at most maxBytes each.
	// Nonpositive budgets return empty prefixes. Missing entries are omitted; other errors propagate.
	Previews(ctx context.Context, dates []Date, maxBytes int) (map[Date]string, error)
	Put(ctx context.Context, d Date, text string, base Version) (Entry, error)
	Trash(ctx context.Context, d Date) error
}

// Clock supplies the current time to domain services.
type Clock interface {
	Now() time.Time
}
