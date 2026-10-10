package handler

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"net/url"

	"github/mridha/myjournal-app/internal/journal"
	"github/mridha/myjournal-app/internal/provider"
	"github/mridha/myjournal-app/internal/session"
	"github/mridha/myjournal-app/internal/web/api"
)

func (s *Server) StartProviderLogin(w http.ResponseWriter, r *http.Request, providerName api.StartProviderLoginParamsProvider, params api.StartProviderLoginParams) {
	p, ok := s.registry.Get(string(providerName))
	if !ok {
		api.WriteError(w, http.StatusNotFound, s.message("errors.providerUnavailable"))
		return
	}

	request := provider.LoginRequest{}
	var pending *http.Cookie
	if p.ID() != "local" {
		flow, err := session.NewLoginFlow(p.ID(), s.now())
		if err != nil {
			api.WriteError(w, http.StatusInternalServerError, s.message("errors.session"))
			return
		}
		encoded, err := session.EncodeLoginFlow(s.cfg.SessionKey, flow)
		if err != nil {
			api.WriteError(w, http.StatusInternalServerError, s.message("errors.session"))
			return
		}
		pending = s.loginCookie(encoded, int(session.LoginLifetime.Seconds()))
		pending.Expires = flow.IssuedAt.Add(session.LoginLifetime)
		request = provider.LoginRequest{State: flow.State, Verifier: flow.Verifier, Nonce: flow.Nonce}
		current, sessionErr := s.currentSession(r)
		request.ForceConsent = (params.ForceConsent != nil && *params.ForceConsent == 1) || sessionErr != nil || current.ProviderID != p.ID()
	}
	authURL, err := p.Begin(r.Context(), request)
	if err != nil {
		api.WriteError(w, http.StatusBadGateway, s.message("errors.signInUnavailable"))
		return
	}
	if pending != nil {
		http.SetCookie(w, pending)
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (s *Server) CompleteProviderLogin(w http.ResponseWriter, r *http.Request, providerName api.CompleteProviderLoginParamsProvider, params api.CompleteProviderLoginParams) {
	p, ok := s.registry.Get(string(providerName))
	if !ok {
		api.WriteError(w, http.StatusNotFound, s.message("errors.providerUnavailable"))
		return
	}

	callback := provider.LoginCallback{}
	if p.ID() != "local" {
		http.SetCookie(w, s.loginCookie("", -1))
		flow, valid := s.callbackFlow(r, p.ID(), params)
		if !valid {
			api.WriteError(w, http.StatusUnauthorized, s.message("errors.signInFailed"))
			return
		}
		callback = provider.LoginCallback{Code: *params.Code, Verifier: flow.Verifier, Nonce: flow.Nonce}
		if current, err := s.currentSession(r); err == nil && current.ProviderID == p.ID() {
			callback.Previous = &current.Connection
		}
	}
	conn, err := p.Complete(r.Context(), callback)
	if err != nil {
		if errors.Is(err, journal.ErrUnavailable) {
			api.WriteError(w, http.StatusBadGateway, s.message("errors.signInUnavailable"))
		} else {
			api.WriteError(w, http.StatusUnauthorized, s.message("errors.signInFailed"))
		}
		return
	}

	stored := session.Session{Connection: conn, IssuedAt: s.now().UTC()}
	encoded, err := session.Encode(s.cfg.SessionKey, stored)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, s.message("errors.session"))
		return
	}

	if len(encoded) > 3600 {
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

func (s *Server) loginCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{Name: session.LoginCookieName, Value: value, Path: session.LoginCookiePath, MaxAge: maxAge, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.cfg.CookieSecure}
}

func (s *Server) callbackFlow(r *http.Request, providerID string, params api.CompleteProviderLoginParams) (session.LoginFlow, bool) {
	cookie, err := r.Cookie(session.LoginCookieName)
	if err != nil || len(cookie.Value) > 4096 {
		return session.LoginFlow{}, false
	}
	flow, err := session.DecodeLoginFlow(s.cfg.SessionKey, cookie.Value)
	// ParseQuery reports malformed encoding instead of silently dropping fields.
	query, queryErr := url.ParseQuery(r.URL.RawQuery)
	if err != nil || queryErr != nil || !flow.ValidAt(s.now()) || flow.ProviderID != providerID || params.Code == nil || *params.Code == "" || len(*params.Code) > 4096 || params.State == nil || len(query["code"]) != 1 || len(query["state"]) != 1 || query.Has("error") || subtle.ConstantTimeCompare([]byte(*params.State), []byte(flow.State)) != 1 {
		return session.LoginFlow{}, false
	}
	return flow, true
}
