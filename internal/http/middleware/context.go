package middleware

import (
	"context"

	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
)

type authContextKey string

const (
	userIDKey authContextKey = "auth_user_id"
	emailKey  authContextKey = "auth_email"
	roleKey   authContextKey = "auth_role"
)

func WithAuth(ctx context.Context, userID uuid.UUID, email string, role domain.UserRole) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, emailKey, email)
	ctx = context.WithValue(ctx, roleKey, role)
	return ctx
}

func UserID(ctx context.Context) (uuid.UUID, bool) {
	value, ok := ctx.Value(userIDKey).(uuid.UUID)
	return value, ok
}

func Email(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(emailKey).(string)
	return value, ok
}

func Role(ctx context.Context) (domain.UserRole, bool) {
	value, ok := ctx.Value(roleKey).(domain.UserRole)
	return value, ok
}
