package web

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github/mridha/myjournal-app/assets"
	provider "github/mridha/myjournal-app/internal/core/connection"
	"github/mridha/myjournal-app/internal/core/journal"
	"github/mridha/myjournal-app/internal/driver/web/api"
	"github/mridha/myjournal-app/internal/driver/web/authsession"
	"github/mridha/myjournal-app/internal/driver/web/handler"
	webMiddleware "github/mridha/myjournal-app/internal/driver/web/middleware"
)

func NewRouter(cfg Options, registry *provider.Registry, clk journal.Clock, serviceFactory handler.ServiceFactory, renderer *Renderer, connections *provider.Service) http.Handler {

	r := chi.NewRouter()
	r.Use(webMiddleware.SecurityHeaders)
	// Trust startup configuration rather than client-supplied proxy headers.
	if cfg.HTTPS {
		r.Use(webMiddleware.StrictTransportSecurity)
	}
	r.Use(middleware.RequestID)
	r.Use(webMiddleware.RequestLog)
	r.Use(webMiddleware.Recoverer(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1/") {
			api.WriteError(w, http.StatusInternalServerError, renderer.catalog.Text("errors.internal"))
			return
		}
		http.Error(w, renderer.catalog.Text("errors.internal"), http.StatusInternalServerError)
	}))
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(webMiddleware.TimeZone)
	protection := http.NewCrossOriginProtection()
	protection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.WriteError(w, http.StatusForbidden, renderer.catalog.Text("errors.crossOrigin"))
	}))
	r.Use(protection.Handler)

	if staticFS, err := fs.Sub(assets.FS, "static"); err == nil {
		r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	}

	// API routes
	apiHandler := handler.New(cfg.Session, registry, serviceFactory, renderer.catalog, clk, connections)
	api.HandlerWithOptions(apiHandler, api.ChiServerOptions{
		BaseRouter: r,
		ErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, _ error) {
			api.WriteError(w, http.StatusBadRequest, renderer.catalog.Text("errors.invalidRequest"))
		},
	})

	// Web routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if current, err := currentSession(r, cfg.Session.SessionKey, registry, clk.Now()); err == nil {
			renderer.renderEntryPage(w, clk.Now().In(journal.TimeZone(r.Context())), clk, current, journal.TimeZone(r.Context()))
			return
		}
		_, googleLogin := registry.Get(provider.GoogleID)
		_, localLogin := registry.Get(provider.LocalID)
		renderer.renderPage(w, pageData{TitleKey: "landing.title", DescriptionKey: "landing.description", SignedIn: false, Page: "landing", GoogleLogin: googleLogin, LocalLogin: localLogin})
	})
	r.Get("/entries/today", func(w http.ResponseWriter, r *http.Request) {
		current, err := currentSession(r, cfg.Session.SessionKey, registry, clk.Now())
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		renderer.renderEntryPage(w, clk.Now().In(journal.TimeZone(r.Context())), clk, current, journal.TimeZone(r.Context()))
	})
	r.Get("/entries/{date}", func(w http.ResponseWriter, r *http.Request) {
		current, err := currentSession(r, cfg.Session.SessionKey, registry, clk.Now())
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		dateStr := chi.URLParam(r, "date")
		date, err := time.Parse(time.DateOnly, dateStr)
		if err != nil {
			renderer.renderNotFoundPage(w)
			return
		}
		if journal.ValidateEntryDate(civil.DateOf(date), civil.DateOf(clk.Now().In(journal.TimeZone(r.Context())))) != nil {
			renderer.renderNotFoundPage(w)
			return
		}
		renderer.renderEntryPage(w, date, clk, current, journal.TimeZone(r.Context()))
	})
	r.Get("/entries", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r, cfg.Session.SessionKey, registry, clk.Now()) {
			return
		}
		renderer.renderPage(w, pageData{TitleKey: "entries.title", SignedIn: true, Page: "entries"})
	})
	r.Get("/calendar", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r, cfg.Session.SessionKey, registry, clk.Now()) {
			return
		}
		renderer.renderPage(w, pageData{TitleKey: "calendar.title", SignedIn: true, Page: "calendar", Today: civil.DateOf(clk.Now().In(journal.TimeZone(r.Context()))).String()})
	})
	r.Get("/settings", func(w http.ResponseWriter, r *http.Request) {
		current, err := currentSession(r, cfg.Session.SessionKey, registry, clk.Now())
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		settings, folderErr := connections.Settings(r.Context(), current)
		folderURL := settings.FolderURL
		renderer.renderPage(w, pageData{TitleKey: "settings.title", SignedIn: true, Page: "settings", AccountName: current.AccountEmail, DraftScope: draftScope(current), FolderName: renderer.catalog.Text("app.name"), FolderURL: folderURL, FolderUnavailable: folderErr != nil})
	})
	r.Get("/privacy", func(w http.ResponseWriter, r *http.Request) {
		_, err := currentSession(r, cfg.Session.SessionKey, registry, clk.Now())
		renderer.renderPage(w, pageData{TitleKey: "privacy.title", DescriptionKey: "privacy.description", SignedIn: err == nil, Page: "privacy"})
	})
	r.Get("/help", func(w http.ResponseWriter, r *http.Request) {
		_, err := currentSession(r, cfg.Session.SessionKey, registry, clk.Now())
		renderer.renderPage(w, pageData{TitleKey: "help.title", DescriptionKey: "help.description", SignedIn: err == nil, Page: "help"})
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) {
		// Match against the registered routes, using separate contexts so probing
		// methods cannot modify the rejected request's routing parameters.
		for _, method := range []string{http.MethodConnect, http.MethodDelete, http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPatch, http.MethodPost, http.MethodPut, http.MethodTrace} {
			if r.Match(chi.NewRouteContext(), method, req.URL.Path) {
				w.Header().Add("Allow", method)
			}
		}

		if strings.HasPrefix(req.URL.Path, "/api/v1/") {
			api.WriteError(w, http.StatusMethodNotAllowed, renderer.catalog.Text("errors.invalidRequest"))
			return
		}
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1/") {
			api.WriteError(w, http.StatusNotFound, renderer.catalog.Text("errors.invalidRequest"))
			return
		}
		renderer.renderNotFoundPage(w)
	})

	return r
}

func (renderer *Renderer) renderNotFoundPage(w http.ResponseWriter) {
	renderer.renderPageStatus(w, http.StatusNotFound, pageData{TitleKey: "notFound.title", DescriptionKey: "notFound.description", Page: "notfound"})
}

func (renderer *Renderer) renderEntryPage(w http.ResponseWriter, date time.Time, clk journal.Clock, current provider.Session, location *time.Location) {
	today := clk.Now().In(location)
	renderer.renderPage(w, pageData{
		Title:          renderer.catalog.FormatDate(date),
		DraftScope:     draftScope(current),
		DescriptionKey: "editor.description",
		SignedIn:       true,
		Page:           "editor",
		Date:           date.Format(time.DateOnly),
		Today:          civil.DateOf(today).String(),
		IsToday:        date.Year() == today.Year() && date.Month() == today.Month() && date.Day() == today.Day(),
	})
}

func currentSession(r *http.Request, secret string, registry *provider.Registry, now time.Time) (provider.Session, error) {
	return authsession.Read(r, secret, registry, now)
}

func requireSession(w http.ResponseWriter, r *http.Request, secret string, registry *provider.Registry, now time.Time) bool {
	if _, err := currentSession(r, secret, registry, now); err == nil {
		return true
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return false
}

// Scope browser drafts by provider/account without exposing account labels or credentials.
func draftScope(current provider.Session) string {
	identity, _ := json.Marshal([2]string{current.ProviderID, current.AccountID})
	sum := sha256.Sum256(identity)
	return hex.EncodeToString(sum[:])
}
