package controllers

import (
	"context"
	"smart-fridge-backend/internal/transport/http/api"
)

type SettingController struct{}

func (c *SettingController) GetSettings(ctx context.Context, request api.GetSettingsRequestObject) (api.GetSettingsResponseObject, error) {
	_ = ctx
	_ = request
	panic("implement me")
}

func (c *SettingController) PatchSettings(ctx context.Context, request api.PatchSettingsRequestObject) (api.PatchSettingsResponseObject, error) {
	_ = ctx
	_ = request
	panic("implement me")
}
