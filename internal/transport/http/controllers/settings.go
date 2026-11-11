package controllers

import (
	"context"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/middleware"
	"smart-fridge-backend/internal/transport/http/utils/header"
	"smart-fridge-backend/internal/usecase/service"
	"time"
)

type SettingsController struct {
	service *service.SettingsService
}

func NewSettingController(s *service.SettingsService) *SettingsController {
	return &SettingsController{s}
}

func (c *SettingsController) GetSettings(ctx context.Context, _ api.GetSettingsRequestObject) (api.GetSettingsResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	s, err := c.service.Get(ctx, domain.UserID(userID))
	if err != nil {
		return nil, err
	}

	tag := new(fmt.Sprintf(`"%d"`, s.Version))
	return api.GetSettings200JSONResponse{
		SettingsSuccessJSONResponse: api.SettingsSuccessJSONResponse{
			Body:    newSettingsOutput(s),
			Headers: api.SettingsSuccessResponseHeaders{ETag: tag},
		},
	}, nil
}

func (c *SettingsController) PatchSettings(ctx context.Context, r api.PatchSettingsRequestObject) (api.PatchSettingsResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	base, err := header.GetVersion(*r.Params.IfMatch)
	if err != nil {
		return nil, err
	}

	changes, err := toSettingsChanges(*r.Body)
	if err != nil {
		return nil, err
	}

	s, err := c.service.Patch(ctx, domain.UserID(userID), base, changes)
	if err != nil {
		return nil, err
	}

	tag := new(fmt.Sprintf(`"%d"`, s.Version))
	return api.PatchSettings200JSONResponse{
		SettingsSuccessJSONResponse: api.SettingsSuccessJSONResponse{
			Body:    newSettingsOutput(s),
			Headers: api.SettingsSuccessResponseHeaders{ETag: tag},
		},
	}, nil
}

func toSettingsChanges(b api.PatchSettingsJSONRequestBody) (domain.SettingsChanges, error) {
	c := domain.SettingsChanges{HasNotifications: b.HasNotifications, Timezone: b.Timezone}

	if b.NotificationTime != nil {
		t, err := time.Parse(time.TimeOnly, *b.NotificationTime)
		if err != nil {
			return domain.SettingsChanges{}, domain.NewValidationErr("notificationTime", "must be HH:MM:SS")
		}

		c.NotificationTime = new(t.Sub(time.Time{}))
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

func newSettingsOutput(s *domain.Settings) api.Settings {
	inds := make([]api.Indicator, 0, len(s.Indicators))
	for _, i := range s.Indicators {
		inds = append(inds, api.Indicator{Color: i.Color, MaxDays: i.MaxDays})
	}
	return api.Settings{
		HasNotifications: &s.HasNotifications,
		NotificationTime: new(time.Time{}.Add(s.NotificationTime).Format(time.TimeOnly)),
		Timezone:         &s.Timezone,
		Indicators:       &inds,
		Version:          &s.Version,
	}
}
