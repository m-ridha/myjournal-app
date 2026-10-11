package connection

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"

	"github/mridha/myjournal-app/internal/core/journal"
)

var ErrProviderUnavailable = errors.New("provider is not configured")
var ErrLoginFlow = errors.New("cannot create login transaction")

// Service owns transport-independent connection workflows.
type Service struct {
	registry Catalog
	clock    journal.Clock
	entropy  io.Reader
}

func NewService(registry Catalog, clock journal.Clock, entropy io.Reader) *Service {
	return &Service{registry: registry, clock: clock, entropy: entropy}
}

type BeginInput struct {
	ProviderID   string
	ForceConsent bool
	Previous     *Session
}
type BeginResult struct {
	URL     string
	Pending *LoginFlow
}
type CompleteInput struct {
	ProviderID, Code, State string
	Pending                 *LoginFlow
	Previous                *Session
}

func (s *Service) previous(id string, current *Session) *Connection {
	if current == nil || !current.ValidAt(s.clock.Now()) || current.ProviderID != id {
		return nil
	}
	value := current.Connection
	return &value
}
func (s *Service) Begin(ctx context.Context, input BeginInput) (BeginResult, error) {
	p, ok := s.registry.Lookup(input.ProviderID)
	if !ok {
		return BeginResult{}, ErrProviderUnavailable
	}
	request := LoginRequest{}
	var pending *LoginFlow
	if p.ID() != LocalID {
		flow, err := NewLoginFlow(p.ID(), s.clock.Now(), s.entropy)
		if err != nil {
			return BeginResult{}, fmt.Errorf("%w: %w", ErrLoginFlow, err)
		}
		pending = &flow
		request = LoginRequest{State: flow.State, Verifier: flow.Verifier, Nonce: flow.Nonce, ForceConsent: input.ForceConsent || s.previous(p.ID(), input.Previous) == nil}
	}
	target, err := p.Begin(ctx, request)
	if err != nil {
		return BeginResult{}, err
	}
	return BeginResult{URL: target, Pending: pending}, nil
}
func (s *Service) Complete(ctx context.Context, input CompleteInput) (Session, error) {
	p, ok := s.registry.Lookup(input.ProviderID)
	if !ok {
		return Session{}, ErrProviderUnavailable
	}
	callback := LoginCallback{}
	if p.ID() != LocalID {
		flow := input.Pending
		if flow == nil || !flow.ValidAt(s.clock.Now()) || flow.ProviderID != p.ID() || input.Code == "" || subtle.ConstantTimeCompare([]byte(input.State), []byte(flow.State)) != 1 {
			return Session{}, &LoginError{Stage: LoginTransaction, Err: journal.ErrAuthRequired}
		}
		callback = LoginCallback{Code: input.Code, Verifier: flow.Verifier, Nonce: flow.Nonce, Previous: s.previous(p.ID(), input.Previous)}
	}
	current, err := p.Complete(ctx, callback)
	if err != nil {
		return Session{}, err
	}
	return Session{Connection: current, IssuedAt: s.clock.Now().UTC()}, nil
}
func (s *Service) active(current Session) (Gateway, error) {
	if !current.ValidAt(s.clock.Now()) {
		return nil, journal.ErrAuthRequired
	}
	p, ok := s.registry.Lookup(current.ProviderID)
	if !ok {
		return nil, journal.ErrAuthRequired
	}
	return p, nil
}
func (s *Service) Disconnect(ctx context.Context, current Session) error {
	p, err := s.active(current)
	if err != nil {
		return err
	}
	return p.Revoke(ctx, current.Connection)
}
