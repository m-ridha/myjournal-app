package handler

import (
	"crypto/rand"
	"time"

	"github/mridha/myjournal-app/internal/config"
	provider "github/mridha/myjournal-app/internal/core/connection"
	"github/mridha/myjournal-app/internal/core/journal"
	"github/mridha/myjournal-app/internal/driven/session"
	"github/mridha/myjournal-app/internal/driver/web/i18n"
)

type testClock func() time.Time

func (c testClock) Now() time.Time { return c() }
func newTestServer(cfg config.Config, registry *provider.Registry, factory ServiceFactory, messages i18n.Catalog, clk journal.Clock) *Server {
	return New(Options{Codec: session.NewCodec(cfg.SessionKey), CookieSecure: cfg.CookieSecure}, registry, factory, messages, clk, provider.NewService(registry, clk, rand.Reader))
}
