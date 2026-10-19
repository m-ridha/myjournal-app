package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	provider "myjournal-app/internal/core/connection"
	"myjournal-app/internal/core/journal"
	"myjournal-app/internal/driver/web/authsession"
)

func TestProtectedRequestRequiresOriginalBrowserAccount(t *testing.T) {
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name, scope string
		mismatch    bool
	}{
		{name: "ordinary API client"}, {name: "same account", scope: "same"}, {name: "different account", scope: "other", mismatch: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codec, catalog, current := boundarySession(t, now)
			expectBoundarySession(codec, catalog, current)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/entries", nil)
			req.AddCookie(&http.Cookie{Name: authsession.CookieName, Value: "synthetic"})
			if tc.scope == "same" {
				req.Header.Set("X-Myjournal-Account", authsession.DraftScope(current))
			} else if tc.scope != "" {
				req.Header.Set("X-Myjournal-Account", tc.scope)
			}
			called := false
			server := New(Options{Codec: codec}, catalog, func(_ context.Context, _ provider.Session) (JournalService, error) {
				called = true
				return newMockJournalService(t), nil
			}, testCatalog(t), testClock(func() time.Time { return now }), newMockConnectionService(t))
			_, err := server.service(req)
			if tc.mismatch {
				require.ErrorIs(t, err, journal.ErrAuthRequired)
				assert.False(t, called)
			} else {
				require.NoError(t, err)
				assert.True(t, called)
			}
		})
	}
}
