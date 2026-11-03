package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"reflect"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/repository/postgres/sqlc"
)

type ProductRepository struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{pool, sqlc.New(pool)}
}
func (r *ProductRepository) List(ctx context.Context, user uuid.UUID, limit, offset int32) ([]domain.Product, error) {
	rows, err := r.q.ListProductDocuments(ctx, sqlc.ListProductDocumentsParams{UserID: user, Limit: limit, Offset: offset})
	if err != nil {
		return nil, fridgeError(err)
	}
	result := make([]domain.Product, 0, len(rows))
	for _, row := range rows {
		var item domain.Product
		if err = json.Unmarshal(row, &item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}
func (r *ProductRepository) Save(ctx context.Context, user uuid.UUID, item domain.Product, base *int64) (domain.Product, error) {
	err := ownerTransaction(ctx, r.pool, user, func(q *sqlc.Queries) error {
		raw, err := q.FindProductDocument(ctx, sqlc.FindProductDocumentParams{ID: item.ID, UserID: user})
		if errors.Is(err, pgx.ErrNoRows) {
			if base != nil {
				return domain.ErrNotFound
			}
			item.Version = 1
		} else if err != nil {
			return err
		} else {
			var old domain.Product
			if err = json.Unmarshal(raw, &old); err != nil {
				return err
			}
			item.Version = old.Version
			if base == nil {
				if reflect.DeepEqual(item, old) {
					return nil
				}
				return domain.ErrVersionConflict
			}
			if *base != int64(old.Version) {
				return domain.ErrVersionConflict
			}
			item.Version++
		}
		data, err := document(item, user)
		if err != nil {
			return err
		}
		raw, err = q.SaveProductDocument(ctx, data)
		if err != nil {
			return err
		}
		return json.Unmarshal(raw, &item)
	})
	return item, err
}
func (r *ProductRepository) Delete(ctx context.Context, user, id uuid.UUID, base *int64) error {
	return ownerTransaction(ctx, r.pool, user, func(q *sqlc.Queries) error {
		raw, err := q.FindProductDocument(ctx, sqlc.FindProductDocumentParams{ID: id, UserID: user})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		var old domain.Product
		if err = json.Unmarshal(raw, &old); err != nil {
			return err
		}
		if base != nil && *base != int64(old.Version) {
			return domain.ErrVersionConflict
		}

		return q.DeleteProductDocument(ctx, sqlc.DeleteProductDocumentParams{ID: id, UserID: user})
	})
}
