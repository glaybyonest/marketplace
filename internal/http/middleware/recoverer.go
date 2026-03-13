package middleware

import (
	"log/slog"
	"net/http"

	"marketplace-backend/internal/http/response"
	"marketplace-backend/internal/observability"
)

func Recoverer(logger *slog.Logger, reporter *observability.ErrorReporter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					route := routePattern(r)
					logger.Error("http_panic", "path", r.URL.Path, "route", route, "method", r.Method, "request_id", observability.RequestIDFromContext(r.Context()))
					if reporter != nil {
						reporter.CapturePanic(r.Context(), recovered, r.Method, r.URL.Path, route)
					}
					response.Error(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
