package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/mailer"
	"marketplace-backend/internal/observability"
	"marketplace-backend/internal/security"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type authState struct {
	now          func() time.Time
	users        map[uuid.UUID]domain.User
	byEmail      map[string]uuid.UUID
	sessions     map[string]domain.UserSession
	actionTokens map[string]domain.AuthActionToken
	sentMessages []mailer.Message
	auditEntries []observability.AuditEntry
}

type authUserRepoMock struct {
	state *authState
	force error
}

func (m *authUserRepoMock) Create(ctx context.Context, input CreateUserInput) (domain.User, error) {
	if m.force != nil {
		return domain.User{}, m.force
	}
	if _, exists := m.state.byEmail[input.Email]; exists {
		return domain.User{}, domain.ErrConflict
	}

	user := domain.User{
		ID:              uuid.New(),
		Email:           input.Email,
		PasswordHash:    input.PasswordHash,
		Role:            input.Role,
		CreatedAt:       m.state.now(),
		UpdatedAt:       m.state.now(),
		IsActive:        true,
		EmailVerifiedAt: input.EmailVerifiedAt,
		IsEmailVerified: input.EmailVerifiedAt != nil,
	}
	if input.FullName != nil {
		user.FullName = *input.FullName
	}

	m.state.users[user.ID] = user
	m.state.byEmail[user.Email] = user.ID
	return user, nil
}

func (m *authUserRepoMock) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	if m.force != nil {
		return domain.User{}, m.force
	}
	id, ok := m.state.byEmail[email]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return m.state.users[id], nil
}

func (m *authUserRepoMock) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	if m.force != nil {
		return domain.User{}, m.force
	}
	user, ok := m.state.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return user, nil
}

func (m *authUserRepoMock) UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error {
	if m.force != nil {
		return m.force
	}
	user, ok := m.state.users[id]
	if !ok {
		return domain.ErrNotFound
	}
	user.PasswordHash = passwordHash
	user.UpdatedAt = m.state.now()
	m.state.users[id] = user
	return nil
}

func (m *authUserRepoMock) MarkEmailVerified(ctx context.Context, id uuid.UUID, verifiedAt time.Time) (domain.User, error) {
	if m.force != nil {
		return domain.User{}, m.force
	}
	user, ok := m.state.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	user.EmailVerifiedAt = &verifiedAt
	user.IsEmailVerified = true
	user.UpdatedAt = m.state.now()
	m.state.users[id] = user
	return user, nil
}

func (m *authUserRepoMock) RegisterFailedLogin(
	ctx context.Context,
	id uuid.UUID,
	failedAt time.Time,
	window time.Duration,
	maxAttempts int,
	lockoutDuration time.Duration,
) (domain.User, error) {
	if m.force != nil {
		return domain.User{}, m.force
	}
	user, ok := m.state.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	if user.LastFailedLoginAt == nil || !user.LastFailedLoginAt.After(failedAt.Add(-window)) {
		user.FailedLoginAttempts = 1
	} else {
		user.FailedLoginAttempts++
	}
	user.LastFailedLoginAt = &failedAt
	if user.FailedLoginAttempts >= maxAttempts {
		lockedUntil := failedAt.Add(lockoutDuration)
		user.LockedUntil = &lockedUntil
	}
	user.UpdatedAt = m.state.now()
	m.state.users[id] = user
	return user, nil
}

func (m *authUserRepoMock) ClearFailedLogin(ctx context.Context, id uuid.UUID) error {
	if m.force != nil {
		return m.force
	}
	user, ok := m.state.users[id]
	if !ok {
		return domain.ErrNotFound
	}
	user.FailedLoginAttempts = 0
	user.LastFailedLoginAt = nil
	user.LockedUntil = nil
	user.UpdatedAt = m.state.now()
	m.state.users[id] = user
	return nil
}

type authSessionRepoMock struct {
	state *authState
	force error
}

func (m *authSessionRepoMock) Create(ctx context.Context, input CreateSessionInput) (domain.UserSession, error) {
	if m.force != nil {
		return domain.UserSession{}, m.force
	}
	if _, exists := m.state.sessions[input.RefreshTokenHash]; exists {
		return domain.UserSession{}, domain.ErrConflict
	}
	session := domain.UserSession{
		ID:               uuid.New(),
		UserID:           input.UserID,
		RefreshTokenHash: input.RefreshTokenHash,
		UserAgent:        input.UserAgent,
		IP:               input.IP,
		CreatedAt:        m.state.now(),
		ExpiresAt:        input.ExpiresAt,
	}
	m.state.sessions[input.RefreshTokenHash] = session
	return session, nil
}

func (m *authSessionRepoMock) GetByRefreshTokenHash(ctx context.Context, tokenHash string) (domain.UserSession, error) {
	if m.force != nil {
		return domain.UserSession{}, m.force
	}
	session, ok := m.state.sessions[tokenHash]
	if !ok {
		return domain.UserSession{}, domain.ErrNotFound
	}
	return session, nil
}

func (m *authSessionRepoMock) Rotate(ctx context.Context, oldSessionID uuid.UUID, oldTokenHash string, rotatedAt time.Time, newSession CreateSessionInput) error {
	if m.force != nil {
		return m.force
	}
	session, ok := m.state.sessions[oldTokenHash]
	if !ok || session.ID != oldSessionID {
		return domain.ErrNotFound
	}
	if session.RotatedAt != nil || session.RevokedAt != nil {
		return domain.ErrTokenReused
	}
	session.RotatedAt = &rotatedAt
	m.state.sessions[oldTokenHash] = session
	_, err := m.Create(ctx, newSession)
	return err
}

func (m *authSessionRepoMock) RevokeByRefreshTokenHash(ctx context.Context, userID uuid.UUID, tokenHash string, revokedAt time.Time) (bool, error) {
	if m.force != nil {
		return false, m.force
	}
	session, ok := m.state.sessions[tokenHash]
	if !ok || session.UserID != userID || session.RevokedAt != nil {
		return false, nil
	}
	session.RevokedAt = &revokedAt
	m.state.sessions[tokenHash] = session
	return true, nil
}

func (m *authSessionRepoMock) RevokeAllByUserID(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error {
	if m.force != nil {
		return m.force
	}
	for key, session := range m.state.sessions {
		if session.UserID == userID && session.RevokedAt == nil {
			session.RevokedAt = &revokedAt
			m.state.sessions[key] = session
		}
	}
	return nil
}

type authActionTokenRepoMock struct {
	state *authState
	force error
}

func (m *authActionTokenRepoMock) Create(ctx context.Context, token domain.AuthActionToken) (domain.AuthActionToken, error) {
	if m.force != nil {
		return domain.AuthActionToken{}, m.force
	}
	token.CreatedAt = m.state.now()
	m.state.actionTokens[token.TokenHash] = token
	return token, nil
}

func (m *authActionTokenRepoMock) GetActiveByHash(
	ctx context.Context,
	purpose domain.AuthActionPurpose,
	tokenHash string,
	now time.Time,
) (domain.AuthActionToken, error) {
	if m.force != nil {
		return domain.AuthActionToken{}, m.force
	}
	token, ok := m.state.actionTokens[tokenHash]
	if !ok || token.Purpose != purpose || token.ConsumedAt != nil || !token.ExpiresAt.After(now) {
		return domain.AuthActionToken{}, domain.ErrNotFound
	}
	return token, nil
}

func (m *authActionTokenRepoMock) Consume(ctx context.Context, id uuid.UUID, consumedAt time.Time) (bool, error) {
	if m.force != nil {
		return false, m.force
	}
	for key, token := range m.state.actionTokens {
		if token.ID == id && token.ConsumedAt == nil {
			token.ConsumedAt = &consumedAt
			m.state.actionTokens[key] = token
			return true, nil
		}
	}
	return false, nil
}

func (m *authActionTokenRepoMock) DeleteActiveByUserAndPurpose(ctx context.Context, userID uuid.UUID, purpose domain.AuthActionPurpose) error {
	if m.force != nil {
		return m.force
	}
	for key, token := range m.state.actionTokens {
		if token.UserID == userID && token.Purpose == purpose && token.ConsumedAt == nil {
			delete(m.state.actionTokens, key)
		}
	}
	return nil
}

type authMailerMock struct {
	state *authState
	force error
}

func (m *authMailerMock) Send(ctx context.Context, message mailer.Message) error {
	if m.force != nil {
		return m.force
	}
	m.state.sentMessages = append(m.state.sentMessages, message)
	return nil
}

type authAuditMock struct {
	state *authState
	force error
}

func (m *authAuditMock) Record(ctx context.Context, entry observability.AuditEntry) error {
	if m.force != nil {
		return m.force
	}
	m.state.auditEntries = append(m.state.auditEntries, entry)
	return nil
}

type jwtMock struct {
	fail error
}

func (j *jwtMock) Generate(userID uuid.UUID, email string, role domain.UserRole) (string, time.Time, error) {
	if j.fail != nil {
		return "", time.Time{}, j.fail
	}
	now := time.Now().UTC()
	return "access-" + userID.String() + "-" + string(role), now.Add(15 * time.Minute), nil
}

func extractActionToken(t *testing.T, messages []mailer.Message, subject string) string {
	t.Helper()

	for _, message := range messages {
		if !strings.Contains(message.Subject, subject) {
			continue
		}

		tokenIdx := strings.LastIndex(message.Text, "token=")
		require.NotEqual(t, -1, tokenIdx)

		token := strings.TrimSpace(message.Text[tokenIdx+len("token="):])
		if newlineIdx := strings.IndexByte(token, '\n'); newlineIdx >= 0 {
			token = token[:newlineIdx]
		}
		if token != "" {
			return token
		}
	}

	t.Fatalf("token with subject %q not found", subject)
	return ""
}

func TestAuthService(t *testing.T) {
	now := time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)
	state := &authState{
		now:          func() time.Time { return now },
		users:        map[uuid.UUID]domain.User{},
		byEmail:      map[string]uuid.UUID{},
		sessions:     map[string]domain.UserSession{},
		actionTokens: map[string]domain.AuthActionToken{},
		sentMessages: make([]mailer.Message, 0),
		auditEntries: make([]observability.AuditEntry, 0),
	}
	users := &authUserRepoMock{state: state}
	sessions := &authSessionRepoMock{state: state}
	actionTokens := &authActionTokenRepoMock{state: state}
	messageSender := &authMailerMock{state: state}
	auditLogger := &authAuditMock{state: state}
	jwt := &jwtMock{}
	service := NewAuthService(
		users,
		sessions,
		actionTokens,
		jwt,
		security.NewPasswordManager(),
		messageSender,
		auditLogger,
		"http://localhost:5173",
		"no-reply@marketplace.local",
		[]string{"admin@example.com"},
		30*24*time.Hour,
		24*time.Hour,
		time.Hour,
		15*time.Minute,
		5,
		15*time.Minute,
	)
	service.now = state.now

	t.Run("register requires verification", func(t *testing.T) {
		result, err := service.Register(context.Background(), RegisterInput{
			Email:    "User@Example.com",
			Password: "StrongPass1",
			FullName: "User Name",
		})
		require.NoError(t, err)
		assert.True(t, result.RequiresEmailVerification)
		assert.Nil(t, result.Tokens)
		assert.False(t, result.User.IsEmailVerified)
		assert.Equal(t, domain.UserRoleCustomer, result.User.Role)
		require.Len(t, state.sentMessages, 1)
		assert.Contains(t, state.sentMessages[0].Text, "http://localhost:5173/verify-email?token=")
		require.NotEmpty(t, state.auditEntries)
		assert.Equal(t, "auth.register", state.auditEntries[len(state.auditEntries)-1].Action)

		_, err = service.Login(context.Background(), LoginInput{
			Email:    "user@example.com",
			Password: "StrongPass1",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrEmailNotVerified)
	})

	t.Run("register allowlisted admin", func(t *testing.T) {
		result, err := service.Register(context.Background(), RegisterInput{
			Email:    "admin@example.com",
			Password: "StrongPass1",
			FullName: "Admin User",
		})
		require.NoError(t, err)
		assert.Equal(t, domain.UserRoleAdmin, result.User.Role)
	})

	t.Run("verify email and login", func(t *testing.T) {
		verifyToken := extractActionToken(t, state.sentMessages, "Verify your email")

		user, err := service.ConfirmEmailVerification(context.Background(), VerifyEmailConfirmInput{Token: verifyToken})
		require.NoError(t, err)
		assert.True(t, user.IsEmailVerified)

		result, err := service.Login(context.Background(), LoginInput{
			Email:    "user@example.com",
			Password: "StrongPass1",
		})
		require.NoError(t, err)
		require.NotNil(t, result.Tokens)
		assert.Equal(t, "Bearer", result.Tokens.TokenType)
		assert.Equal(t, "auth.login", state.auditEntries[len(state.auditEntries)-1].Action)

		_, err = service.ConfirmEmailVerification(context.Background(), VerifyEmailConfirmInput{Token: verifyToken})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidToken)
	})

	t.Run("login failure is audited", func(t *testing.T) {
		_, err := service.Login(context.Background(), LoginInput{
			Email:    "user@example.com",
			Password: "WrongPass9",
		})
		require.Error(t, err)
		assert.Equal(t, "auth.login_failed", state.auditEntries[len(state.auditEntries)-1].Action)
	})

	t.Run("login lockout activates after repeated failures", func(t *testing.T) {
		userID := state.byEmail["user@example.com"]
		user := state.users[userID]
		user.FailedLoginAttempts = 0
		user.LastFailedLoginAt = nil
		user.LockedUntil = nil
		state.users[userID] = user

		for i := 0; i < 4; i++ {
			_, err := service.Login(context.Background(), LoginInput{
				Email:    "user@example.com",
				Password: "WrongPass9",
			})
			require.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrUnauthorized)
		}

		_, err := service.Login(context.Background(), LoginInput{
			Email:    "user@example.com",
			Password: "WrongPass9",
		})
		require.Error(t, err)
		var lockErr *domain.LoginLockedError
		require.ErrorAs(t, err, &lockErr)
		assert.Equal(t, "auth.login_lockout_triggered", state.auditEntries[len(state.auditEntries)-1].Action)

		_, err = service.Login(context.Background(), LoginInput{
			Email:    "user@example.com",
			Password: "StrongPass1",
		})
		require.Error(t, err)
		require.ErrorAs(t, err, &lockErr)
		assert.Equal(t, "auth.login_locked", state.auditEntries[len(state.auditEntries)-1].Action)

		now = now.Add(16 * time.Minute)
		result, err := service.Login(context.Background(), LoginInput{
			Email:    "user@example.com",
			Password: "StrongPass1",
		})
		require.NoError(t, err)
		require.NotNil(t, result.Tokens)

		user = state.users[userID]
		assert.Zero(t, user.FailedLoginAttempts)
		assert.Nil(t, user.LockedUntil)
	})

	t.Run("request verification resend is generic", func(t *testing.T) {
		require.NoError(t, service.RequestEmailVerification(context.Background(), VerifyEmailRequestInput{
			Email: "missing@example.com",
		}))
		require.NoError(t, service.RequestEmailVerification(context.Background(), VerifyEmailRequestInput{
			Email: "user@example.com",
		}))
	})

	t.Run("password reset", func(t *testing.T) {
		loginResult, err := service.Login(context.Background(), LoginInput{
			Email:    "user@example.com",
			Password: "StrongPass1",
		})
		require.NoError(t, err)
		oldRefreshToken := loginResult.Tokens.RefreshToken
		messageCountBefore := len(state.sentMessages)

		err = service.RequestPasswordReset(context.Background(), PasswordResetRequestInput{
			Email: "user@example.com",
		})
		require.NoError(t, err)
		require.Len(t, state.sentMessages, messageCountBefore+1)
		assert.Contains(t, state.sentMessages[len(state.sentMessages)-1].Text, "http://localhost:5173/reset-password?token=")

		resetToken := extractActionToken(t, state.sentMessages, "Reset your password")
		require.NoError(t, service.ConfirmPasswordReset(context.Background(), PasswordResetConfirmInput{
			Token:       resetToken,
			NewPassword: "StrongerPass2",
		}))

		_, err = service.Login(context.Background(), LoginInput{
			Email:    "user@example.com",
			Password: "StrongPass1",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrUnauthorized)

		_, err = service.Login(context.Background(), LoginInput{
			Email:    "user@example.com",
			Password: "StrongerPass2",
		})
		require.NoError(t, err)

		_, err = service.Refresh(context.Background(), RefreshInput{
			RefreshToken: oldRefreshToken,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrSessionClosed)
	})

	t.Run("request reset for unknown email is generic", func(t *testing.T) {
		require.NoError(t, service.RequestPasswordReset(context.Background(), PasswordResetRequestInput{
			Email: "nobody@example.com",
		}))
	})

	t.Run("invalid token and infra failures", func(t *testing.T) {
		err := service.ConfirmPasswordReset(context.Background(), PasswordResetConfirmInput{
			Token:       "missing-token-value-1234567890",
			NewPassword: "StrongPass1",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidToken)

		err = service.ConfirmPasswordReset(context.Background(), PasswordResetConfirmInput{
			Token:       "another-missing-token-value-1234567890",
			NewPassword: "123",
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)

		jwt.fail = errors.New("jwt-fail")
		defer func() { jwt.fail = nil }()
		_, err = service.Login(context.Background(), LoginInput{
			Email:    "user@example.com",
			Password: "StrongerPass2",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "jwt-fail")

		messageSender.force = errors.New("mail-fail")
		defer func() { messageSender.force = nil }()
		err = service.RequestPasswordReset(context.Background(), PasswordResetRequestInput{
			Email: "user@example.com",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "mail-fail")
	})

	t.Run("logout and me", func(t *testing.T) {
		result, err := service.Login(context.Background(), LoginInput{
			Email:    "user@example.com",
			Password: "StrongerPass2",
		})
		require.NoError(t, err)

		require.NoError(t, service.Logout(context.Background(), LogoutInput{
			UserID:       result.User.ID,
			RefreshToken: result.Tokens.RefreshToken,
		}))

		_, err = service.Refresh(context.Background(), RefreshInput{
			RefreshToken: result.Tokens.RefreshToken,
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrSessionClosed)

		me, err := service.Me(context.Background(), result.User.ID)
		require.NoError(t, err)
		assert.Equal(t, result.User.ID, me.ID)
	})
}
