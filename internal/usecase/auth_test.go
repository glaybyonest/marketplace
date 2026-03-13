package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/security"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type authState struct {
	now      func() time.Time
	users    map[uuid.UUID]domain.User
	byEmail  map[string]uuid.UUID
	sessions map[string]domain.UserSession
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
		ID:           uuid.New(),
		Email:        input.Email,
		PasswordHash: input.PasswordHash,
		CreatedAt:    m.state.now(),
		UpdatedAt:    m.state.now(),
		IsActive:     true,
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

type jwtMock struct {
	fail error
}

func (j *jwtMock) Generate(userID uuid.UUID, email string) (string, time.Time, error) {
	if j.fail != nil {
		return "", time.Time{}, j.fail
	}
	now := time.Now().UTC()
	return "access-" + userID.String(), now.Add(15 * time.Minute), nil
}

func TestAuthService(t *testing.T) {
	now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	state := &authState{
		now:      func() time.Time { return now },
		users:    map[uuid.UUID]domain.User{},
		byEmail:  map[string]uuid.UUID{},
		sessions: map[string]domain.UserSession{},
	}
	users := &authUserRepoMock{state: state}
	sessions := &authSessionRepoMock{state: state}
	jwt := &jwtMock{}
	service := NewAuthService(users, sessions, jwt, security.NewPasswordManager(), 30*24*time.Hour)
	service.now = state.now

	t.Run("register", func(t *testing.T) {
		tests := []struct {
			name    string
			input   RegisterInput
			wantErr error
		}{
			{"success", RegisterInput{Email: "User@Example.com", Password: "StrongPass1", FullName: "User Name"}, nil},
			{"invalid email", RegisterInput{Email: "bad", Password: "StrongPass1"}, domain.ErrInvalidInput},
			{"weak password short", RegisterInput{Email: "one@example.com", Password: "123"}, domain.ErrInvalidInput},
			{"weak password no digit", RegisterInput{Email: "two@example.com", Password: "StrongPass"}, domain.ErrInvalidInput},
			{"weak password no letter", RegisterInput{Email: "three@example.com", Password: "12345678"}, domain.ErrInvalidInput},
			{"duplicate email", RegisterInput{Email: "user@example.com", Password: "StrongPass1"}, domain.ErrConflict},
			{"normalize email lower", RegisterInput{Email: "LOWER@EXAMPLE.COM", Password: "StrongPass1"}, nil},
			{"trim full name", RegisterInput{Email: "trim@example.com", Password: "StrongPass1", FullName: "  Ivan  "}, nil},
			{"empty full name", RegisterInput{Email: "empty@example.com", Password: "StrongPass1", FullName: "   "}, nil},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				user, tokens, err := service.Register(context.Background(), tc.input)
				if tc.wantErr != nil {
					require.Error(t, err)
					assert.ErrorIs(t, err, tc.wantErr)
					return
				}
				require.NoError(t, err)
				assert.NotEmpty(t, user.ID)
				assert.NotEmpty(t, tokens.AccessToken)
				assert.NotEmpty(t, tokens.RefreshToken)
				assert.NotEqual(t, tc.input.Password, user.PasswordHash)
			})
		}
	})

	t.Run("login", func(t *testing.T) {
		tests := []struct {
			name    string
			input   LoginInput
			wantErr error
		}{
			{"success", LoginInput{Email: "user@example.com", Password: "StrongPass1"}, nil},
			{"normalize email", LoginInput{Email: "USER@EXAMPLE.COM", Password: "StrongPass1"}, nil},
			{"wrong password", LoginInput{Email: "user@example.com", Password: "WrongPass1"}, domain.ErrUnauthorized},
			{"unknown email", LoginInput{Email: "none@example.com", Password: "StrongPass1"}, domain.ErrUnauthorized},
			{"invalid email format", LoginInput{Email: "bad", Password: "StrongPass1"}, domain.ErrUnauthorized},
			{"short password", LoginInput{Email: "user@example.com", Password: "1"}, domain.ErrUnauthorized},
			{"empty password", LoginInput{Email: "user@example.com", Password: ""}, domain.ErrUnauthorized},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				_, pair, err := service.Login(context.Background(), tc.input)
				if tc.wantErr != nil {
					require.Error(t, err)
					assert.ErrorIs(t, err, tc.wantErr)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, "Bearer", pair.TokenType)
			})
		}
	})

	t.Run("refresh logout me and infra", func(t *testing.T) {
		user, pair, err := service.Login(context.Background(), LoginInput{Email: "user@example.com", Password: "StrongPass1"})
		require.NoError(t, err)

		invalid := "invalid-refresh-token-value"
		revoked := "revoked-refresh-token-value"
		reused := "reused-refresh-token-value"
		expired := "expired-refresh-token-value"
		nowCopy := state.now()
		revokedAt := nowCopy
		rotatedAt := nowCopy
		state.sessions[security.HashToken(revoked)] = domain.UserSession{
			ID:               uuid.New(),
			UserID:           user.ID,
			RefreshTokenHash: security.HashToken(revoked),
			ExpiresAt:        nowCopy.Add(time.Hour),
			RevokedAt:        &revokedAt,
		}
		state.sessions[security.HashToken(reused)] = domain.UserSession{
			ID:               uuid.New(),
			UserID:           user.ID,
			RefreshTokenHash: security.HashToken(reused),
			ExpiresAt:        nowCopy.Add(time.Hour),
			RotatedAt:        &rotatedAt,
		}
		state.sessions[security.HashToken(expired)] = domain.UserSession{
			ID:               uuid.New(),
			UserID:           user.ID,
			RefreshTokenHash: security.HashToken(expired),
			ExpiresAt:        nowCopy.Add(-time.Minute),
		}

		tests := []struct {
			name    string
			run     func() error
			wantErr error
		}{
			{"refresh success", func() error {
				_, err := service.Refresh(context.Background(), RefreshInput{RefreshToken: pair.RefreshToken})
				return err
			}, nil},
			{"refresh reused old token", func() error {
				_, err := service.Refresh(context.Background(), RefreshInput{RefreshToken: pair.RefreshToken})
				return err
			}, domain.ErrTokenReused},
			{"refresh unknown token", func() error {
				_, err := service.Refresh(context.Background(), RefreshInput{RefreshToken: invalid})
				return err
			}, domain.ErrUnauthorized},
			{"refresh revoked token", func() error {
				_, err := service.Refresh(context.Background(), RefreshInput{RefreshToken: revoked})
				return err
			}, domain.ErrSessionClosed},
			{"refresh rotated token", func() error {
				_, err := service.Refresh(context.Background(), RefreshInput{RefreshToken: reused})
				return err
			}, domain.ErrTokenReused},
			{"refresh expired token", func() error {
				_, err := service.Refresh(context.Background(), RefreshInput{RefreshToken: expired})
				return err
			}, domain.ErrUnauthorized},
			{"refresh empty token", func() error {
				_, err := service.Refresh(context.Background(), RefreshInput{RefreshToken: ""})
				return err
			}, domain.ErrUnauthorized},
			{"logout success", func() error {
				return service.Logout(context.Background(), LogoutInput{UserID: user.ID, RefreshToken: pair.RefreshToken + "-new"})
			}, domain.ErrUnauthorized},
			{"logout unknown token", func() error {
				return service.Logout(context.Background(), LogoutInput{UserID: user.ID, RefreshToken: invalid})
			}, domain.ErrUnauthorized},
			{"logout nil user", func() error {
				return service.Logout(context.Background(), LogoutInput{UserID: uuid.Nil, RefreshToken: invalid})
			}, domain.ErrUnauthorized},
			{"me success", func() error { _, err := service.Me(context.Background(), user.ID); return err }, nil},
			{"me nil user", func() error { _, err := service.Me(context.Background(), uuid.Nil); return err }, domain.ErrUnauthorized},
			{"me not found", func() error { _, err := service.Me(context.Background(), uuid.New()); return err }, domain.ErrNotFound},
			{"jwt fail on login", func() error {
				jwt.fail = errors.New("jwt-fail")
				defer func() { jwt.fail = nil }()
				_, _, err := service.Login(context.Background(), LoginInput{Email: "user@example.com", Password: "StrongPass1"})
				return err
			}, errors.New("jwt-fail")},
			{"repo user fail", func() error {
				users.force = errors.New("user-fail")
				defer func() { users.force = nil }()
				_, _, err := service.Login(context.Background(), LoginInput{Email: "user@example.com", Password: "StrongPass1"})
				return err
			}, errors.New("user-fail")},
			{"repo session fail", func() error {
				sessions.force = errors.New("session-fail")
				defer func() { sessions.force = nil }()
				_, _, err := service.Login(context.Background(), LoginInput{Email: "user@example.com", Password: "StrongPass1"})
				return err
			}, errors.New("session-fail")},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.run()
				if tc.wantErr == nil {
					require.NoError(t, err)
					return
				}
				require.Error(t, err)
				if tc.wantErr == domain.ErrUnauthorized || tc.wantErr == domain.ErrTokenReused ||
					tc.wantErr == domain.ErrSessionClosed || tc.wantErr == domain.ErrNotFound {
					assert.ErrorIs(t, err, tc.wantErr)
				} else {
					assert.Contains(t, err.Error(), tc.wantErr.Error())
				}
			})
		}
	})
}
