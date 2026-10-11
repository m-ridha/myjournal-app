package journal

import (
	"context"
	"time"
)

type timeZoneKey struct{}

// WithTimeZone supplies the resolved user location without coupling the domain
// to HTTP or browser cookies. Missing/nil locations use UTC.
func WithTimeZone(ctx context.Context, location *time.Location) context.Context {
	if location == nil {
		location = time.UTC
	}
	return context.WithValue(ctx, timeZoneKey{}, location)
}

// TimeZone returns the location resolved for this operation, or UTC.
func TimeZone(ctx context.Context) *time.Location {
	if location, ok := ctx.Value(timeZoneKey{}).(*time.Location); ok && location != nil {
		return location
	}
	return time.UTC
}
