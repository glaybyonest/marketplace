package handlers

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/http/response"
	"marketplace-backend/internal/observability"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

func decodeAndValidate(r *http.Request, dst any, validate *validator.Validate) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return domain.ErrInvalidInput
	}
	if decoder.More() {
		return domain.ErrInvalidInput
	}
	if err := validate.Struct(dst); err != nil {
		return domain.ErrInvalidInput
	}
	return nil
}

func writeDomainError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}
	descriptor := response.DescribeDomainError(err)
	if descriptor.Status >= http.StatusInternalServerError {
		if reporter, ok := observability.ErrorReporterFromContext(r.Context()); ok && reporter != nil {
			reporter.Capture(r.Context(), observability.ErrorEvent{
				Severity: "error",
				Code:     descriptor.Code,
				Message:  err.Error(),
				Method:   r.Method,
				Path:     r.URL.Path,
				Route:    handlerRoutePattern(r),
				Status:   descriptor.Status,
			})
		}
	}
	response.Error(w, descriptor.Status, descriptor.Code, descriptor.Message, descriptor.Details)
}

func getClientIP(r *http.Request) string {
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	if errors.Is(err, net.ErrClosed) {
		return ""
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func handlerRoutePattern(r *http.Request) string {
	if routeCtx := chi.RouteContext(r.Context()); routeCtx != nil {
		if pattern := routeCtx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	if r.URL.Path == "" {
		return "/"
	}
	return r.URL.Path
}
