package controllers

import (
	"context"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/middleware"
	"smart-fridge-backend/internal/usecase/service"
)

type CategoryController struct{ service *service.CategoryService }

func NewCategoryController(s *service.CategoryService) *CategoryController {
	return &CategoryController{s}
}
func (c *CategoryController) CreateCategory(ctx context.Context, r api.CreateCategoryRequestObject) (api.CreateCategoryResponseObject, error) {
	user, err := middleware.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if r.Body == nil {
		return nil, domain.NewValidationErr("body", "required")
	}
	b := r.Body
	if b.Id != r.Id {
		return nil, domain.NewValidationErr("id", "body and path must match")
	}
	item, err := c.service.Save(ctx, user, domain.Category{ID: b.Id, Name: b.Name}, r.Params.BaseVersion)
	if err != nil {
		return nil, err
	}
	return api.CreateCategory200JSONResponse(api.Category{Id: item.ID, Name: item.Name, Version: &item.Version}), nil
}
func (c *CategoryController) DeleteCategory(ctx context.Context, r api.DeleteCategoryRequestObject) (api.DeleteCategoryResponseObject, error) {
	user, err := middleware.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if err = c.service.Delete(ctx, user, r.Id, r.Params.BaseVersion); err != nil {
		return nil, err
	}
	return api.DeleteCategory204Response{}, nil
}
func (c *CategoryController) ListCategories(ctx context.Context, r api.ListCategoriesRequestObject) (api.ListCategoriesResponseObject, error) {
	user, err := middleware.UserID(ctx)
	if err != nil {
		return nil, err
	}
	limit, offset := page(r.Params.Limit, r.Params.Offset)
	items, err := c.service.List(ctx, user, limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]api.Category, 0, len(items))
	for _, item := range items {
		result = append(result, api.Category{Id: item.ID, Name: item.Name, Version: &item.Version})
	}
	return api.ListCategories200JSONResponse{Items: result}, nil
}
