package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"reflect"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/repository/postgres/sqlc"
)

type CategoryRepository struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool, sqlc.New(pool)}
}
func (r *CategoryRepository) List(ctx context.Context, user uuid.UUID, limit, offset int32) ([]domain.Category, error) {
	rows, err := r.q.ListCategoryDocuments(ctx, sqlc.ListCategoryDocumentsParams{UserID: user, Limit: limit, Offset: offset})
	if err != nil {
		return nil, fridgeError(err)
	}
	result := make([]domain.Category, 0, len(rows))
	for _, row := range rows {
		var item domain.Category
		if err = json.Unmarshal(row, &item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}
func (r *CategoryRepository) Save(ctx context.Context, user uuid.UUID, item domain.Category, base *int64) (domain.Category, error) {
	err := ownerTransaction(ctx, r.pool, user, func(q *sqlc.Queries) error {
		raw, err := q.FindCategoryDocument(ctx, sqlc.FindCategoryDocumentParams{ID: item.ID, UserID: user})
		if errors.Is(err, pgx.ErrNoRows) {
			if base != nil {
				return domain.ErrNotFound
			}
			item.Version = 1
		} else if err != nil {
			return err
		} else {
			var old domain.Category
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
		raw, err = q.SaveCategoryDocument(ctx, data)
		if err != nil {
			return err
		}
		return json.Unmarshal(raw, &item)
	})
	return item, err
}
func (r *CategoryRepository) Delete(ctx context.Context, user, id uuid.UUID, base *int64) error {
	return ownerTransaction(ctx, r.pool, user, func(q *sqlc.Queries) error {
		raw, err := q.FindCategoryDocument(ctx, sqlc.FindCategoryDocumentParams{ID: id, UserID: user})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		var old domain.Category
		if err = json.Unmarshal(raw, &old); err != nil {
			return err
		}
		if base != nil && *base != int64(old.Version) {
			return domain.ErrVersionConflict
		}
		if err = q.ClearCategoryReferences(ctx, sqlc.ClearCategoryReferencesParams{UserID: user, CategoryID: pgtype.UUID{Bytes: id, Valid: true}}); err != nil {
			return err
		}
		return q.DeleteCategoryDocument(ctx, sqlc.DeleteCategoryDocumentParams{ID: id, UserID: user})
	})
}
