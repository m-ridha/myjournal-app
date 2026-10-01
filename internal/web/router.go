package web

import (
	"errors"
	"io/fs"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github/mridha/myjournal-app/assets"
	"github/mridha/myjournal-app/internal/config"
	"github/mridha/myjournal-app/internal/journal"
	"github/mridha/myjournal-app/internal/provider"
	"github/mridha/myjournal-app/internal/session"
	"github/mridha/myjournal-app/internal/web/api"
	"github/mridha/myjournal-app/internal/web/handler"
)

func NewRouter(cfg config.Config, registry *provider.Registry, clk journal.Clock, serviceFactory handler.ServiceFactory) http.Handler {

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	if staticFS, err := fs.Sub(assets.FS, "static"); err == nil {
		r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	}

	// API routes
	apiHandler := handler.New(cfg, registry, serviceFactory, localeCatalog)
	api.HandlerWithOptions(apiHandler, api.ChiServerOptions{
		BaseRouter: r,
		ErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, localeCatalog.Text("errors.invalidRequest"), http.StatusBadRequest)
		},
	})

	// Web routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := currentSession(r, cfg.SessionKey, registry); err == nil {
			renderEntryPage(w, clk.Now(), clk)
			return
		}
		renderPage(w, pageData{TitleKey: "landing.title", DescriptionKey: "landing.description", SignedIn: false, Page: "landing"})
	})
	r.Get("/entries/today", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r, cfg.SessionKey, registry) {
			return
		}
		renderEntryPage(w, clk.Now(), clk)
	})
	r.Get("/entries/{date}", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r, cfg.SessionKey, registry) {
			return
		}
		dateStr := chi.URLParam(r, "date")
		date, err := time.Parse(time.DateOnly, dateStr)
		if err != nil {
			renderNotFoundPage(w)
			return
		}
		if date.After(clk.Now()) {
			renderNotFoundPage(w)
			return
		}
		renderEntryPage(w, date, clk)
	})
	r.Get("/entries", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r, cfg.SessionKey, registry) {
			return
		}
		renderPage(w, pageData{TitleKey: "entries.title", SignedIn: true, Page: "entries"})
	})
	r.Get("/calendar", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r, cfg.SessionKey, registry) {
			return
		}
		renderPage(w, pageData{TitleKey: "calendar.title", SignedIn: true, Page: "calendar"})
	})
	r.Get("/settings", func(w http.ResponseWriter, r *http.Request) {
		current, err := currentSession(r, cfg.SessionKey, registry)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		renderPage(w, pageData{TitleKey: "settings.title", SignedIn: true, Page: "settings", AccountName: current.AccountEmail, FolderName: localeCatalog.Text("app.name")})
	})
	r.Get("/privacy", func(w http.ResponseWriter, r *http.Request) {
		renderPage(w, pageData{TitleKey: "privacy.title", DescriptionKey: "privacy.description", Page: "privacy"})
	})
	r.Get("/help", func(w http.ResponseWriter, r *http.Request) {
		renderPage(w, pageData{TitleKey: "help.title", DescriptionKey: "help.description", Page: "help"})
	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		renderNotFoundPage(w)
	})

	return r
}

func renderNotFoundPage(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	renderPage(w, pageData{TitleKey: "notFound.title", DescriptionKey: "notFound.description", Page: "notfound"})
}

func renderEntryPage(w http.ResponseWriter, date time.Time, clk journal.Clock) {
	today := clk.Now()
	renderPage(w, pageData{
		Title:          localeCatalog.FormatDate(date),
		DescriptionKey: "editor.description",
		SignedIn:       true,
		Page:           "editor",
		Date:           date.Format(time.DateOnly),
		IsToday:        date.Year() == today.Year() && date.Month() == today.Month() && date.Day() == today.Day(),
	})
}

func currentSession(r *http.Request, secret string, registry *provider.Registry) (session.Session, error) {
	cookie, err := r.Cookie(session.CookieName)
	if err != nil {
		return session.Session{}, err
	}
	current, err := session.Decode(secret, cookie.Value)
	if err != nil || current.ProviderID == "" || current.AccountID == "" || current.IssuedAt.IsZero() {
		return session.Session{}, errors.New("invalid session")
	}
	if _, ok := registry.Get(current.ProviderID); !ok {
		return session.Session{}, errors.New("session provider is unavailable")
	}
	return current, nil
}

func requireSession(w http.ResponseWriter, r *http.Request, secret string, registry *provider.Registry) bool {
	if _, err := currentSession(r, secret, registry); err == nil {
		return true
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return false
}
