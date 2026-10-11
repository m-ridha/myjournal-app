package middleware

import (
	"log/slog"
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// Recoverer logs only a generic event and request ID, never the recovered value
// or stack. respond writes the localized failure response before headers commit.
// After headers commit, abort the connection rather than append a false response.
func Recoverer(respond func(http.ResponseWriter, *http.Request)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			defer func() {
				if value := recover(); value != nil {
					if value == http.ErrAbortHandler {
						panic(http.ErrAbortHandler)
					}
					slog.ErrorContext(r.Context(), "request panic", "request_id", chimiddleware.GetReqID(r.Context()))
					if response.Status() != 0 {
						panic(http.ErrAbortHandler)
					}
					respond(response, r)
				}
			}()
			next.ServeHTTP(response, r)
		})
	}
}
