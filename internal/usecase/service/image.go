package service

import (
	"context"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"time"

	"github.com/google/uuid"
)

type ImageStorage interface {
	PresignPut(ctx context.Context, key, contentType string, uploadTTL time.Duration) (string, map[string]string, error)
	ObjectExists(ctx context.Context, key string) (bool, error)
	Delete(ctx context.Context, key string) error
}

type ProductImageService struct {
	storage   ImageStorage
	uploadTTL time.Duration
}

func NewProductImageService(
	storage ImageStorage,
	uploadTTL time.Duration,
) *ProductImageService {
	return &ProductImageService{
		storage:   storage,
		uploadTTL: uploadTTL,
	}
}

// PrepareUploads creates presigned PUT URLs for product images.
// It does not create or update products.
func (s *ProductImageService) PrepareUploads(ctx context.Context, userID domain.UserID, inputs []dto.ImageUploadInput) ([]dto.ImageUploadTarget, error) {
	targets := make([]dto.ImageUploadTarget, 0, len(inputs))

	for _, input := range inputs {
		objectID := makeUploadObjectID(userID, input.OperationID)
		key := productImageKey(userID, input.OperationID, objectID)

		uploadURL, headers, err := s.storage.PresignPut(
			ctx,
			key,
			input.ContentType,
			s.uploadTTL,
		)
		if err != nil {
			return nil, fmt.Errorf("presign product image upload: %w", err)
		}

		targets = append(targets, dto.ImageUploadTarget{
			OperationID:     input.OperationID,
			ObjectID:        objectID.String(),
			UploadURL:       uploadURL,
			ExpiresAt:       time.Now().UTC().Add(s.uploadTTL),
			RequiredHeaders: headers,
		})
	}

	return targets, nil
}

// CompleteUploads verifies uploaded objects and returns their ImageKeys.
// It does not modify products.
func (s *ProductImageService) CompleteUploads(
	ctx context.Context,
	userID domain.UserID,
	inputs []dto.ImageCompleteInput,
) ([]dto.ImageCompleteResult, error) {
	results := make([]dto.ImageCompleteResult, 0, len(inputs))

	for _, input := range inputs {
		result := dto.ImageCompleteResult{
			OperationID: input.OperationID,
		}

		objectID := makeUploadObjectID(userID, input.OperationID)

		if input.ObjectID != objectID.String() {
			result.Err = domain.ErrNotFound
			results = append(results, result)
			continue
		}

		key := productImageKey(userID, input.OperationID, objectID)

		exists, err := s.storage.ObjectExists(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("inspect uploaded product image: %w", err)
		}

		if !exists {
			result.Err = domain.ErrNotFound
			results = append(results, result)
			continue
		}

		result.ImageKey = &key
		results = append(results, result)
	}

	return results, nil
}

func productImageKey(
	userID domain.UserID,
	operationID uuid.UUID,
	objectID uuid.UUID,
) string {
	return fmt.Sprintf(
		"%s%s/%s",
		fmt.Sprintf("products/%s/images/", uuid.UUID(userID)),
		operationID,
		objectID,
	)
}

func makeUploadObjectID(userID domain.UserID, operationID uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(
		uuid.UUID(userID),
		[]byte(operationID.String()),
	)
}
