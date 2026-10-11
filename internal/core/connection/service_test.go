package connection

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github/mridha/myjournal-app/internal/core/journal"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type gateway struct {
	calls     int
	request   LoginRequest
	callback  LoginCallback
	revokeErr error
	folderErr error
}

func (*gateway) ID() string   { return GoogleID }
func (*gateway) Name() string { return "Google" }
func (g *gateway) Begin(_ context.Context, r LoginRequest) (string, error) {
	g.request = r
	return "https://example.invalid/login", nil
}
func (g *gateway) Complete(_ context.Context, c LoginCallback) (Connection, error) {
	g.calls++
	g.callback = c
	return Connection{ProviderID: GoogleID, AccountID: "synthetic"}, nil
}
func (g *gateway) Revoke(context.Context, Connection) error { return g.revokeErr }
func (g *gateway) FolderURL(context.Context, Connection) (string, error) {
	return "https://example.invalid/folder", g.folderErr
}

func TestLifecyclePolicy(t *testing.T) {
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	g := &gateway{}
	s := NewService(gatewayCatalog{g}, fixedClock{now}, bytes.NewReader(bytes.Repeat([]byte{1}, 1024)))
	previous := Session{Connection: Connection{ProviderID: GoogleID, AccountID: "synthetic"}, IssuedAt: now.Add(-time.Hour)}
	begin, err := s.Begin(context.Background(), BeginInput{ProviderID: GoogleID, Previous: &previous})
	if err != nil {
		t.Fatal(err)
	}
	if begin.Pending == nil || g.request.ForceConsent {
		t.Fatal("valid reconnect must retain consent policy")
	}
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
			_, err := s.Complete(context.Background(), CompleteInput{ProviderID: GoogleID, Pending: &tc.flow, State: tc.state, Code: "synthetic"})
			if !errors.Is(err, journal.ErrAuthRequired) || g.calls != 0 {
				t.Fatal("invalid flow reached exchange")
			}
		})
	}
	current, err := s.Complete(context.Background(), CompleteInput{ProviderID: GoogleID, Pending: begin.Pending, State: begin.Pending.State, Code: "synthetic", Previous: &previous})
	if err != nil {
		t.Fatal(err)
	}
	if !current.IssuedAt.Equal(now) || g.callback.Previous == nil {
		t.Fatal("issuance or previous connection changed")
	}
	previous.IssuedAt = now.Add(-Lifetime)
	_, err = s.Complete(context.Background(), CompleteInput{ProviderID: GoogleID, Pending: begin.Pending, State: begin.Pending.State, Code: "synthetic", Previous: &previous})
	if err != nil {
		t.Fatal(err)
	}
	if g.callback.Previous != nil {
		t.Fatal("expired previous connection forwarded")
	}
}
func TestDisconnectAndSettings(t *testing.T) {
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	failure := errors.New("synthetic failure")
	g := &gateway{revokeErr: failure}
	s := NewService(gatewayCatalog{g}, fixedClock{now}, bytes.NewReader(nil))
	current := Session{Connection: Connection{ProviderID: GoogleID, AccountID: "synthetic", AccountEmail: "synthetic@example.invalid"}, IssuedAt: now}
	if err := s.Disconnect(context.Background(), current); !errors.Is(err, failure) {
		t.Fatal("revocation error lost")
	}
	result, err := s.Settings(context.Background(), current)
	if err != nil || result.ProviderName != "Google" || result.AccountName != current.AccountEmail || result.FolderURL == "" {
		t.Fatal("settings failed")
	}
	g.folderErr = failure
	if _, err := s.Settings(context.Background(), current); !errors.Is(err, failure) {
		t.Fatal("folder error lost")
	}
	current.IssuedAt = now.Add(-Lifetime)
	if err := s.Disconnect(context.Background(), current); !errors.Is(err, journal.ErrAuthRequired) {
		t.Fatal("expired session accepted")
	}
}

type gatewayCatalog struct{ gateway Gateway }

func (c gatewayCatalog) Lookup(id string) (Gateway, bool) {
	return c.gateway, c.gateway != nil && c.gateway.ID() == id
}

type localGateway struct{ Gateway }

func (localGateway) ID() string   { return LocalID }
func (localGateway) Name() string { return "Local" }
func (localGateway) Complete(context.Context, LoginCallback) (Connection, error) {
	return Connection{ProviderID: LocalID, AccountID: "synthetic"}, nil
}

func TestLifecycleUsesGatewayWithoutStorageConstruction(t *testing.T) {
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	g := &gateway{}
	s := NewService(gatewayCatalog{g}, fixedClock{now}, bytes.NewReader(nil))
	if _, err := s.Begin(context.Background(), BeginInput{ProviderID: GoogleID}); !errors.Is(err, ErrLoginFlow) {
		t.Fatal("entropy failure not classified")
	}
	if _, err := s.Begin(context.Background(), BeginInput{ProviderID: "missing"}); !errors.Is(err, ErrProviderUnavailable) {
		t.Fatal("missing provider accepted")
	}
	local := localGateway{g}
	s = NewService(gatewayCatalog{local}, fixedClock{now}, bytes.NewReader(nil))
	result, err := s.Begin(context.Background(), BeginInput{ProviderID: LocalID})
	if err != nil || result.Pending != nil {
		t.Fatal("local login required entropy")
	}
	if _, err := s.Complete(context.Background(), CompleteInput{ProviderID: LocalID}); err != nil {
		t.Fatal(err)
	}
	current := Session{Connection: Connection{ProviderID: LocalID, AccountID: "synthetic"}, IssuedAt: now}
	settings, err := s.Settings(context.Background(), current)
	if err != nil || settings.FolderURL != "" {
		t.Fatal("optional folder locator was required")
	}
}

func TestReconnectConsentAndPreviousProviderEligibility(t *testing.T) {
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name               string
		previous           *Session
		force, wantConsent bool
	}{
		{name: "new", wantConsent: true},
		{name: "different provider", previous: &Session{Connection: Connection{ProviderID: LocalID, AccountID: "synthetic"}, IssuedAt: now}, wantConsent: true},
		{name: "expired", previous: &Session{Connection: Connection{ProviderID: GoogleID, AccountID: "synthetic"}, IssuedAt: now.Add(-Lifetime)}, wantConsent: true},
		{name: "forced", previous: &Session{Connection: Connection{ProviderID: GoogleID, AccountID: "synthetic"}, IssuedAt: now}, force: true, wantConsent: true},
		{name: "same provider", previous: &Session{Connection: Connection{ProviderID: GoogleID, AccountID: "synthetic"}, IssuedAt: now}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := &gateway{}
			s := NewService(gatewayCatalog{g}, fixedClock{now}, bytes.NewReader(bytes.Repeat([]byte{1}, 96)))
			begin, err := s.Begin(context.Background(), BeginInput{ProviderID: GoogleID, Previous: tc.previous, ForceConsent: tc.force})
			if err != nil {
				t.Fatal(err)
			}
			if g.request.ForceConsent != tc.wantConsent {
				t.Fatal("consent policy changed")
			}
			_, err = s.Complete(context.Background(), CompleteInput{ProviderID: GoogleID, Pending: begin.Pending, Code: "synthetic", State: begin.Pending.State, Previous: tc.previous})
			if err != nil {
				t.Fatal(err)
			}
			eligible := tc.previous != nil && tc.previous.ProviderID == GoogleID && tc.previous.ValidAt(now)
			if (g.callback.Previous != nil) != eligible {
				t.Fatal("previous connection eligibility changed")
			}
		})
	}
}
