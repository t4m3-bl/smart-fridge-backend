package controllers

import (
	"context"
	"smart-fridge-backend/internal/transport/http/api"
)

type ProductController struct{}

func (c *ProductController) UpdateProduct(ctx context.Context, request api.UpdateProductRequestObject) (api.UpdateProductResponseObject, error) {
	_ = ctx
	_ = request
	panic("implement me")
}

func (c *ProductController) DeleteProduct(ctx context.Context, request api.DeleteProductRequestObject) (api.DeleteProductResponseObject, error) {
	_ = ctx
	_ = request
	panic("implement me")
}

func (c *ProductController) ListProducts(ctx context.Context, request api.ListProductsRequestObject) (api.ListProductsResponseObject, error) {
	_ = ctx
	_ = request
	panic("implement me")
}
