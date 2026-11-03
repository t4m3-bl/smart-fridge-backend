package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/repository/postgres/sqlc"
)

func fridgeError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		switch pe.Code {
		case "23505":
			return domain.ErrAlreadyExists
		case "23503", "23514", "22007", "22008":
			return domain.NewValidationErr("data", "invalid reference or value")
		}
	}
	return fmt.Errorf("fridge repository: %w", err)
}
func ownerTransaction(ctx context.Context, pool *pgxpool.Pool, user uuid.UUID, fn func(*sqlc.Queries) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := sqlc.New(tx)
	if err = q.LockFridgeOwner(ctx, user); err != nil {
		return err
	}
	if err = fn(q); err != nil {
		return fridgeError(err)
	}
	return tx.Commit(ctx)
}
func document(v any, user uuid.UUID) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var obj map[string]any
	if err = json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	obj["user_id"] = user.String()
	return json.Marshal(obj)
}
