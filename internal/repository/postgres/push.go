package postgres

import (
	"context"
	"errors"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/repository/postgres/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PushSubscriptionRepository struct {
	q *sqlc.Queries
}

func NewPushSubscriptionRepository(pool *pgxpool.Pool) *PushSubscriptionRepository {
	return &PushSubscriptionRepository{
		q: sqlc.New(pool),
	}
}

func (r *PushSubscriptionRepository) Upsert(ctx context.Context, userID domain.UserID, input dto.PushSubscriptionInput) (uuid.UUID, error) {
	id, err := r.q.UpsertPushSubscription(ctx, sqlc.UpsertPushSubscriptionParams{
		UserID:   uuid.UUID(userID),
		Endpoint: input.Endpoint,
		P256dh:   input.P256DH,
		Auth:     input.Auth,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, domain.ErrNotFound
		}
		return uuid.Nil, fmt.Errorf("upsert push subscription: %w", err)
	}

	return id, nil
}

func (r *PushSubscriptionRepository) Revoke(ctx context.Context, userID domain.UserID, subscriptionID uuid.UUID) error {
	if err := r.q.RevokePushSubscription(ctx, sqlc.RevokePushSubscriptionParams{
		ID:     subscriptionID,
		UserID: uuid.UUID(userID),
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("revoke push subscription: %w", err)
	}

	return nil
}
