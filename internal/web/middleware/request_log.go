package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// RequestLog deliberately omits query strings, headers, and bodies.
func RequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		response := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
		defer func() {
			slog.Info("request", "method", r.Method, "path", r.URL.Path, "status", response.Status(), "duration", time.Since(start), "request_id", chimiddleware.GetReqID(r.Context()))
		}()
		next.ServeHTTP(response, r)
	})
}
