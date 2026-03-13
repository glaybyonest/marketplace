package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"marketplace-backend/internal/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rateLimitAuditMock struct {
	entries []observability.AuditEntry
}

func (m *rateLimitAuditMock) Record(ctx context.Context, entry observability.AuditEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
	assert.Equal(t, "camera=(), geolocation=(), microphone=()", rec.Header().Get("Permissions-Policy"))
	assert.Equal(t, "same-origin", rec.Header().Get("Cross-Origin-Opener-Policy"))
	assert.Equal(t, "none", rec.Header().Get("X-Permitted-Cross-Domain-Policies"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", rec.Header().Get("Strict-Transport-Security"))
}

func TestNoStore(t *testing.T) {
	handler := NoStore(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "no-store, max-age=0", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "no-cache", rec.Header().Get("Pragma"))
	assert.Contains(t, rec.Header().Values("Vary"), "Authorization")
}

func TestRateLimiter(t *testing.T) {
	audit := &rateLimitAuditMock{}
	limiter := NewRateLimiter(audit)
	current := time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return current }

	policy := RateLimitPolicy{
		Name:   "auth_login",
		Limit:  2,
		Window: time.Minute,
	}
	handler := limiter.Middleware(policy)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.RemoteAddr = "203.0.113.10:1234"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNoContent, rec.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "60", rec.Header().Get("Retry-After"))
	require.NotEmpty(t, audit.entries)
	assert.Equal(t, "security.rate_limited", audit.entries[len(audit.entries)-1].Action)

	current = current.Add(time.Minute + time.Second)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}
