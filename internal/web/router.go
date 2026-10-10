package web

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"cloud.google.com/go/civil"
	"github/mridha/myjournal-app/assets"
	"github/mridha/myjournal-app/internal/config"
	"github/mridha/myjournal-app/internal/journal"
	"github/mridha/myjournal-app/internal/provider"
	"github/mridha/myjournal-app/internal/session"
	"github/mridha/myjournal-app/internal/web/api"
	"github/mridha/myjournal-app/internal/web/handler"
	webMiddleware "github/mridha/myjournal-app/internal/web/middleware"
)

func NewRouter(cfg config.Config, registry *provider.Registry, clk journal.Clock, serviceFactory handler.ServiceFactory) http.Handler {

	r := chi.NewRouter()
	r.Use(webMiddleware.SecurityHeaders)
	r.Use(middleware.RequestID)
	r.Use(webMiddleware.RequestLog)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(webMiddleware.TimeZone)
	protection := http.NewCrossOriginProtection()
	protection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.WriteError(w, http.StatusForbidden, localeCatalog.Text("errors.crossOrigin"))
	}))
	r.Use(protection.Handler)

	if staticFS, err := fs.Sub(assets.FS, "static"); err == nil {
		r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	}

	// API routes
	apiHandler := handler.New(cfg, registry, serviceFactory, localeCatalog, clk)
	api.HandlerWithOptions(apiHandler, api.ChiServerOptions{
		BaseRouter: r,
		ErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, _ error) {
			api.WriteError(w, http.StatusBadRequest, localeCatalog.Text("errors.invalidRequest"))
		},
	})

	// Web routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if current, err := currentSession(r, cfg.SessionKey, registry, clk.Now()); err == nil {
			renderEntryPage(w, clk.Now().In(journal.TimeZone(r.Context())), clk, current, journal.TimeZone(r.Context()))
			return
		}
		_, googleLogin := registry.Get("google")
		_, localLogin := registry.Get("local")
		renderPage(w, pageData{TitleKey: "landing.title", DescriptionKey: "landing.description", SignedIn: false, Page: "landing", GoogleLogin: googleLogin, LocalLogin: localLogin})
	})
	r.Get("/entries/today", func(w http.ResponseWriter, r *http.Request) {
		current, err := currentSession(r, cfg.SessionKey, registry, clk.Now())
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		renderEntryPage(w, clk.Now().In(journal.TimeZone(r.Context())), clk, current, journal.TimeZone(r.Context()))
	})
	r.Get("/entries/{date}", func(w http.ResponseWriter, r *http.Request) {
		current, err := currentSession(r, cfg.SessionKey, registry, clk.Now())
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		dateStr := chi.URLParam(r, "date")
		date, err := time.Parse(time.DateOnly, dateStr)
		if err != nil {
			renderNotFoundPage(w)
			return
		}
		if civil.DateOf(date).After(civil.DateOf(clk.Now().In(journal.TimeZone(r.Context())))) {
			renderNotFoundPage(w)
			return
		}
		renderEntryPage(w, date, clk, current, journal.TimeZone(r.Context()))
	})
	r.Get("/entries", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r, cfg.SessionKey, registry, clk.Now()) {
			return
		}
		renderPage(w, pageData{TitleKey: "entries.title", SignedIn: true, Page: "entries"})
	})
	r.Get("/calendar", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r, cfg.SessionKey, registry, clk.Now()) {
			return
		}
		renderPage(w, pageData{TitleKey: "calendar.title", SignedIn: true, Page: "calendar", Today: civil.DateOf(clk.Now().In(journal.TimeZone(r.Context()))).String()})
	})
	r.Get("/settings", func(w http.ResponseWriter, r *http.Request) {
		current, err := currentSession(r, cfg.SessionKey, registry, clk.Now())
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		folderURL := ""
		if p, ok := registry.Get(current.ProviderID); ok {
			if locator, ok := p.(provider.FolderLocator); ok {
				folderURL, _ = locator.FolderURL(r.Context(), current.Connection)
			}
		}
		renderPage(w, pageData{TitleKey: "settings.title", SignedIn: true, Page: "settings", AccountName: current.AccountEmail, DraftScope: draftScope(current), FolderName: localeCatalog.Text("app.name"), FolderURL: folderURL})
	})
	r.Get("/privacy", func(w http.ResponseWriter, r *http.Request) {
		renderPage(w, pageData{TitleKey: "privacy.title", DescriptionKey: "privacy.description", Page: "privacy"})
	})
	r.Get("/help", func(w http.ResponseWriter, r *http.Request) {
		renderPage(w, pageData{TitleKey: "help.title", DescriptionKey: "help.description", Page: "help"})
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
			api.WriteError(w, http.StatusMethodNotAllowed, localeCatalog.Text("errors.invalidRequest"))
			return
		}
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1/") {
			api.WriteError(w, http.StatusNotFound, localeCatalog.Text("errors.invalidRequest"))
			return
		}
		renderNotFoundPage(w)
	})

	return r
}

func renderNotFoundPage(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	renderPage(w, pageData{TitleKey: "notFound.title", DescriptionKey: "notFound.description", Page: "notfound"})
}

func renderEntryPage(w http.ResponseWriter, date time.Time, clk journal.Clock, current session.Session, location *time.Location) {
	today := clk.Now().In(location)
	renderPage(w, pageData{
		Title:          localeCatalog.FormatDate(date),
		DraftScope:     draftScope(current),
		DescriptionKey: "editor.description",
		SignedIn:       true,
		Page:           "editor",
		Date:           date.Format(time.DateOnly),
		Today:          civil.DateOf(today).String(),
		IsToday:        date.Year() == today.Year() && date.Month() == today.Month() && date.Day() == today.Day(),
	})
}

func currentSession(r *http.Request, secret string, registry *provider.Registry, now time.Time) (session.Session, error) {
	cookie, err := r.Cookie(session.CookieName)
	if err != nil {
		return session.Session{}, err
	}
	current, err := session.Decode(secret, cookie.Value)
	if err != nil || !current.ValidAt(now) {
		return session.Session{}, errors.New("invalid session")
	}
	if _, ok := registry.Get(current.ProviderID); !ok {
		return session.Session{}, errors.New("session provider is unavailable")
	}
	return current, nil
}

func requireSession(w http.ResponseWriter, r *http.Request, secret string, registry *provider.Registry, now time.Time) bool {
	if _, err := currentSession(r, secret, registry, now); err == nil {
		return true
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return false
}

// Scope browser drafts by provider/account without exposing account labels or credentials.
func draftScope(current session.Session) string {
	identity, _ := json.Marshal([2]string{current.ProviderID, current.AccountID})
	sum := sha256.Sum256(identity)
	return hex.EncodeToString(sum[:])
}
