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
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		pool: pool,
		q:    sqlc.New(pool),
	}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// --- TRANSACTION ---
	qtx := r.q.WithTx(tx)

	if err := qtx.CreateUser(ctx, sqlc.CreateUserParams{
		ID:    uuid.UUID(user.ID),
		Email: string(user.Email),
	}); err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return fmt.Errorf("create user: %w", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("create user: %w", err)
	}

	for _, identity := range user.Identities {
		if err := qtx.CreateIdentity(ctx, sqlc.CreateIdentityParams{
			ID:             uuid.UUID(identity.IdentityID),
			UserID:         uuid.UUID(user.ID),
			Provider:       identity.Provider.String(),
			ProviderUserID: identity.ProviderUserID,
			PasswordHash:   identity.PasswordHash,
		}); err != nil {
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
				return fmt.Errorf("add identity: %w", domain.ErrAlreadyExists)
			}
			return fmt.Errorf("add identity: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
	row, err := r.q.GetUserByEmail(ctx, string(email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("find user by email: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return r.withIdentities(ctx, &domain.User{
		ID:    domain.UserID(row.ID),
		Email: email,
	})
}

func (r *UserRepository) FindByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
	row, err := r.q.GetUserByID(ctx, uuid.UUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("find user by id: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return r.withIdentities(ctx, &domain.User{
		ID:    id,
		Email: domain.Email(row.Email),
	})
}

func (r *UserRepository) FindByIdentity(ctx context.Context, provider domain.Provider, uid string) (*domain.User, error) {
	row, err := r.q.GetUserByIdentity(ctx, sqlc.GetUserByIdentityParams{
		Provider:       provider.String(),
		ProviderUserID: &uid,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("find user by identity: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("find user by identity: %w", err)
	}

	return r.withIdentities(ctx, &domain.User{
		ID:    domain.UserID(row.ID),
		Email: domain.Email(row.Email),
	})
}

func (r *UserRepository) withIdentities(ctx context.Context, user *domain.User) (*domain.User, error) {
	rows, err := r.q.GetIdentitiesByUserID(ctx, uuid.UUID(user.ID))
	if err != nil {
		return nil, fmt.Errorf("get identities: %w", err)
	}

	user.Identities = make([]domain.Identity, 0, len(rows))
	for _, row := range rows {
		provider, err := domain.ToProvider(row.Provider)
		if err != nil {
			return nil, fmt.Errorf("get identities: %w", err)
		}

		user.Identities = append(user.Identities, domain.Identity{
			IdentityID:     domain.IdentityID(row.ID),
			UserID:         domain.UserID(row.UserID),
			Provider:       provider,
			ProviderUserID: row.ProviderUserID,
			PasswordHash:   row.PasswordHash,
		})
	}

	return user, nil
}
