package middleware

import (
	"net/http"

	"marketplace-backend/internal/observability"
)

func ErrorReporter(reporter *observability.ErrorReporter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if reporter == nil {
				next.ServeHTTP(w, r)
				return
			}
			ctx := observability.WithErrorReporter(r.Context(), reporter)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
