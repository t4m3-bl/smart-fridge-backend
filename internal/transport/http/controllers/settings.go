package controllers

import (
	"context"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/middleware"
	"smart-fridge-backend/internal/usecase/service"
	"time"
)

type SettingsController struct {
	settings *service.SettingsService
}

func NewSettingController(s *service.SettingsService) *SettingsController {
	return &SettingsController{settings: s}
}

func (c *SettingsController) GetSettings(ctx context.Context, _ api.GetSettingsRequestObject) (api.GetSettingsResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	s, err := c.settings.Get(ctx, domain.UserID(userID))
	if err != nil {
		return nil, err
	}

	return api.GetSettings200JSONResponse{
		SettingsSuccessJSONResponse: newSettingsSuccessResponse(s),
	}, nil
}

func (c *SettingsController) PatchSettings(ctx context.Context, r api.PatchSettingsRequestObject) (api.PatchSettingsResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	changes, err := toSettingsChanges(*r.Body)
	if err != nil {
		return nil, err
	}

	s, err := c.settings.Patch(ctx, domain.UserID(userID), changes)
	if err != nil {
		return nil, err
	}

	return api.PatchSettings200JSONResponse{
		SettingsSuccessJSONResponse: newSettingsSuccessResponse(s),
	}, nil
}

func toSettingsChanges(b api.PatchSettingsJSONRequestBody) (domain.SettingsChanges, error) {
	c := domain.SettingsChanges{HasNotifications: b.HasNotifications, Timezone: b.Timezone}

	if b.NotificationTime != nil {
		t, err := time.Parse(time.TimeOnly, *b.NotificationTime)
		if err != nil {
			return domain.SettingsChanges{}, domain.NewValidationErr("notificationTime", "must be HH:MM:SS")
		}

		clock := time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute + time.Duration(t.Second())*time.Second
		c.NotificationTime = &clock
	}

	if b.Indicators != nil {
		list := make([]domain.Indicator, 0, len(*b.Indicators))
		for _, i := range *b.Indicators {
			list = append(list, domain.Indicator{Color: i.Color, MaxDays: i.MaxDays})
		}
		c.Indicators = &list
	}

	return c, nil
}

func newSettingsSuccessResponse(s *domain.Settings) api.SettingsSuccessJSONResponse {
	inds := make([]api.Indicator, 0, len(s.Indicators))
	for _, i := range s.Indicators {
		inds = append(inds, api.Indicator{Color: i.Color, MaxDays: i.MaxDays})
	}
	return api.SettingsSuccessJSONResponse{
		HasNotifications: &s.HasNotifications,
		NotificationTime: new(time.Time{}.Add(s.NotificationTime).Format(time.TimeOnly)),
		Timezone:         &s.Timezone,
		Indicators:       &inds,
	}
}
