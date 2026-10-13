package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github/m-ridha/myjournal-app/internal/config"
	provider "github/m-ridha/myjournal-app/internal/core/connection"
	localProvider "github/m-ridha/myjournal-app/internal/driven/provider/local"
	"github/m-ridha/myjournal-app/internal/testutil"
)

func TestSignInOffersLoginWithOrWithoutExistingSession(t *testing.T) {
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	cfg := config.Config{BaseURL: testutil.LocalBaseURL, SessionKey: "synthetic-key"}
	router := newTestRouter(cfg, provider.NewRegistry(localProvider.New(testutil.LocalBaseURL, t.TempDir())), testutil.StubClock{Time: now}, nil)
	for _, signedIn := range []bool{false, true} {
		t.Run(map[bool]string{false: "signed out", true: "existing session"}[signedIn], func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/sign-in", nil)
			if signedIn {
				addLocalSessionCookie(t, req, cfg.SessionKey, now)
			}
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			assert.Equal(t, http.StatusOK, res.Code)
			assert.Equal(t, "no-store", res.Header().Get("Cache-Control"))
			assert.Equal(t, "noindex", res.Header().Get("X-Robots-Tag"))
			assert.Empty(t, res.Result().Cookies())
			assert.Contains(t, res.Body.String(), `href="/api/v1/auth/local/login"`)
			assert.Contains(t, res.Body.String(), `href="/sign-in" target="_blank" rel="noopener noreferrer"`)
			assert.Contains(t, res.Body.String(), "Sign in with the same account")
		})
	}
}

func TestSignInLanguageReturnPath(t *testing.T) {
	assert.Equal(t, "/sign-in", languageReturnPath("/sign-in"))
}
