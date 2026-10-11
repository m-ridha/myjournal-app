package session

import (
	provider "github/mridha/myjournal-app/internal/core/connection"
	"testing"
	"time"
)

func TestKeyRotationRejectsPreviousCookie(t *testing.T) {
	s := provider.Session{Connection: provider.Connection{ProviderID: "local", AccountID: "dev"}, IssuedAt: time.Now()}
	encoded, err := Encode("original-key", s)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode("rotated-key", encoded); err == nil {
		t.Fatal("old-key cookie accepted")
	}
}
