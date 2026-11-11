package service

import (
	"context"
	"errors"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"time"
)

type ProductRepository interface {
	List(ctx context.Context, userID domain.UserID) ([]domain.Product, error)
	Get(ctx context.Context, userID domain.UserID, id domain.ProductID) (*domain.Product, error)
	Create(ctx context.Context, product *domain.Product) error
	Update(ctx context.Context, product *domain.Product) error
	Delete(ctx context.Context, userID domain.UserID, id domain.ProductID, baseVersion int32) (*string, error)
	SetImage(ctx context.Context, userID domain.UserID, id domain.ProductID, baseVersion int32, key *string) (int32, error)
}

type ProductService struct {
	repo        ProductRepository
	images      ProductImageStorage
	imageURLTTL time.Duration
}

func NewProductService(repo ProductRepository, images ProductImageStorage, imageURLTTL time.Duration) *ProductService {
	return &ProductService{repo: repo, images: images, imageURLTTL: imageURLTTL}
}

func (s *ProductService) List(ctx context.Context, userID domain.UserID) ([]domain.Product, error) {
	products, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}

	for i := range products {
		s.attachImageURL(ctx, &products[i])
	}
	return products, nil
}

func (s *ProductService) Save(ctx context.Context, in dto.ProductInput) (*domain.Product, error) {
	p, err := domain.NewProduct(domain.ProductID(in.ProductID), domain.UserID(in.UserID), in.BaseVersion, in.Data)
	if err != nil {
		return nil, err
	}

	if p.Version == 0 {
		err = s.repo.Create(ctx, p)
	} else {
		err = s.repo.Update(ctx, p)
	}
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			err = domain.ErrVersionConflict
		}
		return nil, fmt.Errorf("save product: %w", err)
	}

	p.UpdateVersion()
	s.attachImageURL(ctx, p)
	return p, nil
}

func (s *ProductService) Delete(ctx context.Context, userID domain.UserID, id domain.ProductID, base int32) error {
	if base < 1 {
		return domain.NewValidationErr("baseVersion", "must be >= 1")
	}
	key, err := s.repo.Delete(ctx, userID, id, base)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	s.removeImageObject(ctx, key)
	return nil
}
