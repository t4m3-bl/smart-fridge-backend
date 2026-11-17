package postgres

import (
	"context"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/repository/postgres/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CategoryRepository struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{
		pool: pool,
		q:    sqlc.New(pool),
	}
}

// List returns all available categories.
func (r *CategoryRepository) List(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.q.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}

	res := make([]domain.Category, 0, len(rows))
	for _, row := range rows {
		res = append(res, domain.Category{
			ID:   domain.CategoryID(row.ID),
			Name: row.Name,
		})
	}

	return res, nil
}
