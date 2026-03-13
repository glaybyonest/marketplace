package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	UserRoleCustomer UserRole = "customer"
	UserRoleAdmin    UserRole = "admin"
)

// User is a registered account.
type User struct {
	ID                  uuid.UUID  `json:"id"`
	Email               string     `json:"email"`
	PasswordHash        string     `json:"-"`
	FullName            string     `json:"full_name,omitempty"`
	Role                UserRole   `json:"role"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	IsActive            bool       `json:"is_active"`
	EmailVerifiedAt     *time.Time `json:"email_verified_at,omitempty"`
	IsEmailVerified     bool       `json:"is_email_verified"`
	FailedLoginAttempts int        `json:"-"`
	LastFailedLoginAt   *time.Time `json:"-"`
	LockedUntil         *time.Time `json:"-"`
}

// UserSession stores refresh token metadata.
type UserSession struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	RefreshTokenHash string     `json:"-"`
	UserAgent        string     `json:"user_agent,omitempty"`
	IP               string     `json:"ip,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	LastSeenAt       time.Time  `json:"last_seen_at"`
	ExpiresAt        time.Time  `json:"expires_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	RotatedAt        *time.Time `json:"rotated_at,omitempty"`
	IsCurrent        bool       `json:"is_current,omitempty"`
}

// TokenPair is returned after authentication operations.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type AuthResult struct {
	User                      User       `json:"user"`
	Tokens                    *TokenPair `json:"tokens,omitempty"`
	RequiresEmailVerification bool       `json:"requires_email_verification"`
	Message                   string     `json:"message,omitempty"`
}

type AuthActionPurpose string

const (
	AuthActionVerifyEmail   AuthActionPurpose = "verify_email"
	AuthActionResetPassword AuthActionPurpose = "reset_password"
)

type AuthActionToken struct {
	ID         uuid.UUID         `json:"id"`
	UserID     uuid.UUID         `json:"user_id"`
	Purpose    AuthActionPurpose `json:"purpose"`
	TokenHash  string            `json:"-"`
	ExpiresAt  time.Time         `json:"expires_at"`
	CreatedAt  time.Time         `json:"created_at"`
	ConsumedAt *time.Time        `json:"consumed_at,omitempty"`
}
