package session

import (
	"github/mridha/myjournal-app/internal/provider"
	"testing"
	"time"
)

func TestFixedSessionLifetime(t *testing.T) {
	issued := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	s := Session{Connection: provider.Connection{ProviderID: "local", AccountID: "dev"}, IssuedAt: issued}
	for _, tc := range []struct {
		now   time.Time
		valid bool
	}{
		{issued.Add(-time.Nanosecond), false}, {issued, true},
		{issued.Add(Lifetime - time.Nanosecond), true}, {issued.Add(Lifetime), false}, {issued.Add(Lifetime + time.Second), false},
	} {
		if got := s.ValidAt(tc.now); got != tc.valid {
			t.Fatalf("now=%v valid=%v", tc.now, got)
		}
	}
	s.IssuedAt = time.Time{}
	if s.ValidAt(issued) {
		t.Fatal("missing issuance accepted")
	}
}

func TestKeyRotationRejectsPreviousCookie(t *testing.T) {
	s := Session{Connection: provider.Connection{ProviderID: "local", AccountID: "dev"}, IssuedAt: time.Now()}
	encoded, err := Encode("original-key", s)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode("rotated-key", encoded); err == nil {
		t.Fatal("old-key cookie accepted")
	}
}
