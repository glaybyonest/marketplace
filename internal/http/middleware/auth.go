package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/http/response"
	"marketplace-backend/internal/observability"
	"marketplace-backend/internal/security"

	"github.com/google/uuid"
)

type SessionToucher interface {
	TouchLastSeen(ctx context.Context, sessionID uuid.UUID, seenAt time.Time) error
	IsActive(ctx context.Context, sessionID uuid.UUID, now time.Time) (bool, error)
}

type Auth struct {
	jwt      *security.JWTManager
	cookies  security.CookieAuthConfig
	sessions SessionToucher
}

func NewAuth(jwt *security.JWTManager, cookies security.CookieAuthConfig, sessions SessionToucher) *Auth {
	return &Auth{jwt: jwt, cookies: cookies, sessions: sessions}
}

func (a *Auth) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a == nil || a.jwt == nil {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "missing authorization handler", nil)
			return
		}

		token := ""

		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid authorization header", nil)
				return
			}
			token = strings.TrimSpace(parts[1])
		} else {
			token = a.cookies.AccessToken(r)
			if token == "" {
				response.Error(w, http.StatusUnauthorized, "unauthorized", "missing authorization header", nil)
				return
			}
		}

		claims, err := a.jwt.Parse(token)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid or expired access token", nil)
			return
		}

		userID, err := security.UserIDFromClaims(claims)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid token subject", nil)
			return
		}
		sessionID, err := security.SessionIDFromClaims(claims)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid token session", nil)
			return
		}
		now := time.Now().UTC()
		if a.sessions != nil && sessionID != uuid.Nil {
			active, err := a.sessions.IsActive(r.Context(), sessionID, now)
			if err != nil {
				response.FromDomainError(w, domain.ErrUnauthorized)
				return
			}
			if !active {
				a.cookies.ClearAuthCookies(w)
				response.FromDomainError(w, domain.ErrSessionClosed)
				return
			}
			_ = a.sessions.TouchLastSeen(r.Context(), sessionID, now)
		}

		role := domain.UserRole(strings.TrimSpace(claims.Role))
		if role == "" {
			role = domain.UserRoleCustomer
		}

		ctx := WithAuth(r.Context(), userID, sessionID, claims.Email, role)
		ctx = observability.WithActor(ctx, observability.Actor{
			UserID: userID,
			Email:  claims.Email,
			Role:   string(role),
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
