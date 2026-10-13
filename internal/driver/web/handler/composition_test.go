package handler

import (
	"crypto/rand"
	"time"

	"github/m-ridha/myjournal-app/internal/config"
	provider "github/m-ridha/myjournal-app/internal/core/connection"
	"github/m-ridha/myjournal-app/internal/core/journal"
	"github/m-ridha/myjournal-app/internal/driven/session"
	"github/m-ridha/myjournal-app/internal/driver/web/i18n"
)

type testClock func() time.Time

func (c testClock) Now() time.Time { return c() }
func newTestServer(cfg config.Config, registry *provider.Registry, factory ServiceFactory, messages i18n.Catalog, clk journal.Clock) *Server {
	return New(Options{Codec: session.NewCodec(cfg.SessionKey), CookieSecure: cfg.CookieSecure}, registry, factory, messages, clk, provider.NewService(registry, clk, rand.Reader))
}
