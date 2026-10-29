package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestAuthority(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, baseURL, authority string
		allowed                  bool
	}{
		{name: "configured host", baseURL: "http://localhost:3030", authority: "localhost:3030", allowed: true},
		{name: "case insensitive host", baseURL: "http://LOCALHOST:3030", authority: "LocalHost:3030", allowed: true},
		{name: "rebound host", baseURL: "http://localhost:3030", authority: "attacker.example:3030"},
		{name: "hostname suffix", baseURL: "http://localhost:3030", authority: "localhost.evil:3030"},
		{name: "unconfigured loopback alias", baseURL: "http://localhost:3030", authority: "127.0.0.1:3030"},
		{name: "wrong port", baseURL: "http://localhost:3030", authority: "localhost:4040"},
		{name: "missing port", baseURL: "http://localhost:3030", authority: "localhost"},
		{name: "missing host", baseURL: "http://localhost:3030"},
		{name: "userinfo", baseURL: "http://localhost:3030", authority: "attacker@localhost:3030"},
		{name: "multiple authorities", baseURL: "http://localhost:3030", authority: "localhost:3030,attacker.example"},
		{name: "IPv4", baseURL: "http://127.0.0.1:3030", authority: "127.0.0.1:3030", allowed: true},
		{name: "IPv6", baseURL: "http://[::1]:3030", authority: "[::1]:3030", allowed: true},
		{name: "IPv6 wrong port", baseURL: "http://[::1]:3030", authority: "[::1]:4040"},
		{name: "HTTP implicit default", baseURL: "http://localhost", authority: "localhost", allowed: true},
		{name: "HTTP explicit request default", baseURL: "http://localhost", authority: "localhost:80", allowed: true},
		{name: "HTTP explicit configured default", baseURL: "http://localhost:80", authority: "localhost", allowed: true},
		{name: "HTTPS implicit default", baseURL: "https://localhost", authority: "localhost", allowed: true},
		{name: "HTTPS explicit request default", baseURL: "https://localhost", authority: "localhost:443", allowed: true},
		{name: "HTTPS explicit configured default", baseURL: "https://localhost:443", authority: "localhost", allowed: true},
		{name: "HTTPS wrong default", baseURL: "https://localhost", authority: "localhost:80"},
		{name: "IPv6 implicit default", baseURL: "http://[::1]:80", authority: "[::1]", allowed: true},
		{name: "IPv6 explicit default", baseURL: "https://[::1]", authority: "[::1]:443", allowed: true},
		{name: "configured trailing dot", baseURL: "http://localhost.:3030/", authority: "localhost.:3030", allowed: true},
		{name: "invalid configuration", baseURL: "://", authority: "localhost:3030"},
		{name: "missing configuration", authority: "localhost:3030"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			called := false
			h := RequestAuthority(tc.baseURL)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/local/callback", nil)
			req.Host = tc.authority
			req.Header.Set("Forwarded", "host=localhost:3030;proto=http")
			req.Header.Set("X-Forwarded-Host", "localhost:3030")
			req.Header.Set("X-Forwarded-Proto", "https")
			res := httptest.NewRecorder()
			h.ServeHTTP(res, req)

			assert.Equal(t, tc.allowed, called)
			assert.Empty(t, res.Result().Cookies())
			assert.Empty(t, res.Header().Get("Location"))
			if tc.allowed {
				assert.Equal(t, http.StatusNoContent, res.Code)
			} else {
				assert.Equal(t, http.StatusForbidden, res.Code)
				assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
				assert.JSONEq(t, `{"message":"Forbidden"}`, res.Body.String())
			}
		})
	}
}
