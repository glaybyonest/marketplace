package domain

import (
	"errors"
	"time"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("conflict")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrForbidden        = errors.New("forbidden")
	ErrInvalidInput     = errors.New("invalid input")
	ErrInactiveUser     = errors.New("inactive user")
	ErrEmailNotVerified = errors.New("email not verified")
	ErrInvalidToken     = errors.New("invalid token")
	ErrTokenReused      = errors.New("refresh token already used")
	ErrSessionClosed    = errors.New("session is closed")
	ErrRateLimited      = errors.New("rate limited")
	ErrLoginLocked      = errors.New("login locked")
	ErrCartEmpty        = errors.New("cart is empty")
	ErrStockShortage    = errors.New("insufficient stock")
	ErrUnavailable      = errors.New("product unavailable")
)

type RateLimitError struct {
	Scope      string
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return ErrRateLimited.Error()
}

func (e *RateLimitError) Unwrap() error {
	return ErrRateLimited
}

type LoginLockedError struct {
	LockedUntil time.Time
	RetryAfter  time.Duration
}

func (e *LoginLockedError) Error() string {
	return ErrLoginLocked.Error()
}

func (e *LoginLockedError) Unwrap() error {
	return ErrLoginLocked
}
