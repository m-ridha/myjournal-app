package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"myjournal-app/internal/core/journal"
)

func TestTimeZoneResolution(t *testing.T) {
	for _, tc := range []struct{ value, want string }{
		{"", "UTC"}, {"Asia%2FJakarta", "Asia/Jakarta"}, {"America/Los_Angeles", "America/Los_Angeles"},
		{"Local", "UTC"}, {"unknown/location", "UTC"}, {"%zz", "UTC"}, {strings.Repeat("a", 129), "UTC"},
	} {
		t.Run(tc.value, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.value != "" {
				request.AddCookie(&http.Cookie{Name: TimeZoneCookie, Value: tc.value})
			}
			TimeZone(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				if got := journal.TimeZone(r.Context()).String(); got != tc.want {
					t.Fatalf("zone=%s want %s", got, tc.want)
				}
				r.Header.Set("Cookie", "mj_tz=Pacific/Honolulu")
				if got := journal.TimeZone(r.Context()).String(); got != tc.want {
					t.Fatal("location was resolved again")
				}
			})).ServeHTTP(httptest.NewRecorder(), request)
		})
	}
}
