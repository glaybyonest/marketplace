package middleware

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/http/response"
	"marketplace-backend/internal/observability"
)

type rateLimitAuditor interface {
	Record(ctx context.Context, entry observability.AuditEntry) error
}

type RateLimitPolicy struct {
	Name   string
	Limit  int
	Window time.Duration
}

type rateLimitEntry struct {
	windowStartedAt time.Time
	count           int
}

type RateLimiter struct {
	mu       sync.Mutex
	entries  map[string]rateLimitEntry
	requests uint64
	now      func() time.Time
	audit    rateLimitAuditor
}

func NewRateLimiter(audit rateLimitAuditor) *RateLimiter {
	return &RateLimiter{
		entries: make(map[string]rateLimitEntry),
		now: func() time.Time {
			return time.Now().UTC()
		},
		audit: audit,
	}
}

func (l *RateLimiter) Middleware(policy RateLimitPolicy) func(http.Handler) http.Handler {
	if l == nil || policy.Limit <= 0 || policy.Window <= 0 {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := policy.Name + ":" + requestClientIP(r)
			allowed, retryAfter := l.allow(key, policy)
			if allowed {
				next.ServeHTTP(w, r)
				return
			}

			retrySeconds := retryAfterSeconds(retryAfter)
			w.Header().Set("Retry-After", strconv.Itoa(retrySeconds))
			if l.audit != nil {
				_ = l.audit.Record(r.Context(), observability.AuditEntry{
					Action:     "security.rate_limited",
					EntityType: "auth_endpoint",
					Metadata: map[string]any{
						"policy":              policy.Name,
						"path":                r.URL.Path,
						"remote_ip":           requestClientIP(r),
						"retry_after_seconds": retrySeconds,
					},
				})
			}
			response.FromDomainError(w, &domain.RateLimitError{
				Scope:      policy.Name,
				RetryAfter: retryAfter,
			})
		})
	}
}

func (l *RateLimiter) allow(key string, policy RateLimitPolicy) (bool, time.Duration) {
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.requests++
	if l.requests%256 == 0 {
		l.pruneExpired(now)
	}

	entry, ok := l.entries[key]
	if !ok || now.Sub(entry.windowStartedAt) >= policy.Window {
		l.entries[key] = rateLimitEntry{
			windowStartedAt: now,
			count:           1,
		}
		return true, 0
	}
	if entry.count >= policy.Limit {
		return false, policy.Window - now.Sub(entry.windowStartedAt)
	}

	entry.count++
	l.entries[key] = entry
	return true, 0
}

func (l *RateLimiter) pruneExpired(now time.Time) {
	for key, entry := range l.entries {
		if now.Sub(entry.windowStartedAt) >= time.Hour {
			delete(l.entries, key)
		}
	}
}

func retryAfterSeconds(delay time.Duration) int {
	if delay <= 0 {
		return 1
	}
	seconds := int(delay / time.Second)
	if delay%time.Second != 0 {
		seconds++
	}
	if seconds <= 0 {
		return 1
	}
	return seconds
}
