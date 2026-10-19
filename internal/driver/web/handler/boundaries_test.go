package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	provider "myjournal-app/internal/core/connection"
	"myjournal-app/internal/core/journal"
	"myjournal-app/internal/driver/web/api"
	"myjournal-app/internal/driver/web/authsession"
)

func boundarySession(t *testing.T, now time.Time) (*mockCodec, *mockCatalog, provider.Session) {
	t.Helper()
	current := provider.Session{Connection: provider.Connection{ProviderID: "synthetic", AccountID: "account"}, IssuedAt: now}
	codec := newMockCodec(t)
	catalog := newMockCatalog(t)
	return codec, catalog, current
}

func expectBoundarySession(codec *mockCodec, catalog *mockCatalog, current provider.Session) {
	codec.EXPECT().Decode("synthetic").Return(current, nil).Once()
	catalog.EXPECT().Lookup("synthetic").Return(nil, true).Once()
}

func TestDeliveryUsesInjectedCodecAndConnectionPorts(t *testing.T) {
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	failure := errors.New("synthetic codec failure")
	for _, tc := range []struct {
		name    string
		prepare func(*testing.T, *mockCodec, *mockCatalog, *mockConnectionService, provider.Session)
		call    func(*Server, http.ResponseWriter, *http.Request)
		status  int
	}{
		{"begin", func(t *testing.T, codec *mockCodec, catalog *mockCatalog, connections *mockConnectionService, current provider.Session) {
			expectBoundarySession(codec, catalog, current)
			connections.EXPECT().Begin(mock.Anything, mock.Anything).Return(provider.BeginResult{Pending: &provider.LoginFlow{}}, nil).Once()
			codec.EXPECT().EncodeLoginFlow(provider.LoginFlow{}).Return("", failure).Once()
		}, func(s *Server, w http.ResponseWriter, r *http.Request) {
			s.StartProviderLogin(w, r, "synthetic", api.StartProviderLoginParams{})
		}, 500},
		{"complete", func(t *testing.T, codec *mockCodec, catalog *mockCatalog, connections *mockConnectionService, _ provider.Session) {
			gateway := newMockProvider(t)
			gateway.EXPECT().ID().Return("synthetic").Once()
			gateway.EXPECT().RequiresLoginFlow().Return(false).Once()
			catalog.EXPECT().Lookup("synthetic").Return(gateway, true).Once()
			connections.EXPECT().Complete(mock.Anything, mock.Anything).Return(provider.Session{}, nil).Once()
			codec.EXPECT().Encode(provider.Session{}).Return("", failure).Once()
		}, func(s *Server, w http.ResponseWriter, r *http.Request) {
			s.CompleteProviderLogin(w, r, "synthetic", api.CompleteProviderLoginParams{})
		}, 500},
		{"disconnect", func(t *testing.T, codec *mockCodec, catalog *mockCatalog, connections *mockConnectionService, current provider.Session) {
			expectBoundarySession(codec, catalog, current)
			connections.EXPECT().Disconnect(mock.Anything, current).Return(journal.ErrUnavailable).Once()
		}, func(s *Server, w http.ResponseWriter, r *http.Request) { s.Disconnect(w, r) }, 503},
		{"settings", func(t *testing.T, codec *mockCodec, catalog *mockCatalog, connections *mockConnectionService, current provider.Session) {
			expectBoundarySession(codec, catalog, current)
			connections.EXPECT().Settings(mock.Anything, current).Return(provider.Settings{ProviderName: "Synthetic"}, nil).Once()
		}, func(s *Server, w http.ResponseWriter, r *http.Request) { s.GetSettings(w, r) }, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codec, catalog, current := boundarySession(t, now)
			connections := newMockConnectionService(t)
			tc.prepare(t, codec, catalog, connections, current)
			server := New(Options{Codec: codec}, catalog, nil, testCatalog(t), testClock(func() time.Time { return now }), connections)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/synthetic", nil)
			req.AddCookie(&http.Cookie{Name: authsession.CookieName, Value: "synthetic"})
			response := httptest.NewRecorder()
			tc.call(server, response, req)
			assert.Equal(t, tc.status, response.Code)
			assert.Empty(t, response.Result().Cookies())
		})
	}
}
