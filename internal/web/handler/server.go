package handler

import (
	"context"

	"github/mridha/myjournal-app/internal/config"
	"github/mridha/myjournal-app/internal/i18n"
	"github/mridha/myjournal-app/internal/journal"
	"github/mridha/myjournal-app/internal/provider"
	"github/mridha/myjournal-app/internal/web/api"
)

type ServiceFactory func(context.Context) (*journal.Service, error)

// Server implements the generated OpenAPI chi server interface.
type Server struct {
	cfg      config.Config
	registry *provider.Registry
	service  ServiceFactory
	messages i18n.Catalog
}

func New(cfg config.Config, registry *provider.Registry, service ServiceFactory, messages i18n.Catalog) *Server {
	return &Server{cfg: cfg, registry: registry, service: service, messages: messages}
}

func (s *Server) message(key string) string { return s.messages.Text(key) }

var _ api.ServerInterface = (*Server)(nil)
