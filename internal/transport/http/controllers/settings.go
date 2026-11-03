package controllers

import (
	"context"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/middleware"
	"smart-fridge-backend/internal/usecase/service"
)

type SettingController struct{ service *service.SettingsService }

func NewSettingController(s *service.SettingsService) *SettingController {
	return &SettingController{s}
}
func (c *SettingController) GetSettings(ctx context.Context, _ api.GetSettingsRequestObject) (api.GetSettingsResponseObject, error) {
	user, err := middleware.UserID(ctx)
	if err != nil {
		return nil, err
	}
	s, err := c.service.Get(ctx, user)
	if err != nil {
		return nil, err
	}
	return api.GetSettings200JSONResponse{SettingsSuccessJSONResponse: api.SettingsSuccessJSONResponse(settingsOutput(s))}, nil
}
func (c *SettingController) PatchSettings(ctx context.Context, r api.PatchSettingsRequestObject) (api.PatchSettingsResponseObject, error) {
	user, err := middleware.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if r.Body == nil {
		return nil, domain.NewValidationErr("body", "required")
	}
	b := r.Body
	p := domain.SettingsPatch{HasNotifications: b.HasNotifications, NotificationTime: b.NotificationTime, Timezone: b.Timezone}
	if b.Indicators != nil {
		items := make([]domain.Indicator, 0, len(*b.Indicators))
		for _, i := range *b.Indicators {
			items = append(items, domain.Indicator{Color: i.Color, MaxDays: i.MaxDays})
		}
		p.Indicators = &items
	}
	s, err := c.service.Patch(ctx, user, p, r.Params.BaseVersion)
	if err != nil {
		return nil, err
	}
	return api.PatchSettings200JSONResponse{SettingsSuccessJSONResponse: api.SettingsSuccessJSONResponse(settingsOutput(s))}, nil
}
