package session

import (
	"crypto/rand"
	"errors"
	"io"
	"testing"

	provider "myjournal-app/internal/core/connection"
)

type failingEntropy struct{}

func (failingEntropy) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestEncodingRejectsUnsupportedPayloadAndEntropyFailure(t *testing.T) {
	if _, err := encode("synthetic", make(chan int), nil); err == nil {
		t.Fatal("unsupported payload encoded")
	}
	previous := rand.Reader
	rand.Reader = failingEntropy{}
	defer func() { rand.Reader = previous }()
	if _, err := Encode("synthetic", provider.Session{}); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatal("entropy failure lost")
	}
}
func TestAuthenticatedWrongPayloadShapeIsRejected(t *testing.T) {
	value, err := encode("synthetic", []int{1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode("synthetic", value); err == nil {
		t.Fatal("authenticated incompatible payload accepted")
	}
}
