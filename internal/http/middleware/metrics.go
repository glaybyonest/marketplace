package middleware

import (
	"net/http"
	"time"

	"marketplace-backend/internal/observability"

	"github.com/go-chi/chi/v5"
)

type metricsRecorder struct {
	http.ResponseWriter
	status int
}

func (r *metricsRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func Metrics(metrics *observability.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if metrics != nil {
				metrics.IncInFlight()
				defer metrics.DecInFlight()
			}

			startedAt := time.Now()
			rec := &metricsRecorder{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(rec, r)

			if metrics != nil {
				metrics.ObserveHTTPRequest(r.Method, routePattern(r), rec.status, time.Since(startedAt))
			}
		})
	}
}

func routePattern(r *http.Request) string {
	if routeCtx := chi.RouteContext(r.Context()); routeCtx != nil {
		if pattern := routeCtx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	if r.URL.Path == "" {
		return "/"
	}
	return r.URL.Path
}
