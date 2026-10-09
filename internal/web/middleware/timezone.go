package middleware

import (
	"net/http"
	"net/url"
	"time"

	"github/mridha/myjournal-app/internal/journal"
)

const TimeZoneCookie = "mj_tz"

// TimeZone resolves the browser's IANA location once per request. Invalid,
// missing, and host-dependent Local values fall back to UTC.
func TimeZone(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		location := time.UTC
		if cookie, err := r.Cookie(TimeZoneCookie); err == nil && len(cookie.Value) <= 128 {
			name, err := url.PathUnescape(cookie.Value)
			if err == nil && name != "" && name != "Local" {
				if parsed, err := time.LoadLocation(name); err == nil {
					location = parsed
				}
			}
		}
		next.ServeHTTP(w, r.WithContext(journal.WithTimeZone(r.Context(), location)))
	})
}
