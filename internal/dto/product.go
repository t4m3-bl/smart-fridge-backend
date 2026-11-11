package dto

import (
	"smart-fridge-backend/internal/domain"

	"github.com/google/uuid"
)

type ProductInput struct {
	ProductID   uuid.UUID
	UserID      uuid.UUID
	Data        domain.ProductData
	BaseVersion int32
}
