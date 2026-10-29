package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"myjournal-app/internal/driver/web/api"
)

// RequestAuthority restricts requests to the configured host and effective port.
// baseURL must be the validated startup origin; proxy headers are never trusted.
func RequestAuthority(baseURL string) func(http.Handler) http.Handler {
	authorities := make(map[string]bool, 2)
	if origin, err := url.Parse(baseURL); err == nil && origin.Host != "" {
		authority := strings.ToLower(origin.Host)
		authorities[authority] = true
		defaultPort := "80"
		if origin.Scheme == "https" {
			defaultPort = "443"
		}
		// Browsers omit an explicit default port from the request authority.
		switch origin.Port() {
		case "":
			authorities[authority+":"+defaultPort] = true
		case defaultPort:
			authorities[strings.TrimSuffix(authority, ":"+defaultPort)] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authorities[strings.ToLower(r.Host)] {
				w.Header().Set("Cache-Control", "no-store")
				api.WriteError(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
