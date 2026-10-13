package web

import (
	"crypto/rand"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github/mridha/myjournal-app/internal/config"
	provider "github/mridha/myjournal-app/internal/core/connection"
	"github/mridha/myjournal-app/internal/core/journal"
	"github/mridha/myjournal-app/internal/driven/session"
	"github/mridha/myjournal-app/internal/driver/web/handler"
)

func newTestRouter(cfg config.Config, registry *provider.Registry, clk journal.Clock, factory handler.ServiceFactory) http.Handler {
	renderer, err := NewRenderer()
	if err != nil {
		panic(err)
	}
	origin, _ := url.Parse(cfg.BaseURL)
	return NewRouter(Options{BaseURL: cfg.BaseURL, Session: handler.Options{Codec: session.NewCodec(cfg.SessionKey), CookieSecure: cfg.CookieSecure}, HTTPS: origin != nil && origin.Scheme == "https"}, registry, clk, factory, renderer, provider.NewService(registry, clk, rand.Reader))
}
func testRenderer(t *testing.T) *Renderer {
	t.Helper()
	r, err := NewRenderer()
	require.NoError(t, err)
	return r
}
