package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"marketplace-backend/internal/observability"

	"github.com/google/uuid"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(payload []byte) (int, error) {
	size, err := r.ResponseWriter.Write(payload)
	r.bytes += size
	return size, err
}

func Logger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			rec := &statusRecorder{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(rec, r)

			level := slog.LevelInfo
			switch {
			case rec.status >= http.StatusInternalServerError:
				level = slog.LevelError
			case rec.status >= http.StatusBadRequest:
				level = slog.LevelWarn
			}

			requestMeta := observability.RequestMetaFromContext(r.Context())
			route := routePattern(r)
			attrs := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"route", route,
				"status", rec.status,
				"duration_ms", time.Since(startedAt).Milliseconds(),
				"request_id", requestMeta.RequestID,
				"remote_ip", requestMeta.RemoteIP,
				"user_agent", requestMeta.UserAgent,
				"response_bytes", rec.bytes,
			}
			if actor, ok := observability.ActorFromContext(r.Context()); ok && actor.UserID != uuid.Nil {
				attrs = append(attrs,
					"user_id", actor.UserID.String(),
					"user_role", actor.Role,
				)
			}

			logger.Log(r.Context(), level, "http_request", attrs...)
		})
	}
}
