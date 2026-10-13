package session

import (
	"testing"
	"time"

	provider "github/mridha/myjournal-app/internal/core/connection"
)

func TestInjectedCodecPreservesFormatAndPurpose(t *testing.T) {
	codec := NewCodec("synthetic-key")
	current := provider.Session{Connection: provider.Connection{ProviderID: "google", AccountID: "synthetic"}, IssuedAt: time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)}
	value, err := codec.Encode(current)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := Decode("synthetic-key", value); err != nil || got != current {
		t.Fatal("session format changed")
	}
	flow := provider.LoginFlow{ProviderID: "google", State: "synthetic-state", IssuedAt: current.IssuedAt}
	pending, err := codec.EncodeLoginFlow(flow)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := DecodeLoginFlow("synthetic-key", pending); err != nil || got != flow {
		t.Fatal("pending-login format changed")
	}
	if _, err := codec.Decode(pending); err == nil {
		t.Fatal("login cookie accepted as session")
	}
	if _, err := codec.DecodeLoginFlow(value); err == nil {
		t.Fatal("session accepted as login cookie")
	}
	if _, err := NewCodec("different-key").Decode(value); err == nil {
		t.Fatal("wrong secret accepted")
	}
}
