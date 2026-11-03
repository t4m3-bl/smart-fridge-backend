package controllers

import (
	"context"
	"smart-fridge-backend/internal/transport/http/api"
)

type CategoryController struct{}

func (c *CategoryController) ListCategories(ctx context.Context, request api.ListCategoriesRequestObject) (api.ListCategoriesResponseObject, error) {
	_ = ctx
	_ = request
	panic("implement me")
}

func (c *CategoryController) DeleteCategory(ctx context.Context, request api.DeleteCategoryRequestObject) (api.DeleteCategoryResponseObject, error) {
	_ = ctx
	_ = request
	panic("implement me")
}

func (c *CategoryController) CreateCategory(ctx context.Context, request api.CreateCategoryRequestObject) (api.CreateCategoryResponseObject, error) {
	_ = ctx
	_ = request
	panic("implement me")
}
