package controllers

import (
	"context"
	"fmt"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/middleware"
	"smart-fridge-backend/internal/transport/http/utils/header"
	"smart-fridge-backend/internal/usecase/service"

	"github.com/google/uuid"
	openapitypes "github.com/oapi-codegen/runtime/types"
)

type ProductController struct {
	service      *service.ProductService
	maxImageSize int64
}

func NewProductController(s *service.ProductService, cfg *config.Config) *ProductController {
	return &ProductController{service: s, maxImageSize: cfg.Storage.MaxImageSize}
}

func (c *ProductController) UpsertProduct(ctx context.Context, r api.UpsertProductRequestObject) (api.UpsertProductResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	base, err := header.ResolveVersion(r.Params.IfMatch, r.Params.IfNoneMatch)
	if err != nil {
		return nil, err
	}

	p, err := c.service.Save(ctx, dto.ProductInput{
		ProductID:   r.Id,
		UserID:      userID,
		BaseVersion: base,
		Data:        toProductData(*r.Body),
	})
	if err != nil {
		return nil, err
	}

	tag := new(fmt.Sprintf(`"%d"`, p.Version))
	if base == 0 {
		return api.UpsertProduct201JSONResponse{
			Body:    newProductOutput(p),
			Headers: api.UpsertProduct201ResponseHeaders{ETag: tag},
		}, nil
	}
	return api.UpsertProduct200JSONResponse{
		Body:    newProductOutput(p),
		Headers: api.UpsertProduct200ResponseHeaders{ETag: tag},
	}, nil
}

func (c *ProductController) DeleteProduct(ctx context.Context, r api.DeleteProductRequestObject) (api.DeleteProductResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	base, err := header.ResolveVersion(r.Params.IfMatch, nil)
	if err != nil {
		return nil, err
	}

	if err = c.service.Delete(ctx, domain.UserID(userID), domain.ProductID(r.Id), base); err != nil {
		return nil, err
	}
	return api.DeleteProduct204Response{}, nil
}

func (c *ProductController) ListProducts(ctx context.Context, _ api.ListProductsRequestObject) (api.ListProductsResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	items, err := c.service.List(ctx, domain.UserID(userID))
	if err != nil {
		return nil, err
	}

	out := make([]api.Product, 0, len(items))
	for _, item := range items {
		out = append(out, newProductOutput(&item))
	}

	return api.ListProducts200JSONResponse{Items: out}, nil
}

func newProductOutput(p *domain.Product) api.Product {
	out := api.Product{
		Id:                    new(uuid.UUID(p.ID)),
		Name:                  p.Name,
		CategoryId:            uuid.UUID(p.CategoryID),
		Description:           p.Description,
		Quantity:              p.Quantity,
		ImageLink:             p.ImageLink,
		ExpiresAt:             openapitypes.Date{Time: p.ExpiresAt},
		ShelfLifeAfterOpening: p.ShelfLifeAfterOpening,
		Version:               new(p.Version),
	}

	if p.ManufacturedAt != nil {
		out.ManufacturedAt = &openapitypes.Date{Time: *p.ManufacturedAt}
	}
	if p.OpenedAt != nil {
		out.OpenedAt = &openapitypes.Date{Time: *p.OpenedAt}
	}

	return out
}
