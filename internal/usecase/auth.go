package usecase

import (
	"context"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/mailer"
	"marketplace-backend/internal/observability"
	"marketplace-backend/internal/security"

	"github.com/google/uuid"
)

type AuthUserRepository interface {
	Create(ctx context.Context, input CreateUserInput) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error
	MarkEmailVerified(ctx context.Context, id uuid.UUID, verifiedAt time.Time) (domain.User, error)
	RegisterFailedLogin(ctx context.Context, id uuid.UUID, failedAt time.Time, window time.Duration, maxAttempts int, lockoutDuration time.Duration) (domain.User, error)
	ClearFailedLogin(ctx context.Context, id uuid.UUID) error
}

type AuthSessionRepository interface {
	Create(ctx context.Context, input CreateSessionInput) (domain.UserSession, error)
	GetByRefreshTokenHash(ctx context.Context, tokenHash string) (domain.UserSession, error)
	Rotate(ctx context.Context, oldSessionID uuid.UUID, oldTokenHash string, rotatedAt time.Time, newSession CreateSessionInput) error
	RevokeByRefreshTokenHash(ctx context.Context, userID uuid.UUID, tokenHash string, revokedAt time.Time) (bool, error)
	RevokeAllByUserID(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error
}

type AuthActionTokenRepository interface {
	Create(ctx context.Context, token domain.AuthActionToken) (domain.AuthActionToken, error)
	GetActiveByHash(ctx context.Context, purpose domain.AuthActionPurpose, tokenHash string, now time.Time) (domain.AuthActionToken, error)
	Consume(ctx context.Context, id uuid.UUID, consumedAt time.Time) (bool, error)
	DeleteActiveByUserAndPurpose(ctx context.Context, userID uuid.UUID, purpose domain.AuthActionPurpose) error
}

type JWTProvider interface {
	Generate(userID uuid.UUID, email string, role domain.UserRole) (token string, expiresAt time.Time, err error)
}

type PasswordProvider interface {
	Hash(password string) (string, error)
	Compare(hash, password string) bool
}

type Mailer interface {
	Send(ctx context.Context, message mailer.Message) error
}

type AuthAuditLogger interface {
	Record(ctx context.Context, entry observability.AuditEntry) error
}

type CreateUserInput struct {
	Email           string
	PasswordHash    string
	FullName        *string
	Role            domain.UserRole
	EmailVerifiedAt *time.Time
}

type CreateSessionInput struct {
	UserID           uuid.UUID
	RefreshTokenHash string
	UserAgent        string
	IP               string
	ExpiresAt        time.Time
}

type RegisterInput struct {
	Email     string
	Password  string
	FullName  string
	UserAgent string
	IP        string
}

type LoginInput struct {
	Email     string
	Password  string
	UserAgent string
	IP        string
}

type RefreshInput struct {
	RefreshToken string
	UserAgent    string
	IP           string
}

type LogoutInput struct {
	UserID       uuid.UUID
	RefreshToken string
}

type VerifyEmailRequestInput struct {
	Email string
}

type VerifyEmailConfirmInput struct {
	Token string
}

type PasswordResetRequestInput struct {
	Email string
}

type PasswordResetConfirmInput struct {
	Token       string
	NewPassword string
}

type AuthService struct {
	users                AuthUserRepository
	sessions             AuthSessionRepository
	actionTokens         AuthActionTokenRepository
	jwt                  JWTProvider
	passwords            PasswordProvider
	mailer               Mailer
	audit                AuthAuditLogger
	refreshTTL           time.Duration
	emailVerifyTTL       time.Duration
	passwordResetTTL     time.Duration
	loginFailureWindow   time.Duration
	loginMaxAttempts     int
	loginLockoutDuration time.Duration
	appBaseURL           string
	mailFrom             string
	adminEmails          map[string]struct{}
	now                  func() time.Time
}

func NewAuthService(
	users AuthUserRepository,
	sessions AuthSessionRepository,
	actionTokens AuthActionTokenRepository,
	jwt JWTProvider,
	passwords PasswordProvider,
	mailer Mailer,
	audit AuthAuditLogger,
	appBaseURL string,
	mailFrom string,
	adminEmails []string,
	refreshTTL time.Duration,
	emailVerifyTTL time.Duration,
	passwordResetTTL time.Duration,
	loginFailureWindow time.Duration,
	loginMaxAttempts int,
	loginLockoutDuration time.Duration,
) *AuthService {
	normalizedAdminEmails := make(map[string]struct{}, len(adminEmails))
	for _, email := range adminEmails {
		normalized := strings.ToLower(strings.TrimSpace(email))
		if normalized == "" {
			continue
		}
		normalizedAdminEmails[normalized] = struct{}{}
	}

	return &AuthService{
		users:                users,
		sessions:             sessions,
		actionTokens:         actionTokens,
		jwt:                  jwt,
		passwords:            passwords,
		mailer:               mailer,
		audit:                audit,
		refreshTTL:           refreshTTL,
		emailVerifyTTL:       emailVerifyTTL,
		passwordResetTTL:     passwordResetTTL,
		loginFailureWindow:   loginFailureWindow,
		loginMaxAttempts:     loginMaxAttempts,
		loginLockoutDuration: loginLockoutDuration,
		appBaseURL:           strings.TrimRight(strings.TrimSpace(appBaseURL), "/"),
		mailFrom:             strings.TrimSpace(mailFrom),
		adminEmails:          normalizedAdminEmails,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (domain.AuthResult, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return domain.AuthResult{}, err
	}
	if !isStrongPassword(input.Password) {
		return domain.AuthResult{}, domain.ErrInvalidInput
	}

	fullName := normalizeOptionalString(input.FullName)
	passwordHash, err := s.passwords.Hash(input.Password)
	if err != nil {
		return domain.AuthResult{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.Create(ctx, CreateUserInput{
		Email:           email,
		PasswordHash:    passwordHash,
		FullName:        fullName,
		Role:            s.resolveRole(email),
		EmailVerifiedAt: nil,
	})
	if err != nil {
		return domain.AuthResult{}, err
	}

	if err := s.issueEmailVerification(ctx, user); err != nil {
		return domain.AuthResult{}, err
	}
	s.recordAudit(ctx, observability.AuditEntry{
		ActorUserID: ptrUUID(user.ID),
		Action:      "auth.register",
		EntityType:  "user",
		EntityID:    ptrUUID(user.ID),
		Metadata: map[string]any{
			"email": user.Email,
			"role":  string(user.Role),
		},
	})

	return domain.AuthResult{
		User:                      user,
		RequiresEmailVerification: true,
		Message:                   "verification email sent",
	}, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (domain.AuthResult, error) {
	now := s.now()
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return domain.AuthResult{}, domain.ErrUnauthorized
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if err == domain.ErrNotFound {
			s.recordAudit(ctx, observability.AuditEntry{
				Action:     "auth.login_failed",
				EntityType: "user",
				Metadata: map[string]any{
					"email":  email,
					"reason": "not_found",
				},
			})
			return domain.AuthResult{}, domain.ErrUnauthorized
		}
		return domain.AuthResult{}, err
	}
	if !user.IsActive {
		s.recordAudit(ctx, observability.AuditEntry{
			ActorUserID: ptrUUID(user.ID),
			Action:      "auth.login_failed",
			EntityType:  "user",
			EntityID:    ptrUUID(user.ID),
			Metadata: map[string]any{
				"email":  user.Email,
				"reason": "inactive_user",
			},
		})
		return domain.AuthResult{}, domain.ErrInactiveUser
	}
	if user.LockedUntil != nil && user.LockedUntil.After(now) {
		lockErr := &domain.LoginLockedError{
			LockedUntil: *user.LockedUntil,
			RetryAfter:  user.LockedUntil.Sub(now),
		}
		s.recordAudit(ctx, observability.AuditEntry{
			ActorUserID: ptrUUID(user.ID),
			Action:      "auth.login_locked",
			EntityType:  "user",
			EntityID:    ptrUUID(user.ID),
			Metadata: map[string]any{
				"email":               user.Email,
				"locked_until":        user.LockedUntil.Format(time.RFC3339),
				"retry_after_seconds": retryAfterSeconds(lockErr.RetryAfter),
			},
		})
		return domain.AuthResult{}, lockErr
	}
	if !user.IsEmailVerified {
		s.recordAudit(ctx, observability.AuditEntry{
			ActorUserID: ptrUUID(user.ID),
			Action:      "auth.login_failed",
			EntityType:  "user",
			EntityID:    ptrUUID(user.ID),
			Metadata: map[string]any{
				"email":  user.Email,
				"reason": "email_not_verified",
			},
		})
		return domain.AuthResult{}, domain.ErrEmailNotVerified
	}
	if !s.passwords.Compare(user.PasswordHash, input.Password) {
		user, err = s.users.RegisterFailedLogin(ctx, user.ID, now, s.loginFailureWindow, s.loginMaxAttempts, s.loginLockoutDuration)
		if err != nil {
			return domain.AuthResult{}, err
		}
		s.recordAudit(ctx, observability.AuditEntry{
			ActorUserID: ptrUUID(user.ID),
			Action:      "auth.login_failed",
			EntityType:  "user",
			EntityID:    ptrUUID(user.ID),
			Metadata: map[string]any{
				"email":                 user.Email,
				"reason":                "invalid_password",
				"failed_login_attempts": user.FailedLoginAttempts,
			},
		})
		if user.LockedUntil != nil && user.LockedUntil.After(now) {
			lockErr := &domain.LoginLockedError{
				LockedUntil: *user.LockedUntil,
				RetryAfter:  user.LockedUntil.Sub(now),
			}
			s.recordAudit(ctx, observability.AuditEntry{
				ActorUserID: ptrUUID(user.ID),
				Action:      "auth.login_lockout_triggered",
				EntityType:  "user",
				EntityID:    ptrUUID(user.ID),
				Metadata: map[string]any{
					"email":                 user.Email,
					"failed_login_attempts": user.FailedLoginAttempts,
					"locked_until":          user.LockedUntil.Format(time.RFC3339),
					"retry_after_seconds":   retryAfterSeconds(lockErr.RetryAfter),
				},
			})
			return domain.AuthResult{}, lockErr
		}
		return domain.AuthResult{}, domain.ErrUnauthorized
	}
	if user.FailedLoginAttempts > 0 || user.LastFailedLoginAt != nil || user.LockedUntil != nil {
		if err := s.users.ClearFailedLogin(ctx, user.ID); err != nil {
			return domain.AuthResult{}, err
		}
	}

	tokens, err := s.issueTokens(ctx, user, input.UserAgent, input.IP)
	if err != nil {
		return domain.AuthResult{}, err
	}
	s.recordAudit(ctx, observability.AuditEntry{
		ActorUserID: ptrUUID(user.ID),
		Action:      "auth.login",
		EntityType:  "user",
		EntityID:    ptrUUID(user.ID),
		Metadata: map[string]any{
			"email": user.Email,
		},
	})

	return domain.AuthResult{
		User:   user,
		Tokens: tokens,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, input RefreshInput) (domain.TokenPair, error) {
	refreshToken := strings.TrimSpace(input.RefreshToken)
	if refreshToken == "" {
		return domain.TokenPair{}, domain.ErrUnauthorized
	}

	now := s.now()
	tokenHash := security.HashToken(refreshToken)

	session, err := s.sessions.GetByRefreshTokenHash(ctx, tokenHash)
	if err != nil {
		if err == domain.ErrNotFound {
			s.recordAudit(ctx, observability.AuditEntry{
				Action:     "auth.refresh_failed",
				EntityType: "session",
				Metadata: map[string]any{
					"reason": "not_found",
				},
			})
			return domain.TokenPair{}, domain.ErrUnauthorized
		}
		return domain.TokenPair{}, err
	}
	if session.RevokedAt != nil {
		s.recordAudit(ctx, observability.AuditEntry{
			ActorUserID: ptrUUID(session.UserID),
			Action:      "auth.refresh_failed",
			EntityType:  "session",
			EntityID:    ptrUUID(session.ID),
			Metadata: map[string]any{
				"reason": "revoked",
			},
		})
		return domain.TokenPair{}, domain.ErrSessionClosed
	}
	if session.RotatedAt != nil {
		s.recordAudit(ctx, observability.AuditEntry{
			ActorUserID: ptrUUID(session.UserID),
			Action:      "auth.refresh_failed",
			EntityType:  "session",
			EntityID:    ptrUUID(session.ID),
			Metadata: map[string]any{
				"reason": "rotated",
			},
		})
		return domain.TokenPair{}, domain.ErrTokenReused
	}
	if now.After(session.ExpiresAt) {
		_, _ = s.sessions.RevokeByRefreshTokenHash(ctx, session.UserID, tokenHash, now)
		s.recordAudit(ctx, observability.AuditEntry{
			ActorUserID: ptrUUID(session.UserID),
			Action:      "auth.refresh_failed",
			EntityType:  "session",
			EntityID:    ptrUUID(session.ID),
			Metadata: map[string]any{
				"reason": "expired",
			},
		})
		return domain.TokenPair{}, domain.ErrUnauthorized
	}

	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil {
		if err == domain.ErrNotFound {
			s.recordAudit(ctx, observability.AuditEntry{
				ActorUserID: ptrUUID(session.UserID),
				Action:      "auth.refresh_failed",
				EntityType:  "session",
				EntityID:    ptrUUID(session.ID),
				Metadata: map[string]any{
					"reason": "user_not_found",
				},
			})
			return domain.TokenPair{}, domain.ErrUnauthorized
		}
		return domain.TokenPair{}, err
	}
	if !user.IsActive {
		s.recordAudit(ctx, observability.AuditEntry{
			ActorUserID: ptrUUID(user.ID),
			Action:      "auth.refresh_failed",
			EntityType:  "user",
			EntityID:    ptrUUID(user.ID),
			Metadata: map[string]any{
				"reason": "inactive_user",
				"email":  user.Email,
			},
		})
		return domain.TokenPair{}, domain.ErrInactiveUser
	}
	if !user.IsEmailVerified {
		s.recordAudit(ctx, observability.AuditEntry{
			ActorUserID: ptrUUID(user.ID),
			Action:      "auth.refresh_failed",
			EntityType:  "user",
			EntityID:    ptrUUID(user.ID),
			Metadata: map[string]any{
				"reason": "email_not_verified",
				"email":  user.Email,
			},
		})
		return domain.TokenPair{}, domain.ErrEmailNotVerified
	}

	newRefreshToken, err := security.GenerateRefreshToken()
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("generate refresh token: %w", err)
	}

	if err := s.sessions.Rotate(ctx, session.ID, tokenHash, now, CreateSessionInput{
		UserID:           user.ID,
		RefreshTokenHash: security.HashToken(newRefreshToken),
		UserAgent:        normalizeUserAgent(input.UserAgent),
		IP:               normalizeIP(input.IP),
		ExpiresAt:        now.Add(s.refreshTTL),
	}); err != nil {
		return domain.TokenPair{}, err
	}

	accessToken, accessExp, err := s.jwt.Generate(user.ID, user.Email, user.Role)
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("generate access token: %w", err)
	}
	s.recordAudit(ctx, observability.AuditEntry{
		ActorUserID: ptrUUID(user.ID),
		Action:      "auth.refresh",
		EntityType:  "session",
		Metadata: map[string]any{
			"session_id": session.ID.String(),
		},
	})

	return domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(accessExp.Sub(now).Seconds()),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, input LogoutInput) error {
	if input.UserID == uuid.Nil || strings.TrimSpace(input.RefreshToken) == "" {
		s.recordAudit(ctx, observability.AuditEntry{
			Action:     "auth.logout_failed",
			EntityType: "session",
			Metadata: map[string]any{
				"reason": "invalid_input",
			},
		})
		return domain.ErrUnauthorized
	}

	revoked, err := s.sessions.RevokeByRefreshTokenHash(
		ctx,
		input.UserID,
		security.HashToken(strings.TrimSpace(input.RefreshToken)),
		s.now(),
	)
	if err != nil {
		return err
	}
	if !revoked {
		s.recordAudit(ctx, observability.AuditEntry{
			ActorUserID: ptrUUID(input.UserID),
			Action:      "auth.logout_failed",
			EntityType:  "session",
			Metadata: map[string]any{
				"reason": "refresh_token_not_found",
			},
		})
		return domain.ErrUnauthorized
	}
	s.recordAudit(ctx, observability.AuditEntry{
		ActorUserID: ptrUUID(input.UserID),
		Action:      "auth.logout",
		EntityType:  "session",
		Metadata: map[string]any{
			"refresh_token_revoked": true,
		},
	})
	return nil
}

func (s *AuthService) Me(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	if userID == uuid.Nil {
		return domain.User{}, domain.ErrUnauthorized
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	if !user.IsActive {
		return domain.User{}, domain.ErrInactiveUser
	}
	return user, nil
}

func (s *AuthService) RequestEmailVerification(ctx context.Context, input VerifyEmailRequestInput) error {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return err
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil
		}
		return err
	}
	if !user.IsActive || user.IsEmailVerified {
		return nil
	}

	if err := s.issueEmailVerification(ctx, user); err != nil {
		return err
	}
	s.recordAudit(ctx, observability.AuditEntry{
		ActorUserID: ptrUUID(user.ID),
		Action:      "auth.verify_email_requested",
		EntityType:  "user",
		EntityID:    ptrUUID(user.ID),
		Metadata:    map[string]any{"email": user.Email},
	})
	return nil
}

func (s *AuthService) ConfirmEmailVerification(ctx context.Context, input VerifyEmailConfirmInput) (domain.User, error) {
	token, err := normalizeActionToken(input.Token)
	if err != nil {
		return domain.User{}, err
	}

	actionToken, err := s.actionTokens.GetActiveByHash(ctx, domain.AuthActionVerifyEmail, security.HashToken(token), s.now())
	if err != nil {
		if err == domain.ErrNotFound {
			s.recordAudit(ctx, observability.AuditEntry{
				Action:     "auth.verify_email_failed",
				EntityType: "user",
				Metadata: map[string]any{
					"reason": "invalid_token",
				},
			})
			return domain.User{}, domain.ErrInvalidToken
		}
		return domain.User{}, err
	}

	consumed, err := s.actionTokens.Consume(ctx, actionToken.ID, s.now())
	if err != nil {
		return domain.User{}, err
	}
	if !consumed {
		s.recordAudit(ctx, observability.AuditEntry{
			ActorUserID: ptrUUID(actionToken.UserID),
			Action:      "auth.verify_email_failed",
			EntityType:  "user",
			EntityID:    ptrUUID(actionToken.UserID),
			Metadata: map[string]any{
				"reason": "already_consumed",
			},
		})
		return domain.User{}, domain.ErrInvalidToken
	}

	user, err := s.users.GetByID(ctx, actionToken.UserID)
	if err != nil {
		return domain.User{}, err
	}
	if user.IsEmailVerified {
		return user, nil
	}

	verifiedUser, err := s.users.MarkEmailVerified(ctx, user.ID, s.now())
	if err != nil {
		return domain.User{}, err
	}
	s.recordAudit(ctx, observability.AuditEntry{
		ActorUserID: ptrUUID(user.ID),
		Action:      "auth.email_verified",
		EntityType:  "user",
		EntityID:    ptrUUID(user.ID),
		Metadata:    map[string]any{"email": user.Email},
	})
	return verifiedUser, nil
}

func (s *AuthService) RequestPasswordReset(ctx context.Context, input PasswordResetRequestInput) error {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return err
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil
		}
		return err
	}
	if !user.IsActive {
		return nil
	}

	if err := s.issuePasswordReset(ctx, user); err != nil {
		return err
	}
	s.recordAudit(ctx, observability.AuditEntry{
		ActorUserID: ptrUUID(user.ID),
		Action:      "auth.password_reset_requested",
		EntityType:  "user",
		EntityID:    ptrUUID(user.ID),
		Metadata:    map[string]any{"email": user.Email},
	})
	return nil
}

func (s *AuthService) ConfirmPasswordReset(ctx context.Context, input PasswordResetConfirmInput) error {
	token, err := normalizeActionToken(input.Token)
	if err != nil {
		return err
	}
	if !isStrongPassword(input.NewPassword) {
		return domain.ErrInvalidInput
	}

	actionToken, err := s.actionTokens.GetActiveByHash(ctx, domain.AuthActionResetPassword, security.HashToken(token), s.now())
	if err != nil {
		if err == domain.ErrNotFound {
			s.recordAudit(ctx, observability.AuditEntry{
				Action:     "auth.password_reset_failed",
				EntityType: "user",
				Metadata: map[string]any{
					"reason": "invalid_token",
				},
			})
			return domain.ErrInvalidToken
		}
		return err
	}

	passwordHash, err := s.passwords.Hash(input.NewPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.users.UpdatePasswordHash(ctx, actionToken.UserID, passwordHash); err != nil {
		return err
	}
	if err := s.users.ClearFailedLogin(ctx, actionToken.UserID); err != nil {
		return err
	}
	if err := s.sessions.RevokeAllByUserID(ctx, actionToken.UserID, s.now()); err != nil {
		return err
	}

	consumed, err := s.actionTokens.Consume(ctx, actionToken.ID, s.now())
	if err != nil {
		return err
	}
	if !consumed {
		s.recordAudit(ctx, observability.AuditEntry{
			ActorUserID: ptrUUID(actionToken.UserID),
			Action:      "auth.password_reset_failed",
			EntityType:  "user",
			EntityID:    ptrUUID(actionToken.UserID),
			Metadata: map[string]any{
				"reason": "already_consumed",
			},
		})
		return domain.ErrInvalidToken
	}
	s.recordAudit(ctx, observability.AuditEntry{
		ActorUserID: ptrUUID(actionToken.UserID),
		Action:      "auth.password_reset_completed",
		EntityType:  "user",
		EntityID:    ptrUUID(actionToken.UserID),
	})
	return nil
}

func (s *AuthService) issueTokens(ctx context.Context, user domain.User, userAgent, ip string) (*domain.TokenPair, error) {
	now := s.now()

	accessToken, accessExp, err := s.jwt.Generate(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := security.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	_, err = s.sessions.Create(ctx, CreateSessionInput{
		UserID:           user.ID,
		RefreshTokenHash: security.HashToken(refreshToken),
		UserAgent:        normalizeUserAgent(userAgent),
		IP:               normalizeIP(ip),
		ExpiresAt:        now.Add(s.refreshTTL),
	})
	if err != nil {
		return nil, err
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(accessExp.Sub(now).Seconds()),
	}, nil
}

func (s *AuthService) resolveRole(email string) domain.UserRole {
	if _, ok := s.adminEmails[email]; ok {
		return domain.UserRoleAdmin
	}
	return domain.UserRoleCustomer
}

func (s *AuthService) recordAudit(ctx context.Context, entry observability.AuditEntry) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Record(ctx, entry)
}

func ptrUUID(value uuid.UUID) *uuid.UUID {
	if value == uuid.Nil {
		return nil
	}
	return &value
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

func (s *AuthService) issueEmailVerification(ctx context.Context, user domain.User) error {
	link, err := s.createActionTokenLink(ctx, user.ID, domain.AuthActionVerifyEmail, s.emailVerifyTTL, "/verify-email")
	if err != nil {
		return err
	}

	fullName := strings.TrimSpace(user.FullName)
	greeting := "Hello"
	if fullName != "" {
		greeting = "Hello, " + fullName
	}

	return s.mailer.Send(ctx, mailer.Message{
		To:      user.Email,
		From:    s.mailFrom,
		Subject: "Verify your email",
		Text:    greeting + "\n\nVerify your email by opening this link:\n" + link + "\n\nIf you did not sign up, ignore this email.",
	})
}

func (s *AuthService) issuePasswordReset(ctx context.Context, user domain.User) error {
	link, err := s.createActionTokenLink(ctx, user.ID, domain.AuthActionResetPassword, s.passwordResetTTL, "/reset-password")
	if err != nil {
		return err
	}

	fullName := strings.TrimSpace(user.FullName)
	greeting := "Hello"
	if fullName != "" {
		greeting = "Hello, " + fullName
	}

	return s.mailer.Send(ctx, mailer.Message{
		To:      user.Email,
		From:    s.mailFrom,
		Subject: "Reset your password",
		Text:    greeting + "\n\nReset your password by opening this link:\n" + link + "\n\nIf you did not request a password reset, ignore this email.",
	})
}

func (s *AuthService) createActionTokenLink(
	ctx context.Context,
	userID uuid.UUID,
	purpose domain.AuthActionPurpose,
	ttl time.Duration,
	path string,
) (string, error) {
	rawToken, err := security.GenerateRefreshToken()
	if err != nil {
		return "", fmt.Errorf("generate action token: %w", err)
	}

	if err := s.actionTokens.DeleteActiveByUserAndPurpose(ctx, userID, purpose); err != nil {
		return "", err
	}

	_, err = s.actionTokens.Create(ctx, domain.AuthActionToken{
		ID:        uuid.New(),
		UserID:    userID,
		Purpose:   purpose,
		TokenHash: security.HashToken(rawToken),
		ExpiresAt: s.now().Add(ttl),
	})
	if err != nil {
		return "", err
	}

	return s.appBaseURL + path + "?token=" + url.QueryEscape(rawToken), nil
}

func normalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return "", domain.ErrInvalidInput
	}
	if _, err := mail.ParseAddress(normalized); err != nil {
		return "", domain.ErrInvalidInput
	}
	return normalized, nil
}

func normalizeActionToken(token string) (string, error) {
	normalized := strings.TrimSpace(token)
	if normalized == "" {
		return "", domain.ErrInvalidInput
	}
	return normalized, nil
}

func isStrongPassword(password string) bool {
	if len(password) < 8 || len(password) > 72 {
		return false
	}

	hasLetter := false
	hasDigit := false
	for _, r := range password {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			hasLetter = true
		}
	}
	return hasLetter && hasDigit
}

func normalizeOptionalString(value string) *string {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil
	}
	return &v
}

func normalizeUserAgent(ua string) string {
	return strings.TrimSpace(ua)
}

func normalizeIP(ip string) string {
	return strings.TrimSpace(ip)
}
