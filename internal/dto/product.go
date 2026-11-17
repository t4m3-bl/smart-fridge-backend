package dto

import (
	"smart-fridge-backend/internal/domain"

	"github.com/google/uuid"
)

type CreateProductInput struct {
	OperationID uuid.UUID
	ProductID   domain.ProductID
	Data        domain.ProductData
	ImageKey    *string
}

type UpdateProductInput struct {
	OperationID     uuid.UUID
	ProductID       domain.ProductID
	ExpectedVersion int32
	Data            domain.ProductData
	ImageKey        *string
}

type DeleteProductInput struct {
	OperationID     uuid.UUID
	ProductID       domain.ProductID
	ExpectedVersion int32
}

type ProductOutput struct {
	Product  *domain.Product
	ImageURL *string
}
