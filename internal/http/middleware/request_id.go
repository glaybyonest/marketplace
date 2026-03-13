package middleware

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"marketplace-backend/internal/observability"
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = observability.NewRequestID()
		}

		ctx := observability.WithRequestID(r.Context(), requestID)
		ctx = observability.WithRequestMeta(ctx, observability.RequestMeta{
			RequestID: requestID,
			Method:    r.Method,
			Path:      r.URL.Path,
			RemoteIP:  requestClientIP(r),
			UserAgent: strings.TrimSpace(r.UserAgent()),
		})
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestClientIP(r *http.Request) string {
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	if errors.Is(err, net.ErrClosed) {
		return ""
	}
	return strings.TrimSpace(r.RemoteAddr)
}
