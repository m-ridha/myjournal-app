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

	provider "github/mridha/myjournal-app/internal/core/connection"
)

const CookieName = "mj_session"

func keyFromSecret(secret string) []byte {
	out := sha256.Sum256([]byte(secret))
	return out[:]
}

// Encode encrypts and signs the session payload for storage in a cookie.
func Encode(secret string, s provider.Session) (string, error) {
	return encode(secret, s, nil)
}

func encode(secret string, value any, aad []byte) (string, error) {
	block, err := aes.NewCipher(keyFromSecret(secret))
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal session: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("read nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, payload, aad)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// Decode verifies and decrypts a session from a cookie value.
func Decode(secret, value string) (provider.Session, error) {
	var s provider.Session
	err := decode(secret, value, nil, &s)
	return s, err
}

func decode(secret, value string, aad []byte, target any) error {
	if value == "" {
		return errors.New("empty session cookie")
	}

	ciphertext, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return fmt.Errorf("decode cookie: %w", err)
	}

	block, err := aes.NewCipher(keyFromSecret(secret))
	if err != nil {
		return fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("new gcm: %w", err)
	}
	if len(ciphertext) < gcm.NonceSize() {
		return errors.New("cookie too short")
	}

	nonce, payload := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, payload, aad)
	if err != nil {
		return fmt.Errorf("decrypt session: %w", err)
	}

	if err := json.Unmarshal(plaintext, target); err != nil {
		return fmt.Errorf("unmarshal session: %w", err)
	}
	return nil
}
