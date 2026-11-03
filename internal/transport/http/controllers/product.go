package controllers

import (
	"context"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/middleware"
	"smart-fridge-backend/internal/usecase/service"
)

type ProductController struct{ service *service.ProductService }

func NewProductController(s *service.ProductService) *ProductController { return &ProductController{s} }
func (c *ProductController) UpdateProduct(ctx context.Context, r api.UpdateProductRequestObject) (api.UpdateProductResponseObject, error) {
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
	item, err := c.service.Save(ctx, user, domain.Product{ID: b.Id, Name: b.Name, CategoryID: b.CategoryId, Description: b.Description, Quantity: b.Quantity, ImageLink: b.ImageLink, ManufacturedAt: dateString(b.ManufacturedAt), ExpiresAt: b.ExpiresAt.Format("2006-01-02"), OpenedAt: dateString(b.OpenedAt), ShelfLifeAfterOpening: b.ShelfLifeAfterOpening}, r.Params.BaseVersion)
	if err != nil {
		return nil, err
	}
	return api.UpdateProduct200JSONResponse(productOutput(item)), nil
}
func (c *ProductController) DeleteProduct(ctx context.Context, r api.DeleteProductRequestObject) (api.DeleteProductResponseObject, error) {
	user, err := middleware.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if err = c.service.Delete(ctx, user, r.Id, r.Params.BaseVersion); err != nil {
		return nil, err
	}
	return api.DeleteProduct204Response{}, nil
}
func (c *ProductController) ListProducts(ctx context.Context, r api.ListProductsRequestObject) (api.ListProductsResponseObject, error) {
	user, err := middleware.UserID(ctx)
	if err != nil {
		return nil, err
	}
	limit, offset := page(r.Params.Limit, r.Params.Offset)
	items, err := c.service.List(ctx, user, limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]api.Product, 0, len(items))
	for _, item := range items {
		result = append(result, productOutput(item))
	}
	return api.ListProducts200JSONResponse{Items: result}, nil
}
