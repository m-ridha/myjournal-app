package web

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
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
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(webMiddleware.TimeZone)

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
		if current, err := currentSession(r, cfg.SessionKey, registry); err == nil {
			renderEntryPage(w, clk.Now().In(journal.TimeZone(r.Context())), clk, current, journal.TimeZone(r.Context()))
			return
		}
		renderPage(w, pageData{TitleKey: "landing.title", DescriptionKey: "landing.description", SignedIn: false, Page: "landing"})
	})
	r.Get("/entries/today", func(w http.ResponseWriter, r *http.Request) {
		current, err := currentSession(r, cfg.SessionKey, registry)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		renderEntryPage(w, clk.Now().In(journal.TimeZone(r.Context())), clk, current, journal.TimeZone(r.Context()))
	})
	r.Get("/entries/{date}", func(w http.ResponseWriter, r *http.Request) {
		current, err := currentSession(r, cfg.SessionKey, registry)
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
		if !requireSession(w, r, cfg.SessionKey, registry) {
			return
		}
		renderPage(w, pageData{TitleKey: "entries.title", SignedIn: true, Page: "entries"})
	})
	r.Get("/calendar", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r, cfg.SessionKey, registry) {
			return
		}
		renderPage(w, pageData{TitleKey: "calendar.title", SignedIn: true, Page: "calendar", Today: civil.DateOf(clk.Now().In(journal.TimeZone(r.Context()))).String()})
	})
	r.Get("/settings", func(w http.ResponseWriter, r *http.Request) {
		current, err := currentSession(r, cfg.SessionKey, registry)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		renderPage(w, pageData{TitleKey: "settings.title", SignedIn: true, Page: "settings", AccountName: current.AccountEmail, DraftScope: draftScope(current), FolderName: localeCatalog.Text("app.name")})
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

// Scope browser drafts by provider/account without exposing account labels or credentials.
func draftScope(current session.Session) string {
	identity, _ := json.Marshal([2]string{current.ProviderID, current.AccountID})
	sum := sha256.Sum256(identity)
	return hex.EncodeToString(sum[:])
}
