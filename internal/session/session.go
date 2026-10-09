package session

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github/mridha/myjournal-app/internal/provider"
)

const CookieName = "mj_session"

const Lifetime = 30 * 24 * time.Hour

// ValidAt enforces the fixed lifetime using the original issuance timestamp.
func (s Session) ValidAt(now time.Time) bool {
	return s.ProviderID != "" && s.AccountID != "" && !s.IssuedAt.IsZero() && !s.IssuedAt.After(now) && now.Before(s.IssuedAt.Add(Lifetime))
}

// Session is the user-specific provider connection stored in an encrypted cookie.
type Session struct {
	provider.Connection
	IssuedAt time.Time
}

func keyFromSecret(secret string) []byte {
	out := sha256.Sum256([]byte(secret))
	return out[:]
}

// Encode encrypts and signs the session payload for storage in a cookie.
func Encode(secret string, s Session) (string, error) {
	block, err := aes.NewCipher(keyFromSecret(secret))
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}

	payload, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("marshal session: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("read nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, payload, nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// Decode verifies and decrypts a session from a cookie value.
func Decode(secret, value string) (Session, error) {
	if value == "" {
		return Session{}, errors.New("empty session cookie")
	}

	ciphertext, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Session{}, fmt.Errorf("decode cookie: %w", err)
	}

	block, err := aes.NewCipher(keyFromSecret(secret))
	if err != nil {
		return Session{}, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return Session{}, fmt.Errorf("new gcm: %w", err)
	}
	if len(ciphertext) < gcm.NonceSize() {
		return Session{}, errors.New("cookie too short")
	}

	nonce, payload := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return Session{}, fmt.Errorf("decrypt session: %w", err)
	}

	var s Session
	if err := json.Unmarshal(plaintext, &s); err != nil {
		return Session{}, fmt.Errorf("unmarshal session: %w", err)
	}
	return s, nil
}
