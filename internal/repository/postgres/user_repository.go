package postgres

import (
	"context"
	"time"

	"marketplace-backend/internal/domain"
	"marketplace-backend/internal/usecase"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, input usecase.CreateUserInput) (domain.User, error) {
	const q = `
		INSERT INTO users (email, password_hash, full_name, role, email_verified_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, password_hash, full_name, role, created_at, updated_at, is_active, email_verified_at
	`

	var user domain.User
	err := scanUser(r.db.QueryRow(ctx, q, input.Email, input.PasswordHash, input.FullName, input.Role, input.EmailVerifiedAt), &user)
	if err != nil {
		return domain.User{}, mapError(err)
	}
	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	const q = `
		SELECT id, email, password_hash, full_name, role, created_at, updated_at, is_active, email_verified_at
		FROM users
		WHERE email = $1
	`

	var user domain.User
	err := scanUser(r.db.QueryRow(ctx, q, email), &user)
	if err != nil {
		return domain.User{}, mapError(err)
	}
	return user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	const q = `
		SELECT id, email, password_hash, full_name, role, created_at, updated_at, is_active, email_verified_at
		FROM users
		WHERE id = $1
	`

	var user domain.User
	err := scanUser(r.db.QueryRow(ctx, q, id), &user)
	if err != nil {
		return domain.User{}, mapError(err)
	}
	return user, nil
}

func (r *UserRepository) UpdateFullName(ctx context.Context, id uuid.UUID, fullName *string) (domain.User, error) {
	const q = `
		UPDATE users
		SET full_name = $2
		WHERE id = $1
		RETURNING id, email, password_hash, full_name, role, created_at, updated_at, is_active, email_verified_at
	`

	var user domain.User
	err := scanUser(r.db.QueryRow(ctx, q, id, fullName), &user)
	if err != nil {
		return domain.User{}, mapError(err)
	}
	return user, nil
}

func (r *UserRepository) UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error {
	cmd, err := r.db.Exec(ctx, `
		UPDATE users
		SET password_hash = $2
		WHERE id = $1
	`, id, passwordHash)
	if err != nil {
		return mapError(err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepository) MarkEmailVerified(ctx context.Context, id uuid.UUID, verifiedAt time.Time) (domain.User, error) {
	const q = `
		UPDATE users
		SET email_verified_at = $2
		WHERE id = $1
		RETURNING id, email, password_hash, full_name, role, created_at, updated_at, is_active, email_verified_at
	`

	var user domain.User
	err := scanUser(r.db.QueryRow(ctx, q, id, verifiedAt), &user)
	if err != nil {
		return domain.User{}, mapError(err)
	}
	return user, nil
}

func (r *UserRepository) PromoteAdminsByEmail(ctx context.Context, emails []string) error {
	if len(emails) == 0 {
		return nil
	}

	_, err := r.db.Exec(ctx, `
		UPDATE users
		SET role = 'admin'
		WHERE email = ANY($1)
	`, emails)
	return mapError(err)
}

func scanUser(row pgx.Row, user *domain.User) error {
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.IsActive,
		&user.EmailVerifiedAt,
	)
	if err != nil {
		return err
	}
	if user.Role == "" {
		user.Role = domain.UserRoleCustomer
	}
	user.IsEmailVerified = user.EmailVerifiedAt != nil
	return nil
}
