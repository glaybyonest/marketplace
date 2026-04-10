package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"marketplace-backend/internal/domain"
)

type successEnvelope struct {
	Data any `json:"data"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type DomainErrorDescriptor struct {
	Status  int
	Code    string
	Message string
	Details any
}

func JSON(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, successEnvelope{Data: data})
}

func Error(w http.ResponseWriter, status int, code, message string, details any) {
	writeJSON(w, status, errorEnvelope{
		Error: errorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

func DescribeDomainError(err error) DomainErrorDescriptor {
	var rateLimitErr *domain.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return DomainErrorDescriptor{
			Status:  http.StatusTooManyRequests,
			Code:    "rate_limited",
			Message: "too many requests",
			Details: map[string]any{
				"scope":               rateLimitErr.Scope,
				"retry_after_seconds": retryAfterSeconds(rateLimitErr.RetryAfter),
			},
		}
	}

	var loginLockedErr *domain.LoginLockedError
	if errors.As(err, &loginLockedErr) {
		return DomainErrorDescriptor{
			Status:  http.StatusTooManyRequests,
			Code:    "login_locked",
			Message: "account is temporarily locked",
			Details: map[string]any{
				"retry_after_seconds": retryAfterSeconds(loginLockedErr.RetryAfter),
				"locked_until":        loginLockedErr.LockedUntil.UTC().Format(time.RFC3339),
			},
		}
	}

	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		return DomainErrorDescriptor{Status: http.StatusBadRequest, Code: "invalid_input", Message: "invalid input"}
	case errors.Is(err, domain.ErrUnprocessable):
		return DomainErrorDescriptor{Status: http.StatusUnprocessableEntity, Code: "invalid_request", Message: "invalid request"}
	case errors.Is(err, domain.ErrCSRFInvalid):
		return DomainErrorDescriptor{Status: http.StatusForbidden, Code: "csrf_invalid", Message: "invalid csrf token"}
	case errors.Is(err, domain.ErrCartEmpty):
		return DomainErrorDescriptor{Status: http.StatusBadRequest, Code: "cart_empty", Message: "cart is empty"}
	case errors.Is(err, domain.ErrUnauthorized):
		return DomainErrorDescriptor{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "unauthorized"}
	case errors.Is(err, domain.ErrInvalidLoginCode):
		return DomainErrorDescriptor{Status: http.StatusUnauthorized, Code: "invalid_login_code", Message: "login code is invalid or expired"}
	case errors.Is(err, domain.ErrInvalidToken):
		return DomainErrorDescriptor{Status: http.StatusBadRequest, Code: "invalid_token", Message: "invalid or expired token"}
	case errors.Is(err, domain.ErrInactiveUser):
		return DomainErrorDescriptor{Status: http.StatusForbidden, Code: "inactive_user", Message: "user is inactive"}
	case errors.Is(err, domain.ErrEmailNotVerified):
		return DomainErrorDescriptor{Status: http.StatusForbidden, Code: "email_not_verified", Message: "email is not verified"}
	case errors.Is(err, domain.ErrForbidden):
		return DomainErrorDescriptor{Status: http.StatusForbidden, Code: "forbidden", Message: "forbidden"}
	case errors.Is(err, domain.ErrStockShortage):
		return DomainErrorDescriptor{Status: http.StatusConflict, Code: "insufficient_stock", Message: "insufficient stock"}
	case errors.Is(err, domain.ErrUnavailable):
		return DomainErrorDescriptor{Status: http.StatusConflict, Code: "product_unavailable", Message: "product unavailable"}
	case errors.Is(err, domain.ErrFeatureDisabled):
		return DomainErrorDescriptor{Status: http.StatusServiceUnavailable, Code: "feature_disabled", Message: "feature is disabled"}
	case errors.Is(err, domain.ErrProviderFailed):
		return DomainErrorDescriptor{Status: http.StatusServiceUnavailable, Code: "provider_unavailable", Message: "service is temporarily unavailable"}
	case errors.Is(err, domain.ErrNotFound):
		return DomainErrorDescriptor{Status: http.StatusNotFound, Code: "not_found", Message: "resource not found"}
	case errors.Is(err, domain.ErrConflict):
		return DomainErrorDescriptor{Status: http.StatusConflict, Code: "conflict", Message: "resource conflict"}
	case errors.Is(err, domain.ErrTokenReused):
		return DomainErrorDescriptor{Status: http.StatusUnauthorized, Code: "refresh_token_reused", Message: "refresh token already used"}
	case errors.Is(err, domain.ErrSessionClosed):
		return DomainErrorDescriptor{Status: http.StatusUnauthorized, Code: "session_closed", Message: "session is closed"}
	default:
		return DomainErrorDescriptor{Status: http.StatusInternalServerError, Code: "internal_error", Message: "internal server error"}
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

func FromDomainError(w http.ResponseWriter, err error) {
	descriptor := DescribeDomainError(err)
	Error(w, descriptor.Status, descriptor.Code, descriptor.Message, descriptor.Details)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
