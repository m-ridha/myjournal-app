package session

import (
	"crypto/rand"
	"testing"
	"time"

	provider "github/m-ridha/myjournal-app/internal/core/connection"
)

func TestLoginFlowLifecycle(t *testing.T) {
	now := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	flow, err := provider.NewLoginFlow("google", now, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	other, err := provider.NewLoginFlow("google", now, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if flow.State == other.State || flow.Verifier == other.Verifier || flow.Nonce == other.Nonce || flow.State == flow.Verifier {
		t.Fatal("login entropy must be independent")
	}
	encoded, err := EncodeLoginFlow("test-key", flow)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeLoginFlow("test-key", encoded)
	if err != nil || got != flow {
		t.Fatal("login flow failed round trip")
	}
	if !got.ValidAt(now) || got.ValidAt(now.Add(provider.LoginLifetime)) || got.ValidAt(now.Add(-time.Second)) {
		t.Fatal("incorrect login age bounds")
	}
	if _, err := DecodeLoginFlow("other-key", encoded); err == nil {
		t.Fatal("rotated key accepted")
	}
	if _, err := Decode("test-key", encoded); err == nil {
		t.Fatal("login cookie accepted as a session")
	}
	sessionValue, err := Encode("test-key", provider.Session{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeLoginFlow("test-key", sessionValue); err == nil {
		t.Fatal("session accepted as login cookie")
	}
	bytes := []byte(encoded)
	bytes[len(bytes)/2] ^= 1
	if _, err := DecodeLoginFlow("test-key", string(bytes)); err == nil {
		t.Fatal("tampered login accepted")
	}
	got.Verifier = "short"
	if got.ValidAt(now) {
		t.Fatal("malformed verifier accepted")
	}
}
