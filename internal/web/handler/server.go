package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github/mridha/myjournal-app/internal/config"
	"github/mridha/myjournal-app/internal/i18n"
	"github/mridha/myjournal-app/internal/journal"
	"github/mridha/myjournal-app/internal/provider"
	"github/mridha/myjournal-app/internal/session"
	"github/mridha/myjournal-app/internal/web/api"
)

type ServiceFactory func(context.Context, session.Session) (*journal.Service, error)

var ErrUnauthenticated = errors.New("authentication required")

var _ api.ServerInterface = (*Server)(nil)

// Server implements the generated OpenAPI chi server interface.
type Server struct {
	cfg            config.Config
	registry       *provider.Registry
	serviceFactory ServiceFactory
	messages       i18n.Catalog
	now            func() time.Time
}

func New(cfg config.Config, registry *provider.Registry, service ServiceFactory, messages i18n.Catalog, clk journal.Clock) *Server {
	return &Server{cfg: cfg, registry: registry, serviceFactory: service, messages: messages, now: clk.Now}
}

func (s *Server) message(key string) string { return s.messages.Text(key) }

func (s *Server) currentSession(r *http.Request) (session.Session, error) {
	cookie, err := r.Cookie(session.CookieName)
	if err != nil {
		return session.Session{}, ErrUnauthenticated
	}
	current, err := session.Decode(s.cfg.SessionKey, cookie.Value)
	if err != nil || !current.ValidAt(s.now()) {
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
	status, key := http.StatusInternalServerError, "errors.internal"
	switch {
	case errors.Is(err, ErrUnauthenticated), errors.Is(err, journal.ErrAuthRequired):
		status, key = http.StatusUnauthorized, "errors.unauthorized"
	case errors.Is(err, journal.ErrUnavailable):
		status, key = http.StatusServiceUnavailable, "errors.entryUnavailable"
	case errors.Is(err, journal.ErrNotFound):
		status, key = http.StatusNotFound, "errors.entryUnavailable"
	case errors.Is(err, journal.ErrConflict):
		status, key = http.StatusConflict, "errors.conflict"
	case errors.Is(err, journal.ErrFutureDate):
		status, key = http.StatusBadRequest, "errors.futureDate"
	case errors.Is(err, journal.ErrInvalidText):
		status, key = http.StatusBadRequest, "errors.invalidText"
	case errors.Is(err, journal.ErrInvalidPagination):
		status, key = http.StatusBadRequest, "errors.pagination"
	}
	api.WriteError(w, status, s.message(key))
}
