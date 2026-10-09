package handler

import (
	"net/http"

	"github/mridha/myjournal-app/internal/session"
	"github/mridha/myjournal-app/internal/web/api"
)

func (s *Server) StartProviderLogin(w http.ResponseWriter, r *http.Request, providerName api.StartProviderLoginParamsProvider) {
	p, ok := s.registry.Get(string(providerName))
	if !ok {
		api.WriteError(w, http.StatusNotFound, s.message("errors.providerUnavailable"))
		return
	}

	authURL, err := p.Begin(r.Context(), "")
	if err != nil {
		api.WriteError(w, http.StatusBadGateway, s.message("errors.signInUnavailable"))
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (s *Server) CompleteProviderLogin(w http.ResponseWriter, r *http.Request, providerName api.CompleteProviderLoginParamsProvider, params api.CompleteProviderLoginParams) {
	p, ok := s.registry.Get(string(providerName))
	if !ok {
		api.WriteError(w, http.StatusNotFound, s.message("errors.providerUnavailable"))
		return
	}

	conn, err := p.Complete(r.Context(), "")
	if err != nil {
		api.WriteError(w, http.StatusUnauthorized, s.message("errors.signInFailed"))
		return
	}

	stored := session.Session{Connection: conn, IssuedAt: s.now().UTC()}
	encoded, err := session.Encode(s.cfg.SessionKey, stored)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, s.message("errors.session"))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     session.CookieName,
		Value:    encoded,
		Expires:  stored.IssuedAt.Add(session.Lifetime),
		MaxAge:   int(session.Lifetime.Seconds()),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.CookieSecure,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     session.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.CookieSecure,
	})
}

func (s *Server) Logout(w http.ResponseWriter, r *http.Request) {
	s.clearSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) Disconnect(w http.ResponseWriter, r *http.Request) {
	current, err := s.currentSession(r)
	if err != nil {
		s.writeServiceError(w, err)
		return
	}
	p, ok := s.registry.Get(current.ProviderID)
	if !ok {
		s.writeServiceError(w, ErrUnauthenticated)
		return
	}
	if err := p.Revoke(r.Context(), current.Connection); err != nil {
		s.writeServiceError(w, err)
		return
	}
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
