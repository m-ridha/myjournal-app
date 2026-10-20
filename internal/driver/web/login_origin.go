package web

import (
	"net/http"
	"net/url"
	"strings"
)

// canonicalGoogleLogin starts production OAuth on the configured origin so its
// host-only pending cookie reaches the callback. Other routes retain draft access.
func canonicalGoogleLogin(baseURL string) func(http.Handler) http.Handler {
	origin, err := url.Parse(baseURL)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err == nil && origin.Scheme == "https" && r.Method == http.MethodGet &&
				r.URL.Path == "/api/v1/auth/google/login" && !strings.EqualFold(r.Host, origin.Host) {
				// Do not forward query parameters, callbacks, or browser credentials.
				http.Redirect(w, r, strings.TrimRight(baseURL, "/")+"/api/v1/auth/google/login", http.StatusFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
