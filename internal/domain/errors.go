package domain

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrInvalidInput  = errors.New("invalid input")
	ErrInactiveUser  = errors.New("inactive user")
	ErrTokenReused   = errors.New("refresh token already used")
	ErrSessionClosed = errors.New("session is closed")
	ErrCartEmpty     = errors.New("cart is empty")
	ErrStockShortage = errors.New("insufficient stock")
	ErrUnavailable   = errors.New("product unavailable")
)
