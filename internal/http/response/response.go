package response

import (
	"encoding/json"
	"errors"
	"net/http"

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

func FromDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		Error(w, http.StatusBadRequest, "invalid_input", "invalid input", nil)
	case errors.Is(err, domain.ErrCartEmpty):
		Error(w, http.StatusBadRequest, "cart_empty", "cart is empty", nil)
	case errors.Is(err, domain.ErrUnauthorized):
		Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized", nil)
	case errors.Is(err, domain.ErrInactiveUser):
		Error(w, http.StatusForbidden, "inactive_user", "user is inactive", nil)
	case errors.Is(err, domain.ErrForbidden):
		Error(w, http.StatusForbidden, "forbidden", "forbidden", nil)
	case errors.Is(err, domain.ErrStockShortage):
		Error(w, http.StatusConflict, "insufficient_stock", "insufficient stock", nil)
	case errors.Is(err, domain.ErrUnavailable):
		Error(w, http.StatusConflict, "product_unavailable", "product unavailable", nil)
	case errors.Is(err, domain.ErrNotFound):
		Error(w, http.StatusNotFound, "not_found", "resource not found", nil)
	case errors.Is(err, domain.ErrConflict):
		Error(w, http.StatusConflict, "conflict", "resource conflict", nil)
	case errors.Is(err, domain.ErrTokenReused):
		Error(w, http.StatusUnauthorized, "refresh_token_reused", "refresh token already used", nil)
	case errors.Is(err, domain.ErrSessionClosed):
		Error(w, http.StatusUnauthorized, "session_closed", "session is closed", nil)
	default:
		Error(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
