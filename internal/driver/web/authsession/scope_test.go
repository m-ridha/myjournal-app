package authsession

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	provider "myjournal-app/internal/core/connection"
	"myjournal-app/internal/core/journal"
)

func TestReadRejectsMissingAndOversizedCookiesBeforeDecode(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value string
	}{
		{name: "missing"},
		{name: "oversized", value: strings.Repeat("x", MaxCookieBytes+1)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.value != "" {
				request.AddCookie(&http.Cookie{Name: CookieName, Value: tt.value})
			}
			codec := newMockCodec(t)
			current, err := Read(request, codec, provider.NewRegistry(), time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC))
			require.ErrorIs(t, err, journal.ErrAuthRequired)
			assert.Equal(t, provider.Session{}, current)
		})
	}
}

func TestDraftScopeIdentity(t *testing.T) {
	original := provider.Session{Connection: provider.Connection{ProviderID: "google", AccountID: "account", Credentials: "old"}}
	scope := DraftScope(original)
	assert.Regexp(t, `^[0-9a-f]{64}$`, scope)
	assert.Equal(t, scope, DraftScope(original))
	for _, tt := range []struct {
		name       string
		providerID string
		accountID  string
		wantSame   bool
	}{
		{"refreshed credentials", "google", "account", true},
		{"different provider", "local", "account", false},
		{"different account", "google", "another", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			current := provider.Session{Connection: provider.Connection{ProviderID: tt.providerID, AccountID: tt.accountID, Credentials: "new"}}
			if tt.wantSame {
				assert.Equal(t, scope, DraftScope(current))
			} else {
				assert.NotEqual(t, scope, DraftScope(current))
			}
		})
	}
	left := provider.Session{Connection: provider.Connection{ProviderID: "a:b", AccountID: "c"}}
	right := provider.Session{Connection: provider.Connection{ProviderID: "a", AccountID: "b:c"}}
	assert.NotEqual(t, DraftScope(left), DraftScope(right), "identity components must remain unambiguous")
}
