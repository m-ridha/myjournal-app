package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"myjournal-app/internal/config"
	provider "myjournal-app/internal/core/connection"
	"myjournal-app/internal/driven/clock"
)

func TestCanonicalGoogleLogin(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name     string
		base     string
		method   string
		target   string
		redirect bool
	}
	for _, tc := range []testCase{
		{name: "previous host", base: "https://journal.example", method: http.MethodGet, target: "https://previous.example/api/v1/auth/google/login", redirect: true},
		{name: "discard query", base: "https://journal.example/", method: http.MethodGet, target: "https://previous.example/api/v1/auth/google/login?return_to=https://untrusted.example", redirect: true},
		{name: "canonical host", base: "https://journal.example", method: http.MethodGet, target: "https://journal.example/api/v1/auth/google/login"},
		{name: "case insensitive host", base: "https://journal.example", method: http.MethodGet, target: "https://JOURNAL.EXAMPLE/api/v1/auth/google/login"},
		{name: "callback stays on original host", base: "https://journal.example", method: http.MethodGet, target: "https://previous.example/api/v1/auth/google/callback?code=synthetic"},
		{name: "editor retains recovery access", base: "https://journal.example", method: http.MethodGet, target: "https://previous.example/entries/2020-01-01"},
		{name: "unsafe methods unchanged", base: "https://journal.example", method: http.MethodPost, target: "https://previous.example/api/v1/auth/google/login"},
		{name: "local development unchanged", base: "http://localhost:3030", method: http.MethodGet, target: "http://127.0.0.1:3030/api/v1/auth/google/login"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			called := false
			h := canonicalGoogleLogin(tc.base)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(tc.method, tc.target, nil)
			req.Header.Set("X-Forwarded-Host", "untrusted.example")
			res := httptest.NewRecorder()
			h.ServeHTTP(res, req)
			assert.Equal(t, !tc.redirect, called)
			assert.Empty(t, res.Result().Cookies())
			if tc.redirect {
				assert.Equal(t, http.StatusFound, res.Code)
				assert.Equal(t, "https://journal.example/api/v1/auth/google/login", res.Header().Get("Location"))
			} else {
				assert.Equal(t, http.StatusNoContent, res.Code)
				assert.Empty(t, res.Header().Get("Location"))
			}
		})
	}
}

func TestRouterCanonicalizesGoogleLoginBeforeProviderAccess(t *testing.T) {
	t.Parallel()
	h := newTestRouter(config.Config{BaseURL: "https://journal.example", SessionKey: "synthetic"}, provider.NewRegistry(), clock.System{}, nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "https://previous.example/api/v1/auth/google/login", nil))
	require.Equal(t, http.StatusFound, res.Code)
	assert.Equal(t, "https://journal.example/api/v1/auth/google/login", res.Header().Get("Location"))
	assert.Empty(t, res.Result().Cookies())
	assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
}
