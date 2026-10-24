package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	provider "myjournal-app/internal/core/connection"
	"myjournal-app/internal/core/journal"
	"myjournal-app/internal/driver/web/api"
	"myjournal-app/internal/driver/web/authsession"
)

func TestStartProviderLoginFailures(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
	}{
		{"unavailable provider", provider.ErrProviderUnavailable, http.StatusNotFound},
		{"pending transaction failure", provider.ErrLoginFlow, http.StatusInternalServerError},
		{"upstream failure", journal.ErrUnavailable, http.StatusBadGateway},
	} {
		t.Run(tt.name, func(t *testing.T) {
			connections := newMockConnectionService(t)
			request := httptest.NewRequest(http.MethodGet, "/login", nil)
			consent := api.StartProviderLoginParamsForceConsent(1)
			connections.EXPECT().Begin(request.Context(), provider.BeginInput{ProviderID: "google", ForceConsent: true}).Return(provider.BeginResult{}, fmt.Errorf("wrapped: %w", tt.err)).Once()
			server := &Server{connections: connections, messages: testCatalog(t), now: func() time.Time { return time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC) }}
			response := httptest.NewRecorder()
			server.StartProviderLogin(response, request, "google", api.StartProviderLoginParams{ForceConsent: &consent})
			assert.Equal(t, tt.status, response.Code)
			assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
			assert.Empty(t, response.Header().Get("Location"))
			assert.Empty(t, response.Result().Cookies())
			assert.NotContains(t, response.Body.String(), "wrapped:")
		})
	}
}

func TestSessionTermination(t *testing.T) {
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name          string
		disconnect    bool
		authenticated bool
		status        int
	}{
		{"logout", false, false, http.StatusFound},
		{"disconnect success", true, true, http.StatusNoContent},
		{"disconnect without session", true, false, http.StatusUnauthorized},
	} {
		t.Run(tt.name, func(t *testing.T) {
			codec, registry, current := boundarySession(t, now)
			connections := newMockConnectionService(t)
			request := httptest.NewRequest(http.MethodPost, "/session", nil)
			if tt.authenticated {
				request.AddCookie(&http.Cookie{Name: authsession.CookieName, Value: "synthetic"})
				expectBoundarySession(codec, registry, current)
				connections.EXPECT().Disconnect(request.Context(), current).Return(nil).Once()
			}
			server := New(Options{Codec: codec, CookieSecure: true}, registry, nil, testCatalog(t), testClock(func() time.Time { return now }), connections)
			response := httptest.NewRecorder()
			if tt.disconnect {
				server.Disconnect(response, request)
			} else {
				server.Logout(response, request)
			}
			assert.Equal(t, tt.status, response.Code)
			cookies := response.Result().Cookies()
			if tt.status == http.StatusUnauthorized {
				assert.Empty(t, cookies)
				return
			}
			require.Len(t, cookies, 1)
			cookie := cookies[0]
			assert.Equal(t, authsession.CookieName, cookie.Name)
			assert.Empty(t, cookie.Value)
			assert.Equal(t, -1, cookie.MaxAge)
			assert.Equal(t, "/", cookie.Path)
			assert.True(t, cookie.HttpOnly)
			assert.True(t, cookie.Secure)
			assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
			if !tt.disconnect {
				assert.Equal(t, "/", response.Header().Get("Location"))
			}
		})
	}
}
