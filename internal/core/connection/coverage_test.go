package connection

import (
	"context"
	"errors"
	"testing"
	"time"

	"github/m-ridha/myjournal-app/internal/core/journal"
)

func TestRegistryLookupAndEnumeration(t *testing.T) {
	one, two := newMockProvider(t), newMockProvider(t)
	one.EXPECT().ID().Return("one")
	two.EXPECT().ID().Return("two")
	r := NewRegistry(nil, one, two)
	if len(r.List()) != 2 {
		t.Fatal("registry lost configured providers")
	}
	for _, p := range r.List() {
		if got, ok := r.Get(p.ID()); !ok || got.ID() != p.ID() {
			t.Fatal("enumerated provider cannot be resolved")
		}
		if got, ok := r.Lookup(p.ID()); !ok || got.ID() != p.ID() {
			t.Fatal("lifecycle lookup differs")
		}
	}
	if _, ok := r.Get("missing"); ok {
		t.Fatal("unknown provider resolved")
	}
	if _, ok := r.Lookup("missing"); ok {
		t.Fatal("unknown gateway resolved")
	}
	rejection := &LoginError{Stage: LoginClaims, Err: journal.ErrAuthRequired}
	if rejection.Error() != "sign-in rejected" || !errors.Is(rejection, journal.ErrAuthRequired) {
		t.Fatal("login error exposes details or loses identity")
	}
}

func TestUnknownProviderLifecycleAndOrigins(t *testing.T) {
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	s := NewService(NewRegistry(), fixedClock{now}, nil)
	if _, err := s.Complete(context.Background(), CompleteInput{ProviderID: "missing"}); !errors.Is(err, ErrProviderUnavailable) {
		t.Fatal(err)
	}
	current := Session{Connection: Connection{ProviderID: "missing", AccountID: "account"}, IssuedAt: now}
	if err := s.Disconnect(context.Background(), current); !errors.Is(err, journal.ErrAuthRequired) {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		origin string
		valid  bool
	}{{"http://localhost:3030", true}, {"http://127.0.0.1", true}, {"http://[::1]", true}, {"http://localhost.evil.invalid", false}, {"http://user@localhost", false}, {"http://localhost/path", false}, {"http://localhost?x=y", false}, {"%", false}} {
		if err := ValidateLocalOrigin(tc.origin); (err == nil) != tc.valid {
			t.Fatalf("origin %q: %v", tc.origin, err)
		}
	}
}
