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
		products = append(products, *mapProduct(row))
	}

	return products, nil
}

func (r *ProductRepository) Get(
	ctx context.Context,
	userID domain.UserID,
	id domain.ProductID,
) (*domain.Product, error) {
	row, err := r.q.GetProduct(ctx, sqlc.GetProductParams{
		UserID: uuid.UUID(userID),
		ID:     uuid.UUID(id),
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}

	return mapProduct(row), nil
}

func (r *ProductRepository) Create(
	ctx context.Context,
	product *domain.Product,
) (*domain.Product, error) {
	created, err := r.q.CreateProduct(ctx, sqlc.CreateProductParams{
		UserID:                uuid.UUID(product.UserID),
		ID:                    uuid.UUID(product.ID),
		Name:                  product.Name,
		CategoryID:            int64(product.CategoryID),
		Description:           product.Description,
		Quantity:              product.Quantity,
		ImageKey:              product.ImageKey,
		ManufacturedAt:        product.ManufacturedAt,
		ExpiresAt:             product.ExpiresAt,
		OpenedAt:              product.OpenedAt,
		ShelfLifeAfterOpening: product.ShelfLifeAfterOpening,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAlreadyExists
	}

	if err != nil {
		return nil, productWriteError("create product", err)
	}

	return mapProduct(created), nil
}

func (r *ProductRepository) Update(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	updated, err := r.q.UpdateProduct(ctx, sqlc.UpdateProductParams{
		ID:                    uuid.UUID(product.ID),
		UserID:                uuid.UUID(product.UserID),
		CategoryID:            int64(product.CategoryID),
		Name:                  product.Name,
		Description:           product.Description,
		Quantity:              product.Quantity,
		ImageKey:              product.ImageKey,
		ManufacturedAt:        product.ManufacturedAt,
		ExpiresAt:             product.ExpiresAt,
		OpenedAt:              product.OpenedAt,
		ShelfLifeAfterOpening: product.ShelfLifeAfterOpening,
		BaseVersion:           product.Version,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return r.classifyNoRows(ctx, product.UserID, product.ID, "update product")
	}

	if err != nil {
		return nil, productWriteError("update product", err)
	}

	return mapProduct(updated), nil
}

func (r *ProductRepository) Delete(
	ctx context.Context,
	userID domain.UserID,
	id domain.ProductID,
	baseVersion int32,
) (*domain.Product, error) {
	deleted, err := r.q.DeleteProduct(ctx, sqlc.DeleteProductParams{
		UserID:      uuid.UUID(userID),
		ID:          uuid.UUID(id),
		BaseVersion: baseVersion,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return r.classifyNoRows(ctx, userID, id, "delete product")
	}

	if err != nil {
		return nil, fmt.Errorf("delete product: %w", err)
	}

	return mapProduct(deleted), nil
}

func (r *ProductRepository) classifyNoRows(
	ctx context.Context,
	userID domain.UserID,
	id domain.ProductID,
	operation string,
) (*domain.Product, error) {
	current, err := r.Get(ctx, userID, id)

	if errors.Is(err, domain.ErrNotFound) {
		return nil, domain.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("%s: get current product: %w", operation, err)
	}

	return current, domain.ErrVersionConflict
}

func (r *ProductRepository) GetImageKey(ctx context.Context, id domain.ProductID, userID domain.UserID) (*string, error) {
	key, err := r.q.GetImageKey(ctx, sqlc.GetImageKeyParams{
		ID:     uuid.UUID(id),
		UserID: uuid.UUID(userID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get image key: %w", err)
	}

	return key, nil
}

func mapProduct(row sqlc.Product) *domain.Product {
	return &domain.Product{
		ID:       domain.ProductID(row.ID),
		UserID:   domain.UserID(row.UserID),
		Version:  row.Version,
		ImageKey: row.ImageKey,
		ProductData: domain.ProductData{
			Name:                  row.Name,
			CategoryID:            domain.CategoryID(row.CategoryID),
			Description:           row.Description,
			Quantity:              row.Quantity,
			ManufacturedAt:        row.ManufacturedAt,
			ExpiresAt:             row.ExpiresAt,
			OpenedAt:              row.OpenedAt,
			ShelfLifeAfterOpening: row.ShelfLifeAfterOpening,
		},
	}
}

func productWriteError(operation string, err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23503" && postgresError.ConstraintName == "products_category_id_fkey" {
		return domain.NewValidationErr("categoryId", "category not found")
	}
	return fmt.Errorf("%s: %w", operation, err)
}
