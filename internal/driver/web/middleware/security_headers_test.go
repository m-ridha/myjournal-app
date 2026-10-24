package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		cacheControl string
		wantCache    string
	}{
		{name: "page", path: "/", wantCache: "no-store"},
		{name: "API replaces existing cache policy", path: "/api/v1/entries", cacheControl: "public", wantCache: "no-store"},
		{name: "static asset without cache policy", path: "/static/css/app.css"},
		{name: "static asset preserves cache policy", path: "/static/js/app.js", cacheControl: "public, max-age=3600", wantCache: "public, max-age=3600"},
		{name: "similar prefix is not static", path: "/static-private/entry", wantCache: "no-store"},
		{name: "bare static path", path: "/static", wantCache: "no-store"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			recorder := httptest.NewRecorder()
			if tt.cacheControl != "" {
				recorder.Header().Set("Cache-Control", tt.cacheControl)
			}
			calls := 0
			handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				assert.Same(t, request, r)
				assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
				w.WriteHeader(http.StatusAccepted)
				_, err := w.Write([]byte("handled"))
				assert.NoError(t, err)
			}))

			handler.ServeHTTP(recorder, request)

			response := recorder.Result()
			t.Cleanup(func() {
				assert.NoError(t, response.Body.Close())
			})
			assert.Equal(t, 1, calls)
			assert.Equal(t, http.StatusAccepted, response.StatusCode)
			assert.Equal(t, "handled", recorder.Body.String())
			assert.Equal(t, "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'", response.Header.Get("Content-Security-Policy"))
			assert.Equal(t, "nosniff", response.Header.Get("X-Content-Type-Options"))
			assert.Equal(t, "DENY", response.Header.Get("X-Frame-Options"))
			assert.Equal(t, "no-referrer", response.Header.Get("Referrer-Policy"))
			assert.Equal(t, tt.wantCache, response.Header.Get("Cache-Control"))
			assert.Empty(t, response.Header.Get("Strict-Transport-Security"))
		})
	}
}

func TestStrictTransportSecurity(t *testing.T) {
	for _, target := range []string{"https://example.com/", "http://example.com/"} {
		t.Run(target, func(t *testing.T) {
			// HTTP covers TLS termination at a proxy. Registration depends on
			// the configured public origin, not the incoming transport.
			request := httptest.NewRequest(http.MethodGet, target, nil)
			recorder := httptest.NewRecorder()
			recorder.Header().Set("Strict-Transport-Security", "max-age=0")
			calls := 0
			handler := StrictTransportSecurity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				assert.Same(t, request, r)
				assert.Equal(t, "max-age=31536000", w.Header().Get("Strict-Transport-Security"))
				w.WriteHeader(http.StatusNoContent)
			}))

			handler.ServeHTTP(recorder, request)

			response := recorder.Result()
			t.Cleanup(func() {
				assert.NoError(t, response.Body.Close())
			})
			assert.Equal(t, 1, calls)
			assert.Equal(t, http.StatusNoContent, response.StatusCode)
			assert.Equal(t, "max-age=31536000", response.Header.Get("Strict-Transport-Security"))
		})
	}
}
