package usecase

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
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
	GetByPhone(ctx context.Context, phone string) (domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error
	MarkEmailVerified(ctx context.Context, id uuid.UUID, verifiedAt time.Time) (domain.User, error)
	UpdatePhone(ctx context.Context, id uuid.UUID, phone *string) (domain.User, error)
	RegisterFailedLogin(ctx context.Context, id uuid.UUID, failedAt time.Time, window time.Duration, maxAttempts int, lockoutDuration time.Duration) (domain.User, error)
	ClearFailedLogin(ctx context.Context, id uuid.UUID) error
}

type AuthSessionRepository interface {
	Create(ctx context.Context, input CreateSessionInput) (domain.UserSession, error)
	GetByRefreshTokenHash(ctx context.Context, tokenHash string) (domain.UserSession, error)
	Rotate(ctx context.Context, oldSessionID uuid.UUID, oldTokenHash string, rotatedAt time.Time, newSession CreateSessionInput) (domain.UserSession, error)
	RevokeByRefreshTokenHash(ctx context.Context, userID uuid.UUID, tokenHash string, revokedAt time.Time) (bool, error)
	RevokeAllByUserID(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error
	ListActiveByUserID(ctx context.Context, userID uuid.UUID, now time.Time) ([]domain.UserSession, error)
	RevokeByID(ctx context.Context, userID, sessionID uuid.UUID, revokedAt time.Time) (bool, error)
}

type AuthActionTokenRepository interface {
	Create(ctx context.Context, token domain.AuthActionToken) (domain.AuthActionToken, error)
	GetActiveByHash(ctx context.Context, purpose domain.AuthActionPurpose, tokenHash string, now time.Time) (domain.AuthActionToken, error)
	GetLatestActiveByUserAndPurpose(ctx context.Context, userID uuid.UUID, purpose domain.AuthActionPurpose, now time.Time) (domain.AuthActionToken, error)
	Consume(ctx context.Context, id uuid.UUID, consumedAt time.Time) (bool, error)
	DeleteActiveByUserAndPurpose(ctx context.Context, userID uuid.UUID, purpose domain.AuthActionPurpose) error
}

type JWTProvider interface {
	Generate(userID uuid.UUID, email string, role domain.UserRole, sessionID uuid.UUID) (token string, expiresAt time.Time, err error)
}

type PasswordProvider interface {
	Hash(password string) (string, error)
	Compare(hash, password string) bool
}

type Mailer interface {
	Send(ctx context.Context, message mailer.Message) error
}

type SMSProvider interface {
	Send(ctx context.Context, message mailer.SMSMessage) error
}

type AuthAuditLogger interface {
	Record(ctx context.Context, entry observability.AuditEntry) error
}

type CreateUserInput struct {
	Email           string
	Phone           *string
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
	Phone     string
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

type RevokeSessionInput struct {
	UserID        uuid.UUID
	TargetSession uuid.UUID
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

type EmailCodeRequestInput struct {
	Email string
}

type EmailCodeLoginInput struct {
	Email     string
	Code      string
	UserAgent string
	IP        string
}

type PhoneCodeRequestInput struct {
	Phone string
}

type PhoneCodeLoginInput struct {
	Phone     string
	Code      string
	UserAgent string
	IP        string
}

type AuthService struct {
	users                AuthUserRepository
	sessions             AuthSessionRepository
	actionTokens         AuthActionTokenRepository
	jwt                  JWTProvider
	passwords            PasswordProvider
	mailer               Mailer
	sms                  SMSProvider
	audit                AuthAuditLogger
	refreshTTL           time.Duration
	emailVerifyTTL       time.Duration
	passwordResetTTL     time.Duration
	loginCodeTTL         time.Duration
	loginFailureWindow   time.Duration
	loginMaxAttempts     int
	loginLockoutDuration time.Duration
	appBaseURL           string
	mailFrom             string
	adminEmails          map[string]struct{}
	exposeDevLoginCodes  bool
	now                  func() time.Time
}

func NewAuthService(
	users AuthUserRepository,
	sessions AuthSessionRepository,
	actionTokens AuthActionTokenRepository,
	jwt JWTProvider,
	passwords PasswordProvider,
	mailer Mailer,
	sms SMSProvider,
	audit AuthAuditLogger,
	appBaseURL string,
	mailFrom string,
	adminEmails []string,
	refreshTTL time.Duration,
	emailVerifyTTL time.Duration,
	passwordResetTTL time.Duration,
	loginCodeTTL time.Duration,
	loginFailureWindow time.Duration,
	loginMaxAttempts int,
	loginLockoutDuration time.Duration,
	exposeDevLoginCodes bool,
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
		sms:                  sms,
		audit:                audit,
		refreshTTL:           refreshTTL,
		emailVerifyTTL:       emailVerifyTTL,
		passwordResetTTL:     passwordResetTTL,
		loginCodeTTL:         loginCodeTTL,
		loginFailureWindow:   loginFailureWindow,
		loginMaxAttempts:     loginMaxAttempts,
		loginLockoutDuration: loginLockoutDuration,
		appBaseURL:           strings.TrimRight(strings.TrimSpace(appBaseURL), "/"),
		mailFrom:             strings.TrimSpace(mailFrom),
		adminEmails:          normalizedAdminEmails,
		exposeDevLoginCodes:  exposeDevLoginCodes,
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
	phone, err := normalizeOptionalPhone(input.Phone)
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
		Phone:           phone,
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

func (s *AuthService) RequestEmailLoginCode(ctx context.Context, input EmailCodeRequestInput) (domain.AuthCodeDispatch, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return domain.AuthCodeDispatch{}, err
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if err == domain.ErrNotFound {
			return s.genericLoginCodeDispatch(domain.AuthCodeChannelEmail), nil
		}
		return domain.AuthCodeDispatch{}, err
	}
	if !user.IsActive {
		return s.genericLoginCodeDispatch(domain.AuthCodeChannelEmail), nil
	}

	code, err := s.issueLoginCode(ctx, user.ID, domain.AuthActionLoginEmail)
	if err != nil {
		return domain.AuthCodeDispatch{}, err
	}

	fullName := strings.TrimSpace(user.FullName)
	greeting := "Hello"
	if fullName != "" {
		greeting = "Hello, " + fullName
	}

	if err := s.mailer.Send(ctx, mailer.Message{
		To:      user.Email,
		From:    s.mailFrom,
		Subject: "Your login code",
		Text:    greeting + "\n\nUse this code to sign in:\n" + code + "\n\nThe code expires in " + s.loginCodeTTL.Round(time.Minute).String() + ".",
	}); err != nil {
		return domain.AuthCodeDispatch{}, err
	}

	s.recordAudit(ctx, observability.AuditEntry{
		ActorUserID: ptrUUID(user.ID),
		Action:      "auth.login_email_code_requested",
		EntityType:  "user",
		EntityID:    ptrUUID(user.ID),
		Metadata: map[string]any{
			"email": user.Email,
		},
	})

	return s.loginCodeDispatch(domain.AuthCodeChannelEmail, user.Email, code), nil
}

func (s *AuthService) LoginWithEmailCode(ctx context.Context, input EmailCodeLoginInput) (domain.AuthResult, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return domain.AuthResult{}, domain.ErrInvalidLoginCode
	}
	code, err := normalizeLoginCode(input.Code)
	if err != nil {
		return domain.AuthResult{}, domain.ErrInvalidInput
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if err == domain.ErrNotFound {
			return domain.AuthResult{}, domain.ErrInvalidLoginCode
		}
		return domain.AuthResult{}, err
	}
	if !user.IsActive {
		return domain.AuthResult{}, domain.ErrInactiveUser
	}

	token, err := s.actionTokens.GetLatestActiveByUserAndPurpose(ctx, user.ID, domain.AuthActionLoginEmail, s.now())
	if err != nil {
		if err == domain.ErrNotFound {
			s.recordAudit(ctx, observability.AuditEntry{
				ActorUserID: ptrUUID(user.ID),
				Action:      "auth.login_email_code_failed",
				EntityType:  "user",
				EntityID:    ptrUUID(user.ID),
				Metadata: map[string]any{
					"email":  user.Email,
					"reason": "missing_code",
				},
			})
			return domain.AuthResult{}, domain.ErrInvalidLoginCode
		}
		return domain.AuthResult{}, err
	}
	if token.TokenHash != security.HashToken(code) {
		s.recordAudit(ctx, observability.AuditEntry{
			ActorUserID: ptrUUID(user.ID),
			Action:      "auth.login_email_code_failed",
			EntityType:  "user",
			EntityID:    ptrUUID(user.ID),
			Metadata: map[string]any{
				"email":  user.Email,
				"reason": "invalid_code",
			},
		})
		return domain.AuthResult{}, domain.ErrInvalidLoginCode
	}

	consumed, err := s.actionTokens.Consume(ctx, token.ID, s.now())
	if err != nil {
		return domain.AuthResult{}, err
	}
	if !consumed {
		return domain.AuthResult{}, domain.ErrInvalidLoginCode
	}

	if !user.IsEmailVerified {
		user, err = s.users.MarkEmailVerified(ctx, user.ID, s.now())
		if err != nil {
			return domain.AuthResult{}, err
		}
	}
	if err := s.clearFailedLoginIfNeeded(ctx, user); err != nil {
		return domain.AuthResult{}, err
	}

	tokens, err := s.issueTokens(ctx, user, input.UserAgent, input.IP)
	if err != nil {
		return domain.AuthResult{}, err
	}

	s.recordAudit(ctx, observability.AuditEntry{
		ActorUserID: ptrUUID(user.ID),
		Action:      "auth.login_email_code",
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

func (s *AuthService) RequestPhoneLoginCode(ctx context.Context, input PhoneCodeRequestInput) (domain.AuthCodeDispatch, error) {
	phone, err := normalizePhone(input.Phone)
	if err != nil {
		return domain.AuthCodeDispatch{}, err
	}

	user, err := s.users.GetByPhone(ctx, phone)
	if err != nil {
		if err == domain.ErrNotFound {
			return s.genericLoginCodeDispatch(domain.AuthCodeChannelPhone), nil
		}
		return domain.AuthCodeDispatch{}, err
	}
	if !user.IsActive || !user.IsEmailVerified {
		return s.genericLoginCodeDispatch(domain.AuthCodeChannelPhone), nil
	}

	code, err := s.issueLoginCode(ctx, user.ID, domain.AuthActionLoginPhone)
	if err != nil {
		return domain.AuthCodeDispatch{}, err
	}

	if s.sms != nil {
		if err := s.sms.Send(ctx, mailer.SMSMessage{
			To:   phone,
			Text: "Marketplace code: " + code + ". Expires in " + s.loginCodeTTL.Round(time.Minute).String() + ".",
		}); err != nil {
			return domain.AuthCodeDispatch{}, err
		}
	}

	s.recordAudit(ctx, observability.AuditEntry{
		ActorUserID: ptrUUID(user.ID),
		Action:      "auth.login_phone_code_requested",
		EntityType:  "user",
		EntityID:    ptrUUID(user.ID),
		Metadata: map[string]any{
			"phone": phone,
		},
	})

	return s.loginCodeDispatch(domain.AuthCodeChannelPhone, phone, code), nil
}

func (s *AuthService) LoginWithPhoneCode(ctx context.Context, input PhoneCodeLoginInput) (domain.AuthResult, error) {
	phone, err := normalizePhone(input.Phone)
	if err != nil {
		return domain.AuthResult{}, domain.ErrInvalidLoginCode
	}
	code, err := normalizeLoginCode(input.Code)
	if err != nil {
		return domain.AuthResult{}, domain.ErrInvalidInput
	}

	user, err := s.users.GetByPhone(ctx, phone)
	if err != nil {
		if err == domain.ErrNotFound {
			return domain.AuthResult{}, domain.ErrInvalidLoginCode
		}
		return domain.AuthResult{}, err
	}
	if !user.IsActive {
		return domain.AuthResult{}, domain.ErrInactiveUser
	}
	if !user.IsEmailVerified {
		return domain.AuthResult{}, domain.ErrEmailNotVerified
	}

	token, err := s.actionTokens.GetLatestActiveByUserAndPurpose(ctx, user.ID, domain.AuthActionLoginPhone, s.now())
	if err != nil {
		if err == domain.ErrNotFound {
			s.recordAudit(ctx, observability.AuditEntry{
				ActorUserID: ptrUUID(user.ID),
				Action:      "auth.login_phone_code_failed",
				EntityType:  "user",
				EntityID:    ptrUUID(user.ID),
				Metadata: map[string]any{
					"phone":  phone,
					"reason": "missing_code",
				},
			})
			return domain.AuthResult{}, domain.ErrInvalidLoginCode
		}
		return domain.AuthResult{}, err
	}
	if token.TokenHash != security.HashToken(code) {
		s.recordAudit(ctx, observability.AuditEntry{
			ActorUserID: ptrUUID(user.ID),
			Action:      "auth.login_phone_code_failed",
			EntityType:  "user",
			EntityID:    ptrUUID(user.ID),
			Metadata: map[string]any{
				"phone":  phone,
				"reason": "invalid_code",
			},
		})
		return domain.AuthResult{}, domain.ErrInvalidLoginCode
	}

	consumed, err := s.actionTokens.Consume(ctx, token.ID, s.now())
	if err != nil {
		return domain.AuthResult{}, err
	}
	if !consumed {
		return domain.AuthResult{}, domain.ErrInvalidLoginCode
	}

	if err := s.clearFailedLoginIfNeeded(ctx, user); err != nil {
		return domain.AuthResult{}, err
	}

	tokens, err := s.issueTokens(ctx, user, input.UserAgent, input.IP)
	if err != nil {
		return domain.AuthResult{}, err
	}

	s.recordAudit(ctx, observability.AuditEntry{
		ActorUserID: ptrUUID(user.ID),
		Action:      "auth.login_phone_code",
		EntityType:  "user",
		EntityID:    ptrUUID(user.ID),
		Metadata: map[string]any{
			"phone": phone,
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

	nextSession, err := s.sessions.Rotate(ctx, session.ID, tokenHash, now, CreateSessionInput{
		UserID:           user.ID,
		RefreshTokenHash: security.HashToken(newRefreshToken),
		UserAgent:        normalizeUserAgent(input.UserAgent),
		IP:               normalizeIP(input.IP),
		ExpiresAt:        now.Add(s.refreshTTL),
	})
	if err != nil {
		return domain.TokenPair{}, err
	}

	accessToken, accessExp, err := s.jwt.Generate(user.ID, user.Email, user.Role, nextSession.ID)
	if err != nil {
		return domain.TokenPair{}, fmt.Errorf("generate access token: %w", err)
	}
	s.recordAudit(ctx, observability.AuditEntry{
		ActorUserID: ptrUUID(user.ID),
		Action:      "auth.refresh",
		EntityType:  "session",
		Metadata: map[string]any{
			"session_id":      session.ID.String(),
			"next_session_id": nextSession.ID.String(),
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

func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	if userID == uuid.Nil {
		s.recordAudit(ctx, observability.AuditEntry{
			Action:     "auth.logout_all_failed",
			EntityType: "session",
			Metadata: map[string]any{
				"reason": "invalid_user_id",
			},
		})
		return domain.ErrUnauthorized
	}

	if err := s.sessions.RevokeAllByUserID(ctx, userID, s.now()); err != nil {
		return err
	}

	s.recordAudit(ctx, observability.AuditEntry{
		ActorUserID: ptrUUID(userID),
		Action:      "auth.logout_all",
		EntityType:  "session",
		Metadata: map[string]any{
			"all_sessions_revoked": true,
		},
	})
	return nil
}

func (s *AuthService) Sessions(ctx context.Context, userID, currentSessionID uuid.UUID) ([]domain.UserSession, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrUnauthorized
	}

	sessions, err := s.sessions.ListActiveByUserID(ctx, userID, s.now())
	if err != nil {
		return nil, err
	}

	for index := range sessions {
		sessions[index].IsCurrent = currentSessionID != uuid.Nil && sessions[index].ID == currentSessionID
	}

	s.recordAudit(ctx, observability.AuditEntry{
		ActorUserID: ptrUUID(userID),
		Action:      "auth.sessions_listed",
		EntityType:  "session",
		Metadata: map[string]any{
			"count": len(sessions),
		},
	})
	return sessions, nil
}

func (s *AuthService) RevokeSession(ctx context.Context, input RevokeSessionInput) error {
	if input.UserID == uuid.Nil || input.TargetSession == uuid.Nil {
		return domain.ErrUnauthorized
	}

	revoked, err := s.sessions.RevokeByID(ctx, input.UserID, input.TargetSession, s.now())
	if err != nil {
		return err
	}
	if !revoked {
		s.recordAudit(ctx, observability.AuditEntry{
			ActorUserID: ptrUUID(input.UserID),
			Action:      "auth.session_revoke_failed",
			EntityType:  "session",
			EntityID:    ptrUUID(input.TargetSession),
			Metadata: map[string]any{
				"reason": "not_found",
			},
		})
		return domain.ErrUnauthorized
	}

	s.recordAudit(ctx, observability.AuditEntry{
		ActorUserID: ptrUUID(input.UserID),
		Action:      "auth.session_revoked",
		EntityType:  "session",
		EntityID:    ptrUUID(input.TargetSession),
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

	refreshToken, err := security.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	session, err := s.sessions.Create(ctx, CreateSessionInput{
		UserID:           user.ID,
		RefreshTokenHash: security.HashToken(refreshToken),
		UserAgent:        normalizeUserAgent(userAgent),
		IP:               normalizeIP(ip),
		ExpiresAt:        now.Add(s.refreshTTL),
	})
	if err != nil {
		return nil, err
	}

	accessToken, accessExp, err := s.jwt.Generate(user.ID, user.Email, user.Role, session.ID)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
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

func (s *AuthService) issueLoginCode(ctx context.Context, userID uuid.UUID, purpose domain.AuthActionPurpose) (string, error) {
	rawCode, err := generateNumericCode(6)
	if err != nil {
		return "", fmt.Errorf("generate login code: %w", err)
	}

	if err := s.actionTokens.DeleteActiveByUserAndPurpose(ctx, userID, purpose); err != nil {
		return "", err
	}

	_, err = s.actionTokens.Create(ctx, domain.AuthActionToken{
		ID:        uuid.New(),
		UserID:    userID,
		Purpose:   purpose,
		TokenHash: security.HashToken(rawCode),
		ExpiresAt: s.now().Add(s.loginCodeTTL),
	})
	if err != nil {
		return "", err
	}

	return rawCode, nil
}

func (s *AuthService) genericLoginCodeDispatch(channel domain.AuthCodeChannel) domain.AuthCodeDispatch {
	return domain.AuthCodeDispatch{
		Accepted:  true,
		Channel:   channel,
		ExpiresIn: int64(s.loginCodeTTL.Seconds()),
		Message:   "If the account exists, we sent a login code.",
	}
}

func (s *AuthService) loginCodeDispatch(channel domain.AuthCodeChannel, destination string, code string) domain.AuthCodeDispatch {
	result := domain.AuthCodeDispatch{
		Accepted:          true,
		Channel:           channel,
		MaskedDestination: maskAuthDestination(channel, destination),
		ExpiresIn:         int64(s.loginCodeTTL.Seconds()),
		Message:           "Login code sent.",
	}
	if s.exposeDevLoginCodes {
		result.DevCode = code
	}
	return result
}

func (s *AuthService) clearFailedLoginIfNeeded(ctx context.Context, user domain.User) error {
	if user.FailedLoginAttempts == 0 && user.LastFailedLoginAt == nil && user.LockedUntil == nil {
		return nil
	}
	return s.users.ClearFailedLogin(ctx, user.ID)
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

func normalizeLoginCode(code string) (string, error) {
	normalized := strings.TrimSpace(code)
	if len(normalized) != 6 {
		return "", domain.ErrInvalidInput
	}
	for _, r := range normalized {
		if r < '0' || r > '9' {
			return "", domain.ErrInvalidInput
		}
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

func normalizeOptionalPhone(phone string) (*string, error) {
	value := strings.TrimSpace(phone)
	if value == "" {
		return nil, nil
	}

	normalized, err := normalizePhone(value)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}

func normalizePhone(phone string) (string, error) {
	trimmed := strings.TrimSpace(phone)
	if trimmed == "" {
		return "", domain.ErrInvalidInput
	}

	builder := strings.Builder{}
	builder.Grow(len(trimmed))

	for index, r := range trimmed {
		switch {
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '+' && index == 0:
			builder.WriteRune(r)
		}
	}

	normalized := builder.String()
	if normalized == "" || normalized == "+" {
		return "", domain.ErrInvalidInput
	}
	if normalized[0] != '+' {
		normalized = "+" + normalized
	}

	digitCount := 0
	for _, r := range normalized {
		if r >= '0' && r <= '9' {
			digitCount++
		}
	}
	if digitCount < 10 || digitCount > 15 {
		return "", domain.ErrInvalidInput
	}
	return normalized, nil
}

func normalizeUserAgent(ua string) string {
	return strings.TrimSpace(ua)
}

func normalizeIP(ip string) string {
	return strings.TrimSpace(ip)
}

func generateNumericCode(length int) (string, error) {
	if length <= 0 {
		return "", domain.ErrInvalidInput
	}

	digits := make([]byte, length)
	for index := range digits {
		value, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		digits[index] = byte('0' + value.Int64())
	}
	return string(digits), nil
}

func maskAuthDestination(channel domain.AuthCodeChannel, destination string) string {
	switch channel {
	case domain.AuthCodeChannelPhone:
		return maskPhone(destination)
	default:
		return maskEmail(destination)
	}
}

func maskEmail(email string) string {
	parts := strings.SplitN(strings.TrimSpace(email), "@", 2)
	if len(parts) != 2 {
		return ""
	}

	local := parts[0]
	if len(local) <= 2 {
		return local[:1] + "***@" + parts[1]
	}
	return local[:1] + "***" + local[len(local)-1:] + "@" + parts[1]
}

func maskPhone(phone string) string {
	normalized := strings.TrimSpace(phone)
	if len(normalized) <= 6 {
		return normalized
	}
	return normalized[:3] + "*****" + normalized[len(normalized)-2:]
}
