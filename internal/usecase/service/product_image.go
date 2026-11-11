package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"smart-fridge-backend/internal/domain"
	"time"

	"github.com/google/uuid"
)

type ProductImageStorage interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) error
	GetURL(ctx context.Context, key string, expiresAfter time.Duration) (string, error)
	Delete(ctx context.Context, key string) error
}

func (s *ProductService) UploadImage(ctx context.Context, userID domain.UserID, id domain.ProductID, version int32, file io.Reader, contentType string) (int32, error) {
	product, err := s.productForImage(ctx, userID, id, version)
	if err != nil {
		return 0, err
	}
	key := fmt.Sprintf("products/%s/%s/%s", uuid.UUID(userID), uuid.UUID(id), uuid.New())
	if err := s.images.Upload(ctx, key, file, contentType); err != nil {
		slog.WarnContext(ctx, "upload product image failed", "key", key, "error", err)
		s.removeImageObject(ctx, &key)
		return 0, domain.ErrImageStorageFailed
	}
	newVersion, err := s.repo.SetImage(ctx, userID, id, version, &key)
	if errors.Is(err, domain.ErrVersionConflict) {
		s.removeImageObject(ctx, &key)
	}
	if err != nil {
		if !errors.Is(err, domain.ErrVersionConflict) {
			slog.WarnContext(ctx, "product image update outcome unknown", "key", key, "error", err)
		}
		return 0, err
	}
	s.removeImageObject(ctx, product.ImageKey)
	return newVersion, nil
}

func (s *ProductService) DeleteImage(ctx context.Context, userID domain.UserID, id domain.ProductID, version int32) (int32, error) {
	product, err := s.productForImage(ctx, userID, id, version)
	if err != nil {
		return 0, err
	}
	if product.ImageKey == nil {
		return product.Version, nil
	}
	newVersion, err := s.repo.SetImage(ctx, userID, id, version, nil)
	if err != nil {
		return 0, err
	}
	s.removeImageObject(ctx, product.ImageKey)
	return newVersion, nil
}

func (s *ProductService) productForImage(ctx context.Context, userID domain.UserID, id domain.ProductID, version int32) (*domain.Product, error) {
	if s.images == nil {
		return nil, domain.ErrImageStorageUnavailable
	}
	product, err := s.repo.Get(ctx, userID, id)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, domain.ErrVersionConflict
	}
	if err != nil {
		return nil, err
	}
	if product.Version != version {
		return nil, domain.ErrVersionConflict
	}
	return product, nil
}

func (s *ProductService) attachImageURL(ctx context.Context, product *domain.Product) {
	if product.ImageKey == nil {
		return
	}
	if s.images == nil {
		return
	}
	link, err := s.images.GetURL(ctx, *product.ImageKey, s.imageURLTTL)
	if err != nil {
		slog.WarnContext(ctx, "sign product image URL failed", "key", *product.ImageKey, "error", err)
		return
	}
	product.ImageLink = &link
}

func (s *ProductService) removeImageObject(ctx context.Context, key *string) {
	if key == nil {
		return
	}
	if s.images == nil {
		slog.WarnContext(ctx, "cannot remove product image without storage configuration", "key", *key)
		return
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	if err := s.images.Delete(cleanupCtx, *key); err != nil {
		slog.WarnContext(cleanupCtx, "remove product image failed", "key", *key, "error", err)
	}
}
