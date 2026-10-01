package handler

import (
	"context"
	"errors"
	"net/http"

	"github/mridha/myjournal-app/internal/config"
	"github/mridha/myjournal-app/internal/i18n"
	"github/mridha/myjournal-app/internal/journal"
	"github/mridha/myjournal-app/internal/provider"
	"github/mridha/myjournal-app/internal/session"
	"github/mridha/myjournal-app/internal/web/api"
)

type ServiceFactory func(context.Context, session.Session) (*journal.Service, error)

var ErrUnauthenticated = errors.New("authentication required")

// Server implements the generated OpenAPI chi server interface.
type Server struct {
	cfg            config.Config
	registry       *provider.Registry
	serviceFactory ServiceFactory
	messages       i18n.Catalog
}

func New(cfg config.Config, registry *provider.Registry, service ServiceFactory, messages i18n.Catalog) *Server {
	return &Server{cfg: cfg, registry: registry, serviceFactory: service, messages: messages}
}

func (s *Server) message(key string) string { return s.messages.Text(key) }

func (s *Server) currentSession(r *http.Request) (session.Session, error) {
	cookie, err := r.Cookie(session.CookieName)
	if err != nil {
		return session.Session{}, ErrUnauthenticated
	}
	current, err := session.Decode(s.cfg.SessionKey, cookie.Value)
	if err != nil || current.ProviderID == "" || current.AccountID == "" || current.IssuedAt.IsZero() {
		return session.Session{}, ErrUnauthenticated
	}
	if _, ok := s.registry.Get(current.ProviderID); !ok {
		return session.Session{}, ErrUnauthenticated
	}
	return current, nil
}

func (s *Server) service(r *http.Request) (*journal.Service, error) {
	current, err := s.currentSession(r)
	if err != nil {
		return nil, err
	}
	return s.serviceFactory(r.Context(), current)
}

func (s *Server) writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrUnauthenticated) {
		http.Error(w, s.message("errors.unauthorized"), http.StatusUnauthorized)
		return
	}
	http.Error(w, s.message("errors.internal"), http.StatusInternalServerError)
}

var _ api.ServerInterface = (*Server)(nil)
