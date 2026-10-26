package postgres

import (
	"context"
	"errors"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/repository/postgres/sqlc"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{
		pool: pool,
		q:    sqlc.New(pool),
	}
}

func (r *SessionRepository) Create(ctx context.Context, session *domain.Session) error {
	if err := r.q.CreateSession(ctx, sqlc.CreateSessionParams{
		ID:               uuid.UUID(session.ID),
		UserID:           uuid.UUID(session.UserID),
		RefreshTokenHash: session.RefreshTokenHash,
		CreatedAt:        session.CreatedAt,
		ExpiresAt:        session.CreatedAt.Add(session.ExpiresIn),
	}); err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}

func (r *SessionRepository) Rotate(ctx context.Context, oldRefresh, newRefresh string, expiresIn time.Duration) (*domain.Session, error) {
	var session *domain.Session

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// --- TRANSACTION ---
	qtx := r.q.WithTx(tx)

	userID, err := qtx.RevokeActiveSession(ctx, oldRefresh)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvalidSession
		}
		return nil, err
	}

	session, err = domain.NewSession(domain.UserID(userID), newRefresh, expiresIn)
	if err != nil {
		return nil, err
	}

	if err := qtx.CreateSession(ctx, sqlc.CreateSessionParams{
		ID:               uuid.UUID(session.ID),
		UserID:           uuid.UUID(session.UserID),
		RefreshTokenHash: session.RefreshTokenHash,
		CreatedAt:        session.CreatedAt,
		ExpiresAt:        session.CreatedAt.Add(session.ExpiresIn),
	}); err != nil {
		return nil, fmt.Errorf("rotate session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return session, nil
}

func (r *SessionRepository) Revoke(ctx context.Context, tokenHash string) error {
	if err := r.q.RevokeSession(ctx, tokenHash); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}

	return nil
}
