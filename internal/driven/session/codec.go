package session

import provider "github/mridha/myjournal-app/internal/core/connection"

// Codec binds the configured secret to the existing authenticated cookie format.
// Transport consumers receive its methods, never the raw key.
type Codec struct{ secret string }

// NewCodec constructs a codec using the startup-validated session secret.
func NewCodec(secret string) *Codec { return &Codec{secret: secret} }

// Encode encrypts a session using the existing cookie format.
func (c *Codec) Encode(value provider.Session) (string, error) { return Encode(c.secret, value) }

// Decode authenticates and decrypts a session cookie.
func (c *Codec) Decode(value string) (provider.Session, error) { return Decode(c.secret, value) }

// EncodeLoginFlow encrypts a pending login with its distinct authenticated purpose.
func (c *Codec) EncodeLoginFlow(value provider.LoginFlow) (string, error) {
	return EncodeLoginFlow(c.secret, value)
}

// DecodeLoginFlow authenticates and decrypts a pending login cookie.
func (c *Codec) DecodeLoginFlow(value string) (provider.LoginFlow, error) {
	return DecodeLoginFlow(c.secret, value)
}
