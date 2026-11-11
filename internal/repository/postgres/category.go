package postgres

import (
	"context"
	"errors"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/repository/postgres/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CategoryRepository struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool, sqlc.New(pool)}
}

func (r *CategoryRepository) List(ctx context.Context, userID domain.UserID) ([]domain.Category, error) {
	rows, err := r.q.ListCategories(ctx, uuid.UUID(userID))
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}

	res := make([]domain.Category, 0, len(rows))
	for _, row := range rows {
		res = append(res, domain.Category{
			ID:      domain.CategoryID(row.ID),
			UserID:  domain.UserID(row.UserID),
			Name:    row.Name,
			Version: row.Version,
		})
	}
	return res, nil
}

func (r *CategoryRepository) Create(ctx context.Context, c *domain.Category) error {
	if err := r.q.CreateCategory(ctx, sqlc.CreateCategoryParams{
		UserID: uuid.UUID(c.UserID),
		ID:     uuid.UUID(c.ID),
		Name:   c.Name,
	}); err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("create category: %w", err)
	}
	return nil
}

func (r *CategoryRepository) Update(ctx context.Context, c *domain.Category) error {
	updated, err := r.q.UpdateCategory(ctx, sqlc.UpdateCategoryParams{
		Name:        c.Name,
		UserID:      uuid.UUID(c.UserID),
		ID:          uuid.UUID(c.ID),
		BaseVersion: c.Version,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("update category: %w", err)
	}
	if updated == 0 {
		return domain.ErrVersionConflict
	}
	return nil
}

func (r *CategoryRepository) Delete(ctx context.Context, in dto.CategoryDeleteInput) error {
	deleted, err := r.q.DeleteCategory(ctx, sqlc.DeleteCategoryParams{
		UserID:      in.UserID,
		ID:          in.CategoryID,
		BaseVersion: in.BaseVersion,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23503" {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("delete category: %w", err)
	}
	if deleted == 0 {
		return domain.ErrVersionConflict
	}
	return nil
}
