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
		setLanguageCookie(w, values[0], cfg.HTTPS)
		if _, original, ok := publicRoute(r.PostFormValue("returnTo")); ok {
			http.Redirect(w, r, localizedPublicPath(values[0], original), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, languageReturnPath(r.PostFormValue("returnTo")), http.StatusSeeOther)
	})
}

// Only return to browser pages, never an external origin or auth/API endpoint.
func languageReturnPath(path string) string {
	switch path {
	case "/", "/sign-in", "/sign-in-failed", "/settings", "/help", "/privacy", "/terms", "/entries", "/calendar":
		return path
	}
	if strings.HasPrefix(path, "/entries/") {
		if _, err := time.Parse(time.DateOnly, strings.TrimPrefix(path, "/entries/")); err == nil {
			return path
		}
	}
	return "/settings"
}

func setLanguageCookie(w http.ResponseWriter, locale string, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: languageCookieName, Value: locale, Path: "/", MaxAge: languagePreferenceSeconds, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode})
}
