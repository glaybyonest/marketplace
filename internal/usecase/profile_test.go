package usecase

import (
	"context"
	"testing"
	"time"

	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type profileUserRepoMock struct {
	users map[uuid.UUID]domain.User
}

func (m *profileUserRepoMock) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	user, ok := m.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return user, nil
}

func (m *profileUserRepoMock) UpdateFullName(ctx context.Context, id uuid.UUID, fullName *string) (domain.User, error) {
	user, ok := m.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	if fullName == nil {
		user.FullName = ""
	} else {
		user.FullName = *fullName
	}
	user.UpdatedAt = time.Now().UTC()
	m.users[id] = user
	return user, nil
}

func TestProfileService(t *testing.T) {
	userID := uuid.New()
	repo := &profileUserRepoMock{
		users: map[uuid.UUID]domain.User{
			userID: {ID: userID, Email: "profile@example.com", FullName: "User Name", IsActive: true},
		},
	}
	service := NewProfileService(repo)

	tests := []struct {
		name    string
		run     func() error
		wantErr error
	}{
		{"get success", func() error { _, err := service.Get(context.Background(), userID); return err }, nil},
		{"get unauthorized", func() error { _, err := service.Get(context.Background(), uuid.Nil); return err }, domain.ErrUnauthorized},
		{"get not found", func() error { _, err := service.Get(context.Background(), uuid.New()); return err }, domain.ErrNotFound},
		{"update nil pointer returns current", func() error { _, err := service.Update(context.Background(), userID, nil); return err }, nil},
		{"update unauthorized", func() error {
			name := "Name"
			_, err := service.Update(context.Background(), uuid.Nil, &name)
			return err
		}, domain.ErrUnauthorized},
		{"update not found", func() error {
			name := "Name"
			_, err := service.Update(context.Background(), uuid.New(), &name)
			return err
		}, domain.ErrNotFound},
		{"update trim spaces", func() error {
			name := "   Trimmed Name   "
			updated, err := service.Update(context.Background(), userID, &name)
			if err == nil && updated.FullName != "Trimmed Name" {
				return domain.ErrInvalidInput
			}
			return err
		}, nil},
		{"update empty to null", func() error {
			name := "   "
			updated, err := service.Update(context.Background(), userID, &name)
			if err == nil && updated.FullName != "" {
				return domain.ErrInvalidInput
			}
			return err
		}, nil},
		{"update too long", func() error {
			name := ""
			for i := 0; i < 121; i++ {
				name += "a"
			}
			_, err := service.Update(context.Background(), userID, &name)
			return err
		}, domain.ErrInvalidInput},
		{"update ascii", func() error {
			name := "John"
			_, err := service.Update(context.Background(), userID, &name)
			return err
		}, nil},
		{"update cyrillic", func() error {
			name := "Иван"
			_, err := service.Update(context.Background(), userID, &name)
			return err
		}, nil},
		{"update single char", func() error { name := "A"; _, err := service.Update(context.Background(), userID, &name); return err }, nil},
		{"update with dot", func() error {
			name := "A.B."
			_, err := service.Update(context.Background(), userID, &name)
			return err
		}, nil},
		{"update with dash", func() error {
			name := "Anna-Maria"
			_, err := service.Update(context.Background(), userID, &name)
			return err
		}, nil},
		{"update with apostrophe", func() error {
			name := "O'Connor"
			_, err := service.Update(context.Background(), userID, &name)
			return err
		}, nil},
		{"update with numbers", func() error {
			name := "User 2"
			_, err := service.Update(context.Background(), userID, &name)
			return err
		}, nil},
		{"update with tabs trim", func() error {
			name := "\tName\t"
			updated, err := service.Update(context.Background(), userID, &name)
			if err == nil && updated.FullName != "Name" {
				return domain.ErrInvalidInput
			}
			return err
		}, nil},
		{"update with newline trim", func() error {
			name := "\nName\n"
			updated, err := service.Update(context.Background(), userID, &name)
			if err == nil && updated.FullName != "Name" {
				return domain.ErrInvalidInput
			}
			return err
		}, nil},
		{"update max length boundary", func() error {
			name := ""
			for i := 0; i < 120; i++ {
				name += "x"
			}
			_, err := service.Update(context.Background(), userID, &name)
			return err
		}, nil},
		{"get after update", func() error {
			_, err := service.Get(context.Background(), userID)
			return err
		}, nil},
		{"update idempotent same value", func() error {
			name := "Same Value"
			_, err := service.Update(context.Background(), userID, &name)
			if err != nil {
				return err
			}
			_, err = service.Update(context.Background(), userID, &name)
			return err
		}, nil},
		{"update unicode", func() error {
			name := "Müller"
			_, err := service.Update(context.Background(), userID, &name)
			return err
		}, nil},
		{"update emoji allowed", func() error {
			name := "Name 🙂"
			_, err := service.Update(context.Background(), userID, &name)
			return err
		}, nil},
		{"update trailing spaces", func() error {
			name := "Name   "
			updated, err := service.Update(context.Background(), userID, &name)
			if err == nil && updated.FullName != "Name" {
				return domain.ErrInvalidInput
			}
			return err
		}, nil},
		{"update leading spaces", func() error {
			name := "   Name"
			updated, err := service.Update(context.Background(), userID, &name)
			if err == nil && updated.FullName != "Name" {
				return domain.ErrInvalidInput
			}
			return err
		}, nil},
		{"update long phrase", func() error {
			name := "Very Long But Valid Person Name With Several Words"
			_, err := service.Update(context.Background(), userID, &name)
			return err
		}, nil},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if tc.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}
