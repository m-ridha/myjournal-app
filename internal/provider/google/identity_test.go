package google

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"
)

// Exercise the actual Google validator with a synthetic signing key and local
// certificate transport. No real credentials or Google requests are involved.
func TestSignedIdentityValidation(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	b64 := base64.RawURLEncoding.EncodeToString
	jwks, _ := json.Marshal(map[string]any{"keys": []map[string]string{{"kty": "RSA", "alg": "RS256", "use": "sig", "kid": "test-key", "n": b64(key.N.Bytes()), "e": b64(big.NewInt(int64(key.E)).Bytes())}}})
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://www.googleapis.com/oauth2/v3/certs" {
			t.Error("unexpected certificate endpoint")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}, "Cache-Control": {"max-age=3600"}}, Body: io.NopCloser(strings.NewReader(string(jwks)))}, nil
	})}
	validator, err := idtoken.NewValidator(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		t.Fatal(err)
	}
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "test-key"})
	payload, _ := json.Marshal(map[string]any{"iss": "accounts.google.com", "aud": "client", "sub": "account", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(), "nonce": "nonce"})
	unsigned := b64(header) + "." + b64(payload)
	sum := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	token := unsigned + "." + b64(signature)
	got, err := validator.Validate(context.Background(), token, "client")
	if err != nil || got.Subject != "account" {
		t.Fatal("valid signed identity rejected", err)
	}
	if _, err := validator.Validate(context.Background(), token, "other-client"); err == nil {
		t.Fatal("wrong audience accepted")
	}
	signature[0] ^= 1
	if _, err := validator.Validate(context.Background(), unsigned+"."+b64(signature), "client"); err == nil {
		t.Fatal("invalid signature accepted")
	}
}
