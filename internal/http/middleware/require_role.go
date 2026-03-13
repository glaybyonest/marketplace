package middleware

import (
	"net/http"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/http/response"
)

func RequireRole(role domain.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			currentRole, ok := Role(r.Context())
			if !ok {
				response.FromDomainError(w, domain.ErrUnauthorized)
				return
			}
			if currentRole != role {
				response.FromDomainError(w, domain.ErrForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
