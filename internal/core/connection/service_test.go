package connection

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github/m-ridha/myjournal-app/internal/core/journal"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func lifecycleGateway(t *testing.T, id string, required bool) (*mockGateway, *mockCatalog) {
	t.Helper()
	gateway := newMockGateway(t)
	catalog := newMockCatalog(t)
	catalog.EXPECT().Lookup(id).Return(gateway, true)
	gateway.EXPECT().ID().Return(id).Maybe()
	gateway.EXPECT().RequiresLoginFlow().Return(required).Maybe()
	return gateway, catalog
}

func TestLifecyclePolicy(t *testing.T) {
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	g, catalog := lifecycleGateway(t, GoogleID, true)
	var request LoginRequest
	g.EXPECT().Begin(mock.Anything, mock.Anything).Run(func(_ context.Context, r LoginRequest) { request = r }).Return("https://example.invalid/login", nil).Once()
	s := NewService(catalog, fixedClock{now}, bytes.NewReader(bytes.Repeat([]byte{1}, 1024)))
	previous := Session{Connection: Connection{ProviderID: GoogleID, AccountID: "synthetic"}, IssuedAt: now.Add(-time.Hour)}
	begin, err := s.Begin(t.Context(), BeginInput{ProviderID: GoogleID, Previous: &previous})
	require.NoError(t, err)
	require.NotNil(t, begin.Pending)
	assert.False(t, request.ForceConsent)
	for _, tc := range []struct {
		name  string
		flow  LoginFlow
		state string
	}{
		{"expired", func() LoginFlow { f := *begin.Pending; f.IssuedAt = now.Add(-LoginLifetime); return f }(), begin.Pending.State},
		{"future", func() LoginFlow { f := *begin.Pending; f.IssuedAt = now.Add(time.Second); return f }(), begin.Pending.State},
		{"provider", func() LoginFlow { f := *begin.Pending; f.ProviderID = LocalID; return f }(), begin.Pending.State},
		{"state", *begin.Pending, "wrong"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.Complete(t.Context(), CompleteInput{ProviderID: GoogleID, Pending: &tc.flow, State: tc.state, Code: "synthetic"})
			require.ErrorIs(t, err, journal.ErrAuthRequired)
			g.AssertNotCalled(t, "Complete", mock.Anything, mock.Anything)
		})
	}
	for _, tc := range []struct {
		name     string
		issued   time.Time
		eligible bool
	}{
		{"valid previous", now.Add(-time.Hour), true}, {"expired previous", now.Add(-Lifetime), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			previous.IssuedAt = tc.issued
			var callback LoginCallback
			g.EXPECT().Complete(mock.Anything, mock.Anything).Run(func(_ context.Context, c LoginCallback) { callback = c }).Return(Connection{ProviderID: GoogleID, AccountID: "synthetic"}, nil).Once()
			current, err := s.Complete(t.Context(), CompleteInput{ProviderID: GoogleID, Pending: begin.Pending, State: begin.Pending.State, Code: "synthetic", Previous: &previous})
			require.NoError(t, err)
			assert.Equal(t, now, current.IssuedAt)
			assert.Equal(t, tc.eligible, callback.Previous != nil)
		})
	}
}

type locatedGateway struct {
	*mockGateway
	*mockFolderLocator
}

func TestDisconnectAndSettings(t *testing.T) {
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	current := Session{Connection: Connection{ProviderID: GoogleID, AccountID: "synthetic", AccountEmail: "synthetic@example.invalid"}, IssuedAt: now}
	g := &locatedGateway{newMockGateway(t), newMockFolderLocator(t)}
	catalog := newMockCatalog(t)
	catalog.EXPECT().Lookup(GoogleID).Return(g, true)
	g.mockGateway.EXPECT().Name().Return("Google")
	failure := assert.AnError
	g.mockGateway.EXPECT().Revoke(mock.Anything, current.Connection).Return(failure).Once()
	s := NewService(catalog, fixedClock{now}, nil)
	require.ErrorIs(t, s.Disconnect(t.Context(), current), failure)
	g.mockFolderLocator.EXPECT().FolderURL(mock.Anything, current.Connection).Return("https://example.invalid/folder", nil).Once()
	result, err := s.Settings(t.Context(), current)
	require.NoError(t, err)
	assert.Equal(t, Settings{ProviderName: "Google", AccountName: current.AccountEmail, FolderURL: "https://example.invalid/folder"}, result)
	g.mockFolderLocator.EXPECT().FolderURL(mock.Anything, current.Connection).Return("", failure).Once()
	_, err = s.Settings(t.Context(), current)
	require.ErrorIs(t, err, failure)
	current.IssuedAt = now.Add(-Lifetime)
	require.ErrorIs(t, s.Disconnect(t.Context(), current), journal.ErrAuthRequired)
}

func TestLifecycleUsesGatewayWithoutStorageConstruction(t *testing.T) {
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	_, catalog := lifecycleGateway(t, GoogleID, true)
	catalog.EXPECT().Lookup("missing").Return(nil, false).Once()
	s := NewService(catalog, fixedClock{now}, bytes.NewReader(nil))
	_, err := s.Begin(t.Context(), BeginInput{ProviderID: GoogleID})
	require.ErrorIs(t, err, ErrLoginFlow)
	_, err = s.Begin(t.Context(), BeginInput{ProviderID: "missing"})
	require.ErrorIs(t, err, ErrProviderUnavailable)
	local, localCatalog := lifecycleGateway(t, LocalID, false)
	local.EXPECT().Begin(mock.Anything, LoginRequest{}).Return("/", nil).Once()
	local.EXPECT().Complete(mock.Anything, LoginCallback{}).Return(Connection{ProviderID: LocalID, AccountID: "synthetic"}, nil).Once()
	local.EXPECT().Name().Return("Local").Once()
	s = NewService(localCatalog, fixedClock{now}, bytes.NewReader(nil))
	result, err := s.Begin(t.Context(), BeginInput{ProviderID: LocalID})
	require.NoError(t, err)
	assert.Nil(t, result.Pending)
	_, err = s.Complete(t.Context(), CompleteInput{ProviderID: LocalID})
	require.NoError(t, err)
	settings, err := s.Settings(t.Context(), Session{Connection: Connection{ProviderID: LocalID, AccountID: "synthetic"}, IssuedAt: now})
	require.NoError(t, err)
	assert.Empty(t, settings.FolderURL)
}

func TestReconnectConsentAndPreviousProviderEligibility(t *testing.T) {
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name                         string
		previous                     *Session
		force, wantConsent, eligible bool
	}{
		{name: "new", wantConsent: true},
		{name: "different provider", previous: &Session{Connection: Connection{ProviderID: LocalID, AccountID: "synthetic"}, IssuedAt: now}, wantConsent: true},
		{name: "expired", previous: &Session{Connection: Connection{ProviderID: GoogleID, AccountID: "synthetic"}, IssuedAt: now.Add(-Lifetime)}, wantConsent: true},
		{name: "forced", previous: &Session{Connection: Connection{ProviderID: GoogleID, AccountID: "synthetic"}, IssuedAt: now}, force: true, wantConsent: true, eligible: true},
		{name: "same provider", previous: &Session{Connection: Connection{ProviderID: GoogleID, AccountID: "synthetic"}, IssuedAt: now}, eligible: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, catalog := lifecycleGateway(t, GoogleID, true)
			var request LoginRequest
			var callback LoginCallback
			g.EXPECT().Begin(mock.Anything, mock.Anything).Run(func(_ context.Context, r LoginRequest) { request = r }).Return("https://example.invalid/login", nil).Once()
			g.EXPECT().Complete(mock.Anything, mock.Anything).Run(func(_ context.Context, c LoginCallback) { callback = c }).Return(Connection{ProviderID: GoogleID, AccountID: "synthetic"}, nil).Once()
			s := NewService(catalog, fixedClock{now}, bytes.NewReader(bytes.Repeat([]byte{1}, 96)))
			begin, err := s.Begin(t.Context(), BeginInput{ProviderID: GoogleID, Previous: tc.previous, ForceConsent: tc.force})
			require.NoError(t, err)
			require.NotNil(t, begin.Pending)
			assert.Equal(t, tc.wantConsent, request.ForceConsent)
			_, err = s.Complete(t.Context(), CompleteInput{ProviderID: GoogleID, Pending: begin.Pending, Code: "synthetic", State: begin.Pending.State, Previous: tc.previous})
			require.NoError(t, err)
			assert.Equal(t, tc.eligible, callback.Previous != nil)
		})
	}
}

func TestLoginFlowUsesCapabilityInsteadOfProviderID(t *testing.T) {
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		id       string
		required bool
	}{{"synthetic", false}, {LocalID, true}} {
		t.Run(tc.id, func(t *testing.T) {
			g, catalog := lifecycleGateway(t, tc.id, tc.required)
			g.EXPECT().Begin(mock.Anything, mock.Anything).Return("/", nil).Once()
			if !tc.required {
				g.EXPECT().Complete(mock.Anything, LoginCallback{}).Return(Connection{ProviderID: tc.id, AccountID: "synthetic"}, nil).Once()
			}
			s := NewService(catalog, fixedClock{now}, bytes.NewReader(bytes.Repeat([]byte{1}, 96)))
			result, err := s.Begin(t.Context(), BeginInput{ProviderID: tc.id})
			require.NoError(t, err)
			assert.Equal(t, tc.required, result.Pending != nil)
			_, err = s.Complete(t.Context(), CompleteInput{ProviderID: tc.id})
			if tc.required {
				require.ErrorIs(t, err, journal.ErrAuthRequired)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
