package web

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const languageCookieName = "mj_language"
const languagePreferenceSeconds = 365 * 24 * 60 * 60

func registerLanguageRoute(r chi.Router, cfg Options, renderer *Renderer) {
	r.Post("/settings/language", func(w http.ResponseWriter, r *http.Request) {
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
		http.Redirect(w, r, languageReturnPath(r.PostFormValue("returnTo")), http.StatusSeeOther)
	})
}

// Only return to browser pages, never an external origin or auth/API endpoint.
func languageReturnPath(path string) string {
	switch path {
	case "/", "/sign-in-failed", "/settings", "/help", "/privacy", "/terms", "/entries", "/calendar":
		return path
	}
	if strings.HasPrefix(path, "/entries/") {
		if _, err := time.Parse(time.DateOnly, strings.TrimPrefix(path, "/entries/")); err == nil {
			return path
		}
	}
	return "/settings"
}
