package controllers

import (
	openapi_types "github.com/oapi-codegen/runtime/types"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/transport/http/api"
	"time"
)

func page(limit, offset *int) (int32, int32) {
	l, o := 200, 0
	if limit != nil {
		l = *limit
	}
	if offset != nil {
		o = *offset
	}
	if l < 1 || l > 500 || o < 0 || o > 500 {
		return 0, -1
	}
	return int32(l), int32(o)
}
func dateString(d *openapi_types.Date) *string {
	if d == nil {
		return nil
	}
	s := d.Format("2006-01-02")
	return &s
}
func dateValue(s *string) *openapi_types.Date {
	if s == nil {
		return nil
	}
	t, _ := time.Parse("2006-01-02", *s)
	return &openapi_types.Date{Time: t}
}
func productOutput(p domain.Product) api.Product {
	return api.Product{Id: p.ID, Name: p.Name, CategoryId: p.CategoryID, Description: p.Description, Quantity: p.Quantity, ImageLink: p.ImageLink, ManufacturedAt: dateValue(p.ManufacturedAt), ExpiresAt: *dateValue(&p.ExpiresAt), OpenedAt: dateValue(p.OpenedAt), ShelfLifeAfterOpening: p.ShelfLifeAfterOpening, Version: &p.Version}
}
func settingsOutput(s domain.Settings) api.Settings {
	items := make([]api.Indicator, 0, len(s.Indicators))
	for _, i := range s.Indicators {
		items = append(items, api.Indicator{Color: i.Color, MaxDays: i.MaxDays})
	}
	return api.Settings{HasNotifications: &s.HasNotifications, NotificationTime: &s.NotificationTime, Timezone: &s.Timezone, Version: &s.Version, Indicators: &items}
}
