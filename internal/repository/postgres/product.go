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

type ProductRepository struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{
		pool: pool,
		q:    sqlc.New(pool),
	}
}

func (r *ProductRepository) List(ctx context.Context, userID domain.UserID) ([]domain.Product, error) {
	rows, err := r.q.ListProducts(ctx, uuid.UUID(userID))
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}

	products := make([]domain.Product, 0, len(rows))
	for _, row := range rows {
		products = append(products, domain.Product{
			ID:      domain.ProductID(row.ID),
			UserID:  domain.UserID(row.UserID),
			Version: row.Version,
			ProductData: domain.ProductData{
				Name:                  row.Name,
				CategoryID:            domain.CategoryID(row.CategoryID),
				Description:           row.Description,
				Quantity:              row.Quantity,
				ImageLink:             (*domain.ImageLink)(row.ImageLink),
				ManufacturedAt:        row.ManufacturedAt,
				ExpiresAt:             row.ExpiresAt,
				OpenedAt:              row.OpenedAt,
				ShelfLifeAfterOpening: row.ShelfLifeAfterOpening,
			},
		})
	}

	return products, nil
}

func (r *ProductRepository) Get(ctx context.Context, userID domain.UserID, id domain.ProductID) (*domain.Product, error) {
	row, err := r.q.GetProduct(ctx, sqlc.GetProductParams{
		UserID: uuid.UUID(userID),
		ID:     uuid.UUID(id),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get product: %w", err)
	}

	return &domain.Product{
		ID:      domain.ProductID(row.ID),
		UserID:  domain.UserID(row.UserID),
		Version: row.Version,
		ProductData: domain.ProductData{
			Name:                  row.Name,
			CategoryID:            domain.CategoryID(row.CategoryID),
			Description:           row.Description,
			Quantity:              row.Quantity,
			ImageLink:             (*domain.ImageLink)(row.ImageLink),
			ManufacturedAt:        row.ManufacturedAt,
			ExpiresAt:             row.ExpiresAt,
			OpenedAt:              row.OpenedAt,
			ShelfLifeAfterOpening: row.ShelfLifeAfterOpening,
		},
	}, nil
}

func (r *ProductRepository) Create(ctx context.Context, product *domain.Product) error {
	if err := r.q.CreateProduct(ctx, sqlc.CreateProductParams{
		UserID:                uuid.UUID(product.UserID),
		ID:                    uuid.UUID(product.ID),
		Name:                  product.Name,
		CategoryID:            uuid.UUID(product.CategoryID),
		Description:           product.Description,
		Quantity:              product.Quantity,
		ImageLink:             (*string)(product.ImageLink),
		ManufacturedAt:        product.ManufacturedAt,
		ExpiresAt:             product.ExpiresAt,
		OpenedAt:              product.OpenedAt,
		ShelfLifeAfterOpening: product.ShelfLifeAfterOpening,
	}); err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
			switch pgErr.Code {
			case "23505":
				return fmt.Errorf("create product: %w", domain.ErrAlreadyExists)
			case "23503":
				return domain.NewValidationErr("categoryId", "category not found")
			}
		}
		return fmt.Errorf("create product: %w", err)
	}

	return nil
}

func (r *ProductRepository) Update(ctx context.Context, product *domain.Product) error {
	updated, err := r.q.UpdateProduct(ctx, sqlc.UpdateProductParams{
		Name:                  product.Name,
		CategoryID:            uuid.UUID(product.CategoryID),
		Description:           product.Description,
		Quantity:              product.Quantity,
		ImageLink:             (*string)(product.ImageLink),
		ManufacturedAt:        product.ManufacturedAt,
		ExpiresAt:             product.ExpiresAt,
		OpenedAt:              product.OpenedAt,
		ShelfLifeAfterOpening: product.ShelfLifeAfterOpening,
		UserID:                uuid.UUID(product.UserID),
		ID:                    uuid.UUID(product.ID),
		BaseVersion:           product.Version,
	})
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23503" {
			return domain.NewValidationErr("categoryId", "category not found")
		}
		return fmt.Errorf("update product: %w", err)
	}

	if updated == 0 {
		return domain.ErrVersionConflict
	}

	return nil
}

func (r *ProductRepository) Delete(ctx context.Context, userID domain.UserID, id domain.ProductID, baseVersion int32) error {
	deleted, err := r.q.DeleteProduct(ctx, sqlc.DeleteProductParams{
		UserID:      uuid.UUID(userID),
		ID:          uuid.UUID(id),
		BaseVersion: baseVersion,
	})
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if deleted == 0 {
		return domain.ErrVersionConflict
	}
	return nil
}
