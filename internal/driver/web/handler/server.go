package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	provider "github/mridha/myjournal-app/internal/core/connection"
	"github/mridha/myjournal-app/internal/core/journal"
	"github/mridha/myjournal-app/internal/driver/web/api"
	"github/mridha/myjournal-app/internal/driver/web/authsession"
	"github/mridha/myjournal-app/internal/driver/web/i18n"
)

// Options contains only browser-session transport configuration.
type Options struct {
	SessionKey   string
	CookieSecure bool
}

// JournalService is the journal use-case contract consumed by HTTP handlers.
// Constructors in core continue to return their concrete implementation.
type JournalService interface {
	ListEntries(ctx context.Context, page, size int) (journal.ListPage, error)
	Open(ctx context.Context, d journal.Date) (journal.Entry, error)
	Save(ctx context.Context, d journal.Date, text string, base journal.Version) (journal.Entry, error)
	Delete(ctx context.Context, d journal.Date) error
	Month(ctx context.Context, year int, month time.Month) (journal.MonthView, error)
}

// ServiceFactory resolves a journal service for the validated browser session.
type ServiceFactory func(context.Context, provider.Session) (JournalService, error)

var ErrUnauthenticated = errors.New("authentication required")

// Server implements the generated OpenAPI chi server interface.
type Server struct {
	cfg            Options
	connections    *provider.Service
	registry       *provider.Registry
	serviceFactory ServiceFactory
	messages       i18n.Catalog
	now            func() time.Time
}

var _ api.ServerInterface = (*Server)(nil)

func New(cfg Options, registry *provider.Registry, service ServiceFactory, messages i18n.Catalog, clk journal.Clock, connections *provider.Service) *Server {
	return &Server{connections: connections, cfg: cfg, registry: registry, serviceFactory: service, messages: messages, now: clk.Now}
}

func (s *Server) message(key string) string { return s.messages.Text(key) }

func (s *Server) currentSession(r *http.Request) (provider.Session, error) {
	return authsession.Read(r, s.cfg.SessionKey, s.registry, s.now())
}

func (s *Server) service(r *http.Request) (JournalService, error) {
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
	case errors.Is(err, journal.ErrUnavailable), errors.Is(err, context.DeadlineExceeded):
		status, key = http.StatusServiceUnavailable, "errors.entryUnavailable"
	case errors.Is(err, journal.ErrNotFound):
		status, key = http.StatusNotFound, "errors.entryUnavailable"
	case errors.Is(err, journal.ErrConflict):
		status, key = http.StatusConflict, "errors.conflict"
	case errors.Is(err, journal.ErrFutureDate):
		status, key = http.StatusBadRequest, "errors.futureDate"
	case errors.Is(err, journal.ErrInvalidDate):
		status, key = http.StatusBadRequest, "errors.invalidDate"
	case errors.Is(err, journal.ErrInvalidText):
		status, key = http.StatusBadRequest, "errors.invalidText"
	case errors.Is(err, journal.ErrInvalidPagination):
		status, key = http.StatusBadRequest, "errors.pagination"
	}
	api.WriteError(w, status, s.message(key))
}
