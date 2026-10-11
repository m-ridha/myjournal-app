package connection

import "time"

// Lifetime is the fixed maximum age from original session issuance.
const Lifetime = 30 * 24 * time.Hour

// ValidAt enforces the fixed lifetime using the original issuance timestamp.
func (s Session) ValidAt(now time.Time) bool {
	return s.ProviderID != "" && s.AccountID != "" && !s.IssuedAt.IsZero() && !s.IssuedAt.After(now) && now.Before(s.IssuedAt.Add(Lifetime))
}

// Session is an issued provider connection with a fixed lifetime.
type Session struct {
	Connection
	IssuedAt time.Time
}
