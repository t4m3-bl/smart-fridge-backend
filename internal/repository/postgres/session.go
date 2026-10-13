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
)

type TxDB interface {
	sqlc.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

type SessionRepository struct {
	db TxDB
	q  *sqlc.Queries
}

func NewSessionRepository(db TxDB) *SessionRepository {
	return &SessionRepository{
		db: db,
		q:  sqlc.New(db),
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

	err := pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)

		userID, err := q.RevokeActiveSession(ctx, oldRefresh)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrInvalidSession
		}
		if err != nil {
			return err
		}

		session, err = domain.NewSession(domain.ID(userID), newRefresh, expiresIn)
		if err != nil {
			return err
		}

		return q.CreateSession(ctx, sqlc.CreateSessionParams{
			ID:               uuid.UUID(session.ID),
			UserID:           uuid.UUID(session.UserID),
			RefreshTokenHash: session.RefreshTokenHash,
			CreatedAt:        session.CreatedAt,
			ExpiresAt:        session.CreatedAt.Add(session.ExpiresIn),
		})
	})
	if err != nil {
		return nil, fmt.Errorf("rotate session: %w", err)
	}

	return session, nil
}

func (r *SessionRepository) Revoke(ctx context.Context, tokenHash string) error {
	if err := r.q.RevokeSession(ctx, tokenHash); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}

	return nil
}
