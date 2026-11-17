package controllers

import (
	"context"
	"errors"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/middleware"
	"smart-fridge-backend/internal/transport/http/utils/mapper"
	"smart-fridge-backend/internal/usecase/service"

	openapitypes "github.com/oapi-codegen/runtime/types"
)

type ProductController struct {
	service *service.ProductService
}

func NewProductController(s *service.ProductService) *ProductController {
	return &ProductController{
		service: s,
	}
}

func (c *ProductController) ListProducts(ctx context.Context, _ api.ListProductsRequestObject) (api.ListProductsResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	products, err := c.service.List(ctx, domain.UserID(userID))
	if err != nil {
		return nil, err
	}

	items := make([]api.Product, 0, len(products))
	for i := range products {
		items = append(items, newProductOutput(products[i]))
	}

	return api.ListProducts200JSONResponse{
		Items: items,
	}, nil
}

func (c *ProductController) BatchProducts(ctx context.Context, request api.BatchProductsRequestObject) (api.BatchProductsResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	operations := request.Body.Operations
	if len(operations) == 0 {
		return nil, domain.NewValidationErr("operations", "at least one operation is required")
	}

	results := make([]api.ProductBatchResult, 0, len(operations))
	for i, operation := range operations {
		result, err := c.executeOperation(ctx, domain.UserID(userID), operation)
		if err != nil {
			return nil, fmt.Errorf("operation %d: %w", i, err)
		}

		results = append(results, result)
	}

	return api.BatchProducts200JSONResponse{
		Results: results,
	}, nil
}

// executeOperation dispatches the operation to the corresponding service method.
func (c *ProductController) executeOperation(ctx context.Context, userID domain.UserID, op api.ProductBatchOperation) (api.ProductBatchResult, error) {
	opType, err := op.Discriminator()
	if err != nil {
		return api.ProductBatchResult{}, domain.NewValidationErr("type", "missing or invalid operation type")
	}

	switch opType {
	case "create":
		value, err := op.AsProductCreateOperation()
		if err != nil {
			return api.ProductBatchResult{}, domain.NewValidationErr("operation", err.Error())
		}

		product, err := c.service.Create(ctx, userID, dto.CreateProductInput{
			OperationID: value.OperationId,
			ProductID:   domain.ProductID(value.Id),
			Data:        toDomainProductData(value.Product),
			ImageKey:    value.Product.ImageKey,
		},
		)

		return newBatchResult(
			value.OperationId,
			value.Id,
			product,
			err,
		)

	case "update":
		value, err := op.AsProductUpdateOperation()
		if err != nil {
			return api.ProductBatchResult{}, domain.NewValidationErr("operation", err.Error())
		}

		product, err := c.service.Update(ctx, userID, dto.UpdateProductInput{
			OperationID:     value.OperationId,
			ProductID:       domain.ProductID(value.Id),
			ExpectedVersion: value.ExpectedVersion,
			Data:            toDomainProductData(value.Product),
			ImageKey:        value.Product.ImageKey,
		},
		)

		return newBatchResult(
			value.OperationId,
			value.Id,
			product,
			err,
		)

	case "delete":
		value, err := op.AsProductDeleteOperation()
		if err != nil {
			return api.ProductBatchResult{}, domain.NewValidationErr("operation", err.Error())
		}

		product, err := c.service.Delete(ctx, userID, dto.DeleteProductInput{
			OperationID:     value.OperationId,
			ProductID:       domain.ProductID(value.Id),
			ExpectedVersion: value.ExpectedVersion,
		},
		)

		return newBatchResult(
			value.OperationId,
			value.Id,
			product,
			err,
		)

	default:
		return api.ProductBatchResult{}, domain.NewValidationErr("type", "unsupported operation type")
	}
}

// newBatchResult maps service outcomes into API operation results.
func newBatchResult(
	operationID openapitypes.UUID,
	productID openapitypes.UUID,
	product dto.ProductOutput,
	err error,
) (api.ProductBatchResult, error) {
	result := api.ProductBatchResult{
		OperationId: operationID,
		Id:          productID,
	}

	var validationError domain.ValidationError
	switch {
	case errors.Is(err, domain.ErrAlreadyExists):
		result.Error = &api.Error{
			Code:    "already_exists",
			Message: "Product with this ID already exists",
		}

	case errors.Is(err, domain.ErrVersionConflict):
		result.Error = &api.Error{
			Code:    "version_conflict",
			Message: "Product version has changed",
		}

	case errors.Is(err, domain.ErrNotFound):
		result.Error = &api.Error{
			Code:    "not_found",
			Message: "Product not found",
		}

	case errors.As(err, &validationError):
		result.Error = &api.Error{Code: "validation_error", Message: validationError.Error()}

	case err != nil:
		return result, err
	}

	if product.Product != nil {
		result.Product = new(newProductOutput(product))
	}

	return result, nil
}

func toDomainProductData(fields api.ProductFields) domain.ProductData {
	return domain.ProductData{
		Name:                  fields.Name,
		CategoryID:            domain.CategoryID(fields.CategoryId),
		Description:           fields.Description,
		Quantity:              fields.Quantity,
		ManufacturedAt:        mapper.DateToTime(fields.ManufacturedAt),
		ExpiresAt:             fields.ExpiresAt.Time,
		OpenedAt:              mapper.DateToTime(fields.OpenedAt),
		ShelfLifeAfterOpening: fields.ShelfLifeAfterOpening,
	}
}

func newProductOutput(output dto.ProductOutput) api.Product {
	p := output.Product
	out := api.Product{
		Id:                    new(openapitypes.UUID(p.ID)),
		Name:                  p.Name,
		CategoryId:            int64(p.CategoryID),
		Description:           p.Description,
		Quantity:              p.Quantity,
		ImageKey:              p.ImageKey,
		ImageUrl:              output.ImageURL,
		ExpiresAt:             openapitypes.Date{Time: p.ExpiresAt},
		ShelfLifeAfterOpening: p.ShelfLifeAfterOpening,
		Version:               new(p.Version),
	}

	if p.ManufacturedAt != nil {
		out.ManufacturedAt = &openapitypes.Date{
			Time: *p.ManufacturedAt,
		}
	}

	if p.OpenedAt != nil {
		out.OpenedAt = &openapitypes.Date{
			Time: *p.OpenedAt,
		}
	}

	return out
}
