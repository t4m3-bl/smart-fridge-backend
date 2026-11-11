package controllers

import (
	"context"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/middleware"
	"smart-fridge-backend/internal/transport/http/utils/header"
	"smart-fridge-backend/internal/usecase/service"

	"github.com/google/uuid"
)

type CategoryController struct{ service *service.CategoryService }

func NewCategoryController(s *service.CategoryService) *CategoryController {
	return &CategoryController{s}
}

func (c *CategoryController) UpsertCategory(ctx context.Context, r api.UpsertCategoryRequestObject) (api.UpsertCategoryResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	base, err := header.ResolveVersion(r.Params.IfMatch, r.Params.IfNoneMatch)
	if err != nil {
		return nil, err
	}

	cat, err := c.service.Save(ctx, dto.CategorySaveInput{
		CategoryID:  r.Id,
		UserID:      userID,
		BaseVersion: base,
		Name:        r.Body.Name,
	})
	if err != nil {
		return nil, err
	}

	tag := new(fmt.Sprintf(`"%d"`, cat.Version))
	if base == 0 {
		return api.UpsertCategory201JSONResponse{
			Body:    newCategoryOutput(cat),
			Headers: api.UpsertCategory201ResponseHeaders{ETag: tag},
		}, nil
	}
	return api.UpsertCategory200JSONResponse{
		Body:    newCategoryOutput(cat),
		Headers: api.UpsertCategory200ResponseHeaders{ETag: tag},
	}, nil
}

func (c *CategoryController) DeleteCategory(ctx context.Context, r api.DeleteCategoryRequestObject) (api.DeleteCategoryResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)

	base, err := header.GetVersion(*r.Params.IfMatch)
	if err != nil {
		return nil, err
	}

	if err = c.service.Delete(ctx, dto.CategoryDeleteInput{
		CategoryID:  r.Id,
		UserID:      userID,
		BaseVersion: base,
	}); err != nil {
		return nil, err
	}

	return api.DeleteCategory204Response{}, nil
}

func (c *CategoryController) ListCategories(ctx context.Context, _ api.ListCategoriesRequestObject) (api.ListCategoriesResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	items, err := c.service.List(ctx, domain.UserID(userID))
	if err != nil {
		return nil, err
	}

	out := make([]api.Category, 0, len(items))
	for _, item := range items {
		out = append(out, newCategoryOutput(&item))
	}
	return api.ListCategories200JSONResponse{Items: out}, nil
}

func newCategoryOutput(c *domain.Category) api.Category {
	return api.Category{
		Id:      new(uuid.UUID(c.ID)),
		Name:    c.Name,
		Version: new(c.Version),
	}
}
