package controllers

import (
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/transport/http/api"
	"time"

	openapitypes "github.com/oapi-codegen/runtime/types"
)

func dateToTime(d *openapitypes.Date) *time.Time {
	if d == nil {
		return nil
	}
	return &d.Time
}

func toProductData(b api.Product) domain.ProductData {
	return domain.ProductData{
		Name:                  b.Name,
		CategoryID:            domain.CategoryID(b.CategoryId),
		Description:           b.Description,
		Quantity:              b.Quantity,
		ManufacturedAt:        dateToTime(b.ManufacturedAt),
		ExpiresAt:             b.ExpiresAt.Time,
		OpenedAt:              dateToTime(b.OpenedAt),
		ShelfLifeAfterOpening: b.ShelfLifeAfterOpening,
	}
}
