package connection

import "time"

const Lifetime = 30 * 24 * time.Hour

// ValidAt enforces the fixed lifetime using the original issuance timestamp.
func (s Session) ValidAt(now time.Time) bool {
	return s.ProviderID != "" && s.AccountID != "" && !s.IssuedAt.IsZero() && !s.IssuedAt.After(now) && now.Before(s.IssuedAt.Add(Lifetime))
}

// Session is the user-specific provider connection stored in an encrypted cookie.
type Session struct {
	Connection
	IssuedAt time.Time
}
