package connection

import (
	"encoding/base64"
	"io"
	"time"
)

const LoginLifetime = 10 * time.Minute

// LoginFlow binds a browser's pending authorization to its provider and PKCE secret.
// It is separate from the authenticated session and never extends its lifetime.
type LoginFlow struct {
	ProviderID string
	State      string
	Verifier   string
	Nonce      string
	IssuedAt   time.Time
}

func NewLoginFlow(providerID string, now time.Time, entropySource io.Reader) (LoginFlow, error) {
	flow := LoginFlow{ProviderID: providerID, IssuedAt: now.UTC()}
	for _, target := range []*string{&flow.State, &flow.Verifier, &flow.Nonce} {
		entropy := make([]byte, 32)
		if _, err := io.ReadFull(entropySource, entropy); err != nil {
			return LoginFlow{}, err
		}
		*target = base64.RawURLEncoding.EncodeToString(entropy)
	}
	return flow, nil
}

func (f LoginFlow) ValidAt(now time.Time) bool {
	if f.ProviderID == "" || len(f.ProviderID) > 64 || f.IssuedAt.IsZero() || f.IssuedAt.After(now) || !now.Before(f.IssuedAt.Add(LoginLifetime)) {
		return false
	}
	for _, token := range []string{f.State, f.Verifier, f.Nonce} {
		b, err := base64.RawURLEncoding.DecodeString(token)
		if err != nil || len(b) != 32 || base64.RawURLEncoding.EncodeToString(b) != token {
			return false
		}
	}
	return true
}
