package session

import (
	"testing"
	"time"

	provider "myjournal-app/internal/core/connection"
)

func TestRoundTrip(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef"
	want := provider.Session{
		Connection: provider.Connection{
			ProviderID:   "local",
			AccountID:    "dev",
			AccountEmail: "dev@localhost",
			Credentials:  "local-dev-user",
		},
		IssuedAt: time.Now().UTC(),
	}

	encoded, err := Encode(key, want)
	if err != nil {
		t.Fatalf("Encode() unexpected error: %v", err)
	}

	got, err := Decode(key, encoded)
	if err != nil {
		t.Fatalf("Decode() unexpected error: %v", err)
	}
	if got.ProviderID != want.ProviderID {
		t.Fatalf("ProviderID = %q, want %q", got.ProviderID, want.ProviderID)
	}
	if got.AccountID != want.AccountID {
		t.Fatalf("AccountID = %q, want %q", got.AccountID, want.AccountID)
	}
	if got.AccountEmail != want.AccountEmail {
		t.Fatalf("AccountEmail = %q, want %q", got.AccountEmail, want.AccountEmail)
	}
	if got.Credentials != want.Credentials {
		t.Fatalf("Credentials = %q, want %q", got.Credentials, want.Credentials)
	}
}

func TestDecodeRejectsTampering(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef"
	encoded, err := Encode(key, provider.Session{Connection: provider.Connection{ProviderID: "local"}})
	if err != nil {
		t.Fatalf("Encode() unexpected error: %v", err)
	}

	if len(encoded) == 0 {
		t.Fatal("Encode() returned empty cookie")
	}

	if _, err := Decode(key, encoded+"tamper"); err == nil {
		t.Fatal("Decode() accepted a tampered cookie")
	}
}
