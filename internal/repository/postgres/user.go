package postgres

import (
	"context"
	"errors"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/repository/postgres/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type UserRepository struct {
	q *sqlc.Queries
}

func NewUserRepository(db sqlc.DBTX) *UserRepository {
	return &UserRepository{q: sqlc.New(db)}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	if err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           uuid.UUID(user.ID),
		Email:        string(user.Email),
		PasswordHash: &user.PasswordHash,
	}); err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return fmt.Errorf("create user: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("find user by email: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return toDomainUser(row.ID, row.Email, row.PasswordHash), nil
}

func (r *UserRepository) FindByID(ctx context.Context, id domain.ID) (*domain.User, error) {
	row, err := r.q.GetUserByID(ctx, uuid.UUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("find user by id: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return toDomainUser(row.ID, row.Email, row.PasswordHash), nil
}

func toDomainUser(id uuid.UUID, email string, passwordHash *string) *domain.User {
	user := &domain.User{
		ID:    domain.ID(id),
		Email: domain.Email(email),
	}
	if passwordHash != nil {
		user.PasswordHash = *passwordHash
	}

	return user
}
