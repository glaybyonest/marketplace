package handlers

import (
	"context"
	"net/http"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/http/dto"
	httpmw "marketplace-backend/internal/http/middleware"
	"marketplace-backend/internal/http/response"
	"marketplace-backend/internal/usecase"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type AuthService interface {
	Register(ctx context.Context, input usecase.RegisterInput) (domain.AuthResult, error)
	Login(ctx context.Context, input usecase.LoginInput) (domain.AuthResult, error)
	Refresh(ctx context.Context, input usecase.RefreshInput) (domain.TokenPair, error)
	Logout(ctx context.Context, input usecase.LogoutInput) error
	Me(ctx context.Context, userID uuid.UUID) (domain.User, error)
	RequestEmailVerification(ctx context.Context, input usecase.VerifyEmailRequestInput) error
	ConfirmEmailVerification(ctx context.Context, input usecase.VerifyEmailConfirmInput) (domain.User, error)
	RequestPasswordReset(ctx context.Context, input usecase.PasswordResetRequestInput) error
	ConfirmPasswordReset(ctx context.Context, input usecase.PasswordResetConfirmInput) error
}

type AuthHandler struct {
	service  AuthService
	validate *validator.Validate
}

func NewAuthHandler(service AuthService) *AuthHandler {
	return &AuthHandler{
		service:  service,
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	result, err := h.service.Register(r.Context(), usecase.RegisterInput{
		Email:     req.Email,
		Password:  req.Password,
		FullName:  req.FullName,
		UserAgent: r.UserAgent(),
		IP:        getClientIP(r),
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusCreated, result)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	result, err := h.service.Login(r.Context(), usecase.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: r.UserAgent(),
		IP:        getClientIP(r),
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	tokens, err := h.service.Refresh(r.Context(), usecase.RefreshInput{
		RefreshToken: req.RefreshToken,
		UserAgent:    r.UserAgent(),
		IP:           getClientIP(r),
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, tokens)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	var req dto.LogoutRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	if err := h.service.Logout(r.Context(), usecase.LogoutInput{
		UserID:       userID,
		RefreshToken: req.RefreshToken,
	}); err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"revoked": true})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpmw.UserID(r.Context())
	if !ok {
		writeDomainError(w, r, domain.ErrUnauthorized)
		return
	}

	user, err := h.service.Me(r.Context(), userID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, user)
}

func (h *AuthHandler) RequestEmailVerification(w http.ResponseWriter, r *http.Request) {
	var req dto.VerifyEmailRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	if err := h.service.RequestEmailVerification(r.Context(), usecase.VerifyEmailRequestInput{
		Email: req.Email,
	}); err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"accepted": true})
}

func (h *AuthHandler) ConfirmEmailVerification(w http.ResponseWriter, r *http.Request) {
	var req dto.VerifyEmailConfirmRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	user, err := h.service.ConfirmEmailVerification(r.Context(), usecase.VerifyEmailConfirmInput{
		Token: req.Token,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"verified": true,
		"user":     user,
	})
}

func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req dto.PasswordResetRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	if err := h.service.RequestPasswordReset(r.Context(), usecase.PasswordResetRequestInput{
		Email: req.Email,
	}); err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"accepted": true})
}

func (h *AuthHandler) ConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req dto.PasswordResetConfirmRequest
	if err := decodeAndValidate(r, &req, h.validate); err != nil {
		writeDomainError(w, r, err)
		return
	}

	if err := h.service.ConfirmPasswordReset(r.Context(), usecase.PasswordResetConfirmInput{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	}); err != nil {
		writeDomainError(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"password_reset": true})
}
