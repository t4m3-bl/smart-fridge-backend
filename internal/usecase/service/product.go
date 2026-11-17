package service

import (
	"context"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"strings"
	"time"
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

func (s *ProductService) List(ctx context.Context, userID domain.UserID) ([]dto.ProductOutput, error) {
	products, err := s.product.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}

	var productOutputs []dto.ProductOutput
	for prod, _ := range products {
		prs.toOutput(ctx, created)
	}

	return products, nil
}

func (s *ProductService) Create(ctx context.Context, userID domain.UserID, in dto.CreateProductInput) (dto.ProductOutput, error) {
	product, err := domain.NewProduct(in.ProductID, userID, 0, in.ImageKey, in.Data)
	if err != nil {
		return nil, err
	}

	if err := s.validateImageKey(ctx, userID, in.ImageKey); err != nil {
		return nil, err
	}

	created, err := s.product.Create(ctx, product)
	if err != nil {
		return s.toOutput(ctx, created)
	}

	return s.toOutput(ctx, created)
}

func (s *ProductService) Update(ctx context.Context, userID domain.UserID, in dto.UpdateProductInput) (dto.ProductOutput, error) {
	product, err := domain.NewProduct(in.ProductID, userID, in.ExpectedVersion, in.ImageKey, in.Data)
	if err != nil {
		return nil, err
	}

	oldImageKey, err := s.product.GetImageKey(ctx, in.ProductID, userID)
	if err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	if err := s.validateImageKey(ctx, userID, in.ImageKey); err != nil {
		return nil, err
	}

	updated, err := s.product.Update(ctx, product)
	if err != nil {
		return s.toOutput(ctx, updated)
	}

	if oldImageKey != nil && (updated.ImageKey == nil || *oldImageKey != *updated.ImageKey) {
		if err := s.storage.Delete(ctx, *oldImageKey); err != nil {
			return s.toOutput(ctx, updated)
		}
	}

	return s.toOutput(ctx, updated)
}

func (s *ProductService) Delete(ctx context.Context, userID domain.UserID, in dto.DeleteProductInput) (dto.ProductOutput, error) {
	if in.ExpectedVersion < 1 {
		return nil, domain.NewValidationErr("expectedVersion", "must be >= 1")
	}

	deleted, err := s.product.Delete(ctx, userID, in.ProductID, in.ExpectedVersion)
	if err != nil {
		return deleted, fmt.Errorf("delete product: %w", err)
	}

	if deleted.ImageKey != nil {
		if err := s.storage.Delete(ctx, *deleted.ImageKey); err != nil {
			return s.toOutput(ctx, deleted)
		}
	}

	return s.toOutput(ctx, deleted)
}

func (s *ProductService) validateImageKey(
	ctx context.Context,
	userID domain.UserID,
	imageKey *string,
) error {
	if imageKey == nil {
		return nil
	}

	prefix := fmt.Sprintf("products/%s/images/", userID)
	if !strings.HasPrefix(*imageKey, prefix) {
		return domain.ErrNotFound
	}

	exists, err := s.storage.ObjectExists(ctx, *imageKey)
	if err != nil {
		return fmt.Errorf("validate product image: %w", err)
	}

	if !exists {
		return domain.ErrNotFound
	}

	return nil
}

func (s *ProductService) toOutput(
	ctx context.Context,
	product *domain.Product
) (dto.ProductOutput, error) {
	output := dto.ProductOutput{Product: product}

	if product.ImageKey == nil {
		return output, nil
	}

	imageURL, err := s.storage.PresignGet(
		ctx,
		*product.ImageKey,
		15*time.Minute,
	)
	if err != nil {
		return dto.ProductOutput{}, fmt.Errorf("generate product image URL: %w", err)
	}

	output.ImageURL = &imageURL
	return output, nil
}
