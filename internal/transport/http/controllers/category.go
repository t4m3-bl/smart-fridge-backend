package controllers

import (
	"context"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/transport/http/api"
)

type CategoryRepository interface {
	List(ctx context.Context) ([]domain.Category, error)
}

type CategoryController struct {
	repo CategoryRepository
}

func NewCategoryController(repo CategoryRepository) *CategoryController {
	return &CategoryController{repo: repo}
}

// ListCategories returns a list of all available categories.
func (c *CategoryController) ListCategories(ctx context.Context, _ api.ListCategoriesRequestObject) (api.ListCategoriesResponseObject, error) {
	items, err := c.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]api.Category, 0, len(items))
	for _, item := range items {
		out = append(out, api.Category{
			Id:   int64(item.ID),
			Name: item.Name,
		})
	}

	return api.ListCategories200JSONResponse{Items: out}, nil
}
