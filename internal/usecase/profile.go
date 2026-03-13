package usecase

import (
	"context"
	"strings"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/observability"

	"github.com/google/uuid"
)

type ProfileUserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	UpdateFullName(ctx context.Context, id uuid.UUID, fullName *string) (domain.User, error)
}

type ProfileAuditLogger interface {
	Record(ctx context.Context, entry observability.AuditEntry) error
}

type ProfileService struct {
	users ProfileUserRepository
	audit ProfileAuditLogger
}

func NewProfileService(users ProfileUserRepository, audit ProfileAuditLogger) *ProfileService {
	return &ProfileService{users: users, audit: audit}
}

func (s *ProfileService) Get(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	if userID == uuid.Nil {
		return domain.User{}, domain.ErrUnauthorized
	}
	return s.users.GetByID(ctx, userID)
}

func (s *ProfileService) Update(ctx context.Context, userID uuid.UUID, fullName *string) (domain.User, error) {
	if userID == uuid.Nil {
		return domain.User{}, domain.ErrUnauthorized
	}

	if fullName == nil {
		return s.users.GetByID(ctx, userID)
	}

	currentUser, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}

	value := strings.TrimSpace(*fullName)
	if len(value) > 120 {
		return domain.User{}, domain.ErrInvalidInput
	}
	if value == "" {
		updatedUser, err := s.users.UpdateFullName(ctx, userID, nil)
		if err != nil {
			return domain.User{}, err
		}
		s.recordAudit(ctx, userID, currentUser.FullName, updatedUser.FullName)
		return updatedUser, nil
	}

	updatedUser, err := s.users.UpdateFullName(ctx, userID, &value)
	if err != nil {
		return domain.User{}, err
	}
	s.recordAudit(ctx, userID, currentUser.FullName, updatedUser.FullName)
	return updatedUser, nil
}

func (s *ProfileService) recordAudit(ctx context.Context, userID uuid.UUID, before, after string) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Record(ctx, observability.AuditEntry{
		ActorUserID: ptrUUID(userID),
		Action:      "profile.updated",
		EntityType:  "user",
		EntityID:    ptrUUID(userID),
		Metadata: map[string]any{
			"before_full_name": before,
			"after_full_name":  after,
		},
	})
}
