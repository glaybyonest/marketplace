package middleware

import (
	"net/http"
	"strings"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/http/response"
	"marketplace-backend/internal/observability"
	"marketplace-backend/internal/security"
)

type CSRFMiddleware struct {
	enabled bool
	cookies security.CookieAuthConfig
	audit   rateLimitAuditor
}

func NewCSRF(enabled bool, cookies security.CookieAuthConfig, audit rateLimitAuditor) *CSRFMiddleware {
	return &CSRFMiddleware{
		enabled: enabled,
		cookies: cookies,
		audit:   audit,
	}
}

func (m *CSRFMiddleware) Handler(next http.Handler) http.Handler {
	if m == nil || !m.enabled || !m.cookies.Enabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSafeMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}

		accessToken := m.cookies.AccessToken(r)
		refreshToken := m.cookies.RefreshToken(r)
		if accessToken == "" && refreshToken == "" {
			next.ServeHTTP(w, r)
			return
		}

		cookieToken := m.cookies.CSRFToken(r)
		headerToken := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
		if cookieToken == "" || headerToken == "" || cookieToken != headerToken {
			if m.audit != nil {
				_ = m.audit.Record(r.Context(), observability.AuditEntry{
					Action:     "security.csrf_rejected",
					EntityType: "request",
					Metadata: map[string]any{
						"path":      r.URL.Path,
						"method":    r.Method,
						"remote_ip": requestClientIP(r),
					},
				})
			}
			response.FromDomainError(w, domain.ErrCSRFInvalid)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isSafeMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}
