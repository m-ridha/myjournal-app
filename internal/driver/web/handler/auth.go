package handler

import (
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	provider "github/mridha/myjournal-app/internal/core/connection"
	"github/mridha/myjournal-app/internal/core/journal"
	"github/mridha/myjournal-app/internal/driver/web/api"
	"github/mridha/myjournal-app/internal/driver/web/authsession"
)

func (s *Server) StartProviderLogin(w http.ResponseWriter, r *http.Request, providerName api.StartProviderLoginParamsProvider, params api.StartProviderLoginParams) {

	current, sessionErr := s.currentSession(r)
	var previous *provider.Session
	if sessionErr == nil {
		previous = &current
	}
	result, err := s.connections.Begin(r.Context(), provider.BeginInput{ProviderID: string(providerName), Previous: previous, ForceConsent: params.ForceConsent != nil && *params.ForceConsent == 1})
	if err != nil {
		status, key := http.StatusBadGateway, "errors.signInUnavailable"
		if errors.Is(err, provider.ErrProviderUnavailable) {
			status, key = http.StatusNotFound, "errors.providerUnavailable"
		}
		if errors.Is(err, provider.ErrLoginFlow) {
			status, key = http.StatusInternalServerError, "errors.session"
		}
		api.WriteError(w, status, s.message(key))
		return
	}
	if result.Pending != nil {
		encoded, err := s.cfg.Codec.EncodeLoginFlow(*result.Pending)
		if err != nil {
			api.WriteError(w, http.StatusInternalServerError, s.message("errors.session"))
			return
		}
		pending := s.loginCookie(encoded, int(provider.LoginLifetime.Seconds()))
		pending.Expires = result.Pending.IssuedAt.Add(provider.LoginLifetime)
		http.SetCookie(w, pending)
	}
	http.Redirect(w, r, result.URL, http.StatusFound)
}

func (s *Server) CompleteProviderLogin(w http.ResponseWriter, r *http.Request, providerName api.CompleteProviderLoginParamsProvider, params api.CompleteProviderLoginParams) {
	p, ok := s.registry.Lookup(string(providerName))
	if !ok {
		s.writeLoginFailure(w, r, http.StatusNotFound, s.message("errors.providerUnavailable"))
		return
	}

	input := provider.CompleteInput{ProviderID: p.ID()}
	if p.RequiresLoginFlow() {
		http.SetCookie(w, s.loginCookie("", -1))
		flow, valid := s.callbackFlow(r, params)
		if !valid {
			logLoginRejection(provider.LoginTransaction)
			s.writeLoginFailure(w, r, http.StatusUnauthorized, s.message("errors.signInFailed"))
			return
		}
		input.Pending = &flow
		input.Code = *params.Code
		input.State = *params.State
		if current, err := s.currentSession(r); err == nil {
			input.Previous = &current
		}
	}
	stored, err := s.connections.Complete(r.Context(), input)
	if err != nil {
		var rejection *provider.LoginError
		if errors.As(err, &rejection) {
			logLoginRejection(rejection.Stage)
		}
		if errors.Is(err, journal.ErrUnavailable) {
			s.writeLoginFailure(w, r, http.StatusBadGateway, s.message("errors.signInUnavailable"))
		} else {
			s.writeLoginFailure(w, r, http.StatusUnauthorized, s.message("errors.signInFailed"))
		}
		return
	}

	encoded, err := s.cfg.Codec.Encode(stored)
	if err != nil {
		s.writeLoginFailure(w, r, http.StatusInternalServerError, s.message("errors.session"))
		return
	}

	if len(encoded) > authsession.MaxEncodedBytes {
		s.writeLoginFailure(w, r, http.StatusInternalServerError, s.message("errors.session"))
		return
	}
	http.SetCookie(w, authsession.Cookie(encoded, stored.IssuedAt, s.cfg.CookieSecure))
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, authsession.Cookie("", time.Time{}, s.cfg.CookieSecure))
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

	if err := s.connections.Disconnect(r.Context(), current); err != nil {
		s.writeServiceError(w, err)
		return
	}
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) loginCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{Name: authsession.LoginCookieName, Value: value, Path: authsession.LoginCookiePath, MaxAge: maxAge, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.cfg.CookieSecure}
}

func (s *Server) callbackFlow(r *http.Request, params api.CompleteProviderLoginParams) (provider.LoginFlow, bool) {
	cookie, err := r.Cookie(authsession.LoginCookieName)
	if err != nil || len(cookie.Value) > authsession.MaxCookieBytes {
		return provider.LoginFlow{}, false
	}
	flow, err := s.cfg.Codec.DecodeLoginFlow(cookie.Value)
	// ParseQuery reports malformed encoding instead of silently dropping fields.
	query, queryErr := url.ParseQuery(r.URL.RawQuery)
	if err != nil || queryErr != nil {
		return provider.LoginFlow{}, false
	}
	if query.Has("error") || len(query["code"]) != 1 || len(query["state"]) != 1 {
		return provider.LoginFlow{}, false
	}
	if params.Code == nil || *params.Code == "" || len(*params.Code) > maxAuthorizationCodeBytes || params.State == nil {
		return provider.LoginFlow{}, false
	}
	return flow, true
}

const maxAuthorizationCodeBytes = 4096

func logLoginRejection(stage provider.LoginStage) {
	// Never log arbitrary provider values, token errors, claims, or callback queries.
	switch stage {
	case provider.LoginTransaction, provider.LoginExchange, provider.LoginClaims, provider.LoginGrant, provider.LoginAccount, provider.LoginRefresh:
		slog.Warn("sign-in rejected", "stage", stage)
	}
}

// Browser callbacks recover at a fixed, query-free page. API clients retain JSON errors.
func (s *Server) writeLoginFailure(w http.ResponseWriter, r *http.Request, status int, message string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "Accept")
	for _, item := range strings.Split(r.Header.Get("Accept"), ",") {
		mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(item))
		if err != nil || mediaType != "text/html" {
			continue
		}
		if quality, ok := params["q"]; ok {
			q, err := strconv.ParseFloat(quality, 64)
			if err != nil || !(q > 0 && q <= 1) {
				continue
			}
		}
		http.Redirect(w, r, "/sign-in-failed", http.StatusSeeOther)
		return
	}
	api.WriteError(w, status, message)
}
