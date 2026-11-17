package service

import (
	"context"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
)

type ProductRepository interface {
	List(ctx context.Context, userID domain.UserID) ([]domain.Product, error)
	Create(ctx context.Context, product *domain.Product) (*domain.Product, error)
	Update(ctx context.Context, product *domain.Product) (*domain.Product, error)
	Delete(ctx context.Context, userID domain.UserID, id domain.ProductID, expectedVersion int32) (*domain.Product, error)
	GetImageKey(ctx context.Context, id domain.ProductID, userID domain.UserID) (*string, error)
}

type ProductService struct {
	product ProductRepository
	storage ImageStorage
}

func NewProductService(repo ProductRepository, storage ImageStorage) *ProductService {
	return &ProductService{
		product: repo,
		storage: storage,
	}
}

func (s *ProductService) List(ctx context.Context, userID domain.UserID) ([]domain.Product, error) {
	products, err := s.product.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}

	return products, nil
}

func (s *ProductService) Create(ctx context.Context, userID domain.UserID, in dto.CreateProductInput) (*domain.Product, error) {
	product, err := domain.NewProduct(in.ProductID, userID, 0, in.ImageKey, in.Data)
	if err != nil {
		return nil, err
	}

	created, err := s.product.Create(ctx, product)
	if err != nil {
		return created, fmt.Errorf("create product: %w", err)
	}

	return created, nil
}

func (s *ProductService) Update(ctx context.Context, userID domain.UserID, in dto.UpdateProductInput) (*domain.Product, error) {
	product, err := domain.NewProduct(in.ProductID, userID, in.ExpectedVersion, in.ImageKey, in.Data)
	if err != nil {
		return nil, err
	}

	oldImageKey, err := s.product.GetImageKey(ctx, in.ProductID, userID)
	if err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	updated, err := s.product.Update(ctx, product)
	if err != nil {
		return updated, fmt.Errorf("update product: %w", err)
	}

	if oldImageKey != nil && updated.ImageKey == nil {
		if err := s.storage.Delete(ctx, *oldImageKey); err != nil {
			return updated, fmt.Errorf("delete product: %w", err)
		}
	}

	return updated, nil
}

func (s *ProductService) Delete(ctx context.Context, userID domain.UserID, in dto.DeleteProductInput) (*domain.Product, error) {
	if in.ExpectedVersion < 1 {
		return nil, domain.NewValidationErr("expectedVersion", "must be >= 1")
	}

	deleted, err := s.product.Delete(ctx, userID, in.ProductID, in.ExpectedVersion)
	if err != nil {
		return deleted, fmt.Errorf("delete product: %w", err)
	}

	if deleted.ImageKey != nil {
		if err := s.storage.Delete(ctx, *deleted.ImageKey); err != nil {
			return deleted, fmt.Errorf("delete product: %w", err)
		}
	}

	return deleted, nil
}
