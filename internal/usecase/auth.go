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
	"marketplace-backend/internal/security"

	"github.com/google/uuid"
)

type AuthUserRepository interface {
	Create(ctx context.Context, input CreateUserInput) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error
	MarkEmailVerified(ctx context.Context, id uuid.UUID, verifiedAt time.Time) (domain.User, error)
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
	users            AuthUserRepository
	sessions         AuthSessionRepository
	actionTokens     AuthActionTokenRepository
	jwt              JWTProvider
	passwords        PasswordProvider
	mailer           Mailer
	refreshTTL       time.Duration
	emailVerifyTTL   time.Duration
	passwordResetTTL time.Duration
	appBaseURL       string
	mailFrom         string
	adminEmails      map[string]struct{}
	now              func() time.Time
}

func NewAuthService(
	users AuthUserRepository,
	sessions AuthSessionRepository,
	actionTokens AuthActionTokenRepository,
	jwt JWTProvider,
	passwords PasswordProvider,
	mailer Mailer,
	appBaseURL string,
	mailFrom string,
	adminEmails []string,
	refreshTTL time.Duration,
	emailVerifyTTL time.Duration,
	passwordResetTTL time.Duration,
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
		users:            users,
		sessions:         sessions,
		actionTokens:     actionTokens,
		jwt:              jwt,
		passwords:        passwords,
		mailer:           mailer,
		refreshTTL:       refreshTTL,
		emailVerifyTTL:   emailVerifyTTL,
		passwordResetTTL: passwordResetTTL,
		appBaseURL:       strings.TrimRight(strings.TrimSpace(appBaseURL), "/"),
		mailFrom:         strings.TrimSpace(mailFrom),
		adminEmails:      normalizedAdminEmails,
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

	return domain.AuthResult{
		User:                      user,
		RequiresEmailVerification: true,
		Message:                   "verification email sent",
	}, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (domain.AuthResult, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return domain.AuthResult{}, domain.ErrUnauthorized
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if err == domain.ErrNotFound {
			return domain.AuthResult{}, domain.ErrUnauthorized
		}
		return domain.AuthResult{}, err
	}
	if !user.IsActive {
		return domain.AuthResult{}, domain.ErrInactiveUser
	}
	if !user.IsEmailVerified {
		return domain.AuthResult{}, domain.ErrEmailNotVerified
	}
	if !s.passwords.Compare(user.PasswordHash, input.Password) {
		return domain.AuthResult{}, domain.ErrUnauthorized
	}

	tokens, err := s.issueTokens(ctx, user, input.UserAgent, input.IP)
	if err != nil {
		return domain.AuthResult{}, err
	}

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
			return domain.TokenPair{}, domain.ErrUnauthorized
		}
		return domain.TokenPair{}, err
	}
	if session.RevokedAt != nil {
		return domain.TokenPair{}, domain.ErrSessionClosed
	}
	if session.RotatedAt != nil {
		return domain.TokenPair{}, domain.ErrTokenReused
	}
	if now.After(session.ExpiresAt) {
		_, _ = s.sessions.RevokeByRefreshTokenHash(ctx, session.UserID, tokenHash, now)
		return domain.TokenPair{}, domain.ErrUnauthorized
	}

	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil {
		if err == domain.ErrNotFound {
			return domain.TokenPair{}, domain.ErrUnauthorized
		}
		return domain.TokenPair{}, err
	}
	if !user.IsActive {
		return domain.TokenPair{}, domain.ErrInactiveUser
	}
	if !user.IsEmailVerified {
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

	return domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(accessExp.Sub(now).Seconds()),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, input LogoutInput) error {
	if input.UserID == uuid.Nil || strings.TrimSpace(input.RefreshToken) == "" {
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
		return domain.ErrUnauthorized
	}
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

	return s.issueEmailVerification(ctx, user)
}

func (s *AuthService) ConfirmEmailVerification(ctx context.Context, input VerifyEmailConfirmInput) (domain.User, error) {
	token, err := normalizeActionToken(input.Token)
	if err != nil {
		return domain.User{}, err
	}

	actionToken, err := s.actionTokens.GetActiveByHash(ctx, domain.AuthActionVerifyEmail, security.HashToken(token), s.now())
	if err != nil {
		if err == domain.ErrNotFound {
			return domain.User{}, domain.ErrInvalidToken
		}
		return domain.User{}, err
	}

	consumed, err := s.actionTokens.Consume(ctx, actionToken.ID, s.now())
	if err != nil {
		return domain.User{}, err
	}
	if !consumed {
		return domain.User{}, domain.ErrInvalidToken
	}

	user, err := s.users.GetByID(ctx, actionToken.UserID)
	if err != nil {
		return domain.User{}, err
	}
	if user.IsEmailVerified {
		return user, nil
	}

	return s.users.MarkEmailVerified(ctx, user.ID, s.now())
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

	return s.issuePasswordReset(ctx, user)
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
	if err := s.sessions.RevokeAllByUserID(ctx, actionToken.UserID, s.now()); err != nil {
		return err
	}

	consumed, err := s.actionTokens.Consume(ctx, actionToken.ID, s.now())
	if err != nil {
		return err
	}
	if !consumed {
		return domain.ErrInvalidToken
	}
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
