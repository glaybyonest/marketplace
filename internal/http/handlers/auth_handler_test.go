package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"marketplace-backend/internal/domain"
	httpmw "marketplace-backend/internal/http/middleware"
	"marketplace-backend/internal/usecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type authServiceStub struct {
	registerFn                 func(ctx context.Context, input usecase.RegisterInput) (domain.AuthResult, error)
	loginFn                    func(ctx context.Context, input usecase.LoginInput) (domain.AuthResult, error)
	refreshFn                  func(ctx context.Context, input usecase.RefreshInput) (domain.TokenPair, error)
	logoutFn                   func(ctx context.Context, input usecase.LogoutInput) error
	meFn                       func(ctx context.Context, userID uuid.UUID) (domain.User, error)
	requestEmailVerificationFn func(ctx context.Context, input usecase.VerifyEmailRequestInput) error
	confirmEmailVerificationFn func(ctx context.Context, input usecase.VerifyEmailConfirmInput) (domain.User, error)
	requestPasswordResetFn     func(ctx context.Context, input usecase.PasswordResetRequestInput) error
	confirmPasswordResetFn     func(ctx context.Context, input usecase.PasswordResetConfirmInput) error
}

func (s *authServiceStub) Register(ctx context.Context, input usecase.RegisterInput) (domain.AuthResult, error) {
	return s.registerFn(ctx, input)
}

func (s *authServiceStub) Login(ctx context.Context, input usecase.LoginInput) (domain.AuthResult, error) {
	return s.loginFn(ctx, input)
}

func (s *authServiceStub) Refresh(ctx context.Context, input usecase.RefreshInput) (domain.TokenPair, error) {
	return s.refreshFn(ctx, input)
}

func (s *authServiceStub) Logout(ctx context.Context, input usecase.LogoutInput) error {
	return s.logoutFn(ctx, input)
}

func (s *authServiceStub) Me(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	return s.meFn(ctx, userID)
}

func (s *authServiceStub) RequestEmailVerification(ctx context.Context, input usecase.VerifyEmailRequestInput) error {
	return s.requestEmailVerificationFn(ctx, input)
}

func (s *authServiceStub) ConfirmEmailVerification(ctx context.Context, input usecase.VerifyEmailConfirmInput) (domain.User, error) {
	return s.confirmEmailVerificationFn(ctx, input)
}

func (s *authServiceStub) RequestPasswordReset(ctx context.Context, input usecase.PasswordResetRequestInput) error {
	return s.requestPasswordResetFn(ctx, input)
}

func (s *authServiceStub) ConfirmPasswordReset(ctx context.Context, input usecase.PasswordResetConfirmInput) error {
	return s.confirmPasswordResetFn(ctx, input)
}

func TestAuthHandler(t *testing.T) {
	userID := uuid.New()
	stub := &authServiceStub{
		registerFn: func(ctx context.Context, input usecase.RegisterInput) (domain.AuthResult, error) {
			return domain.AuthResult{
				User:                      domain.User{ID: userID, Email: input.Email, Role: domain.UserRoleCustomer},
				RequiresEmailVerification: true,
				Message:                   "verification email sent",
			}, nil
		},
		loginFn: func(ctx context.Context, input usecase.LoginInput) (domain.AuthResult, error) {
			return domain.AuthResult{
				User: domain.User{ID: userID, Email: input.Email, Role: domain.UserRoleCustomer},
				Tokens: &domain.TokenPair{
					AccessToken:  "a",
					RefreshToken: "r",
					TokenType:    "Bearer",
				},
			}, nil
		},
		refreshFn: func(ctx context.Context, input usecase.RefreshInput) (domain.TokenPair, error) {
			return domain.TokenPair{AccessToken: "a2", RefreshToken: "r2", TokenType: "Bearer"}, nil
		},
		logoutFn: func(ctx context.Context, input usecase.LogoutInput) error { return nil },
		meFn: func(ctx context.Context, userID uuid.UUID) (domain.User, error) {
			return domain.User{ID: userID, Email: "me@example.com", Role: domain.UserRoleCustomer}, nil
		},
		requestEmailVerificationFn: func(ctx context.Context, input usecase.VerifyEmailRequestInput) error { return nil },
		confirmEmailVerificationFn: func(ctx context.Context, input usecase.VerifyEmailConfirmInput) (domain.User, error) {
			return domain.User{ID: userID, Email: "verified@example.com", Role: domain.UserRoleCustomer, IsEmailVerified: true}, nil
		},
		requestPasswordResetFn: func(ctx context.Context, input usecase.PasswordResetRequestInput) error { return nil },
		confirmPasswordResetFn: func(ctx context.Context, input usecase.PasswordResetConfirmInput) error { return nil },
	}
	handler := NewAuthHandler(stub)

	t.Run("register invalid payload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"bad"}`))
		rec := httptest.NewRecorder()
		handler.Register(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("register success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{"email":"user@example.com","password":"StrongPass1","full_name":"User"}`))
		rec := httptest.NewRecorder()
		handler.Register(rec, req)
		require.Equal(t, http.StatusCreated, rec.Code)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
		_, ok := payload["data"]
		assert.True(t, ok)
	})

	t.Run("login success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"user@example.com","password":"StrongPass1"}`))
		rec := httptest.NewRecorder()
		handler.Login(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("request verification success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/verify-email/request", strings.NewReader(`{"email":"user@example.com"}`))
		rec := httptest.NewRecorder()
		handler.RequestEmailVerification(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("confirm verification success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/verify-email/confirm", strings.NewReader(`{"token":"abcdefghijklmnopqrstuvwxyz123456"}`))
		rec := httptest.NewRecorder()
		handler.ConfirmEmailVerification(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("request password reset success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/password-reset/request", strings.NewReader(`{"email":"user@example.com"}`))
		rec := httptest.NewRecorder()
		handler.RequestPasswordReset(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("confirm password reset success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/password-reset/confirm", strings.NewReader(`{"token":"abcdefghijklmnopqrstuvwxyz123456","new_password":"StrongPass2"}`))
		rec := httptest.NewRecorder()
		handler.ConfirmPasswordReset(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("refresh invalid payload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/refresh", strings.NewReader(`{"refresh_token":""}`))
		rec := httptest.NewRecorder()
		handler.Refresh(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("me unauthorized without context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		rec := httptest.NewRecorder()
		handler.Me(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("me success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		req = req.WithContext(httpmw.WithAuth(req.Context(), userID, "me@example.com", domain.UserRoleCustomer))
		rec := httptest.NewRecorder()
		handler.Me(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
