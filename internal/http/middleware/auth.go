package middleware

import (
	"net/http"
	"strings"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/http/response"
	"marketplace-backend/internal/security"
)

type Auth struct {
	jwt *security.JWTManager
}

func NewAuth(jwt *security.JWTManager) *Auth {
	return &Auth{jwt: jwt}
}

func (a *Auth) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if authHeader == "" {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "missing authorization header", nil)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid authorization header", nil)
			return
		}

		claims, err := a.jwt.Parse(strings.TrimSpace(parts[1]))
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid or expired access token", nil)
			return
		}

		userID, err := security.UserIDFromClaims(claims)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid token subject", nil)
			return
		}

		role := domain.UserRole(strings.TrimSpace(claims.Role))
		if role == "" {
			role = domain.UserRoleCustomer
		}

		next.ServeHTTP(w, r.WithContext(WithAuth(r.Context(), userID, claims.Email, role)))
	})
}
