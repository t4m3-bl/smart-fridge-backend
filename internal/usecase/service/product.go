package service

import (
	"context"
	"fmt"
	"log/slog"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"strings"
	"time"

	"github.com/google/uuid"
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

	outputs := make([]dto.ProductOutput, 0, len(products))
	for i := range products {
		outputs = append(outputs, s.toOutput(ctx, &products[i]))
	}
	return outputs, nil
}

func (s *ProductService) Create(ctx context.Context, userID domain.UserID, in dto.CreateProductInput) (dto.ProductOutput, error) {
	product, err := domain.NewProduct(in.ProductID, userID, 0, in.ImageKey, in.Data)
	if err != nil {
		return dto.ProductOutput{}, err
	}

	if err := s.validateImageKey(ctx, userID, in.ImageKey); err != nil {
		return dto.ProductOutput{}, err
	}

	created, err := s.product.Create(ctx, product)
	if err != nil {
		return s.toOutput(ctx, created), fmt.Errorf("create product: %w", err)
	}

	return s.toOutput(ctx, created), nil
}

func (s *ProductService) Update(ctx context.Context, userID domain.UserID, in dto.UpdateProductInput) (dto.ProductOutput, error) {
	product, err := domain.NewProduct(in.ProductID, userID, in.ExpectedVersion, in.ImageKey, in.Data)
	if err != nil {
		return dto.ProductOutput{}, err
	}

	oldImageKey, err := s.product.GetImageKey(ctx, in.ProductID, userID)
	if err != nil {
		return dto.ProductOutput{}, fmt.Errorf("update product: %w", err)
	}

	if err := s.validateImageKey(ctx, userID, in.ImageKey); err != nil {
		return dto.ProductOutput{}, err
	}

	updated, err := s.product.Update(ctx, product)
	if err != nil {
		return s.toOutput(ctx, updated), fmt.Errorf("update product: %w", err)
	}

	if oldImageKey != nil && (updated.ImageKey == nil || *oldImageKey != *updated.ImageKey) {
		s.cleanupImage(ctx, *oldImageKey)
	}

	return s.toOutput(ctx, updated), nil
}

func (s *ProductService) Delete(ctx context.Context, userID domain.UserID, in dto.DeleteProductInput) (dto.ProductOutput, error) {
	if in.ExpectedVersion < 1 {
		return dto.ProductOutput{}, domain.NewValidationErr("expectedVersion", "must be >= 1")
	}

	deleted, err := s.product.Delete(ctx, userID, in.ProductID, in.ExpectedVersion)
	if err != nil {
		return s.toOutput(ctx, deleted), fmt.Errorf("delete product: %w", err)
	}

	if deleted.ImageKey != nil {
		s.cleanupImage(ctx, *deleted.ImageKey)
	}

	return dto.ProductOutput{Product: deleted}, nil
}

func (s *ProductService) validateImageKey(
	ctx context.Context,
	userID domain.UserID,
	imageKey *string,
) error {
	if imageKey == nil {
		return nil
	}

	prefix := fmt.Sprintf("products/%s/images/", uuid.UUID(userID))
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

func (s *ProductService) toOutput(ctx context.Context, product *domain.Product) dto.ProductOutput {
	output := dto.ProductOutput{Product: product}
	if product == nil || product.ImageKey == nil {
		return output
	}
	link, err := s.storage.PresignGet(ctx, *product.ImageKey, 15*time.Minute)
	if err != nil {
		slog.WarnContext(ctx, "generate product image URL failed", "error", err)
		return output
	}
	output.ImageURL = &link
	return output
}

func (s *ProductService) cleanupImage(ctx context.Context, key string) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	if err := s.storage.Delete(cleanupCtx, key); err != nil {
		slog.WarnContext(cleanupCtx, "delete unused product image failed", "key", key, "error", err)
	}
}
