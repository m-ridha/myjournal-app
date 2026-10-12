package web

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	provider "github/mridha/myjournal-app/internal/core/connection"
	"github/mridha/myjournal-app/internal/core/journal"
)

const languageCookieName = "mj_language"
const languagePreferenceSeconds = 365 * 24 * 60 * 60

func registerLanguageRoute(r chi.Router, cfg Options, registry *provider.Registry, clk journal.Clock, renderer *Renderer) {
	r.Post("/settings/language", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r, cfg.Session.SessionKey, registry, clk.Now()) {
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1024)
		if err := r.ParseForm(); err != nil {
			http.Error(w, renderer.catalog.Text("errors.invalidRequest"), http.StatusBadRequest)
			return
		}
		values := r.PostForm["language"]
		if len(values) != 1 || (values[0] != "en" && values[0] != "id") {
			http.Error(w, renderer.catalog.Text("errors.invalidRequest"), http.StatusBadRequest)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: languageCookieName, Value: values[0], Path: "/", MaxAge: languagePreferenceSeconds, HttpOnly: true, Secure: cfg.HTTPS, SameSite: http.SameSiteLaxMode})
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
	})
}
