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

func toProductData(b api.Product) (domain.ProductData, error) {
	var img *domain.ImageLink
	if b.ImageLink != nil {
		l, err := domain.NewImageLink(*b.ImageLink)
		if err != nil {
			return domain.ProductData{}, err
		}
		img = &l
	}

	return domain.ProductData{
		Name:                  b.Name,
		CategoryID:            domain.CategoryID(b.CategoryId),
		Description:           b.Description,
		Quantity:              b.Quantity,
		ImageLink:             img,
		ManufacturedAt:        dateToTime(b.ManufacturedAt),
		ExpiresAt:             b.ExpiresAt.Time,
		OpenedAt:              dateToTime(b.OpenedAt),
		ShelfLifeAfterOpening: b.ShelfLifeAfterOpening,
	}, nil
}
