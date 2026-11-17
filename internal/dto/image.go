package dto

import (
	"smart-fridge-backend/internal/domain"
	"time"

	"github.com/google/uuid"
)

type ImageUploadInput struct {
	OperationID uuid.UUID
	ContentType string
}

type ImageUploadTarget struct {
	OperationID     uuid.UUID
	ObjectID        string
	UploadURL       string
	ExpiresAt       time.Time
	RequiredHeaders map[string]string
}

type ImageCompleteInput struct {
	OperationID uuid.UUID
	ObjectID    string
}

type ImageCompleteResult struct {
	OperationID uuid.UUID
	ImageKey    *string
	Err         error
}

type ImageDeleteInput struct {
	OperationID     uuid.UUID
	ProductID       domain.ProductID
	ExpectedVersion int32
}

type ImageDeleteResult struct {
	OperationID uuid.UUID
	ProductID   domain.ProductID
	Product     *domain.Product
	Err         error
}
