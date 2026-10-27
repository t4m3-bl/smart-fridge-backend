package controllers

import (
	"context"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/middleware"
	"smart-fridge-backend/internal/usecase/service"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

type SyncController struct {
	sync *service.SyncService
}

func NewSyncController(sync *service.SyncService) *SyncController {
	return &SyncController{sync: sync}
}

func (c *SyncController) Push(ctx context.Context, request api.PushRequestObject) (api.PushResponseObject, error) {
	userID, err := middleware.UserID(ctx)
	if err != nil {
		return nil, err
	}

	if request.Body == nil {
		return nil, domain.NewValidationErr("body", "required")
	}

	mutations := make([]domain.Mutation, 0, len(request.Body.Mutations))
	for i, raw := range request.Body.Mutations {
		m, err := toDomainMutation(raw)
		if err != nil {
			return nil, domain.NewValidationErr(fmt.Sprintf("mutations[%d]", i), err.Error())
		}
		mutations = append(mutations, m)
	}

	results, err := c.sync.Push(ctx, userID, mutations)
	if err != nil {
		return nil, err
	}

	response := api.Push200JSONResponse{Results: make([]api.MutationResult, 0, len(results))}
	for _, r := range results {
		item, err := toAPIMutationResult(r)
		if err != nil {
			return nil, err
		}
		response.Results = append(response.Results, item)
	}

	return response, nil
}

func (c *SyncController) Pull(ctx context.Context, request api.PullRequestObject) (api.PullResponseObject, error) {
	userID, err := middleware.UserID(ctx)
	if err != nil {
		return nil, err
	}

	page, err := c.sync.Pull(ctx, userID, request.Params.Since, request.Params.Limit)
	if err != nil {
		return nil, err
	}

	response := api.Pull200JSONResponse{
		Mutations: make([]api.AppliedMutation, 0, len(page.Mutations)),
		Cursor:    page.Cursor,
		HasMore:   page.HasMore,
	}
	for _, m := range page.Mutations {
		item := api.AppliedMutation{
			Seq:        m.Seq,
			MutationId: openapi_types.UUID(m.ID),
			Type:       api.MutationType(m.Type),
			ProductId:  openapi_types.UUID(m.ProductID),
			CreatedAt:  m.CreatedAt,
		}
		if m.Data != nil {
			item.Data = new(toAPIProductData(m.Data))
		}
		response.Mutations = append(response.Mutations, item)
	}

	return response, nil
}

func toDomainMutation(raw api.Mutation) (domain.Mutation, error) {
	value, err := raw.ValueByDiscriminator()
	if err != nil {
		return domain.Mutation{}, err
	}

	switch v := value.(type) {
	case api.CreateProductMutation:
		return domain.Mutation{
			ID:        domain.MutationID(v.MutationId),
			Type:      domain.MutationCreateProduct,
			ProductID: domain.ProductID(v.ProductId),
			Data:      toDomainProductData(v.Data),
		}, nil
	case api.UpdateProductMutation:
		return domain.Mutation{
			ID:        domain.MutationID(v.MutationId),
			Type:      domain.MutationUpdateProduct,
			ProductID: domain.ProductID(v.ProductId),
			Data:      toDomainProductData(v.Data),
		}, nil
	case api.DeleteProductMutation:
		return domain.Mutation{
			ID:        domain.MutationID(v.MutationId),
			Type:      domain.MutationDeleteProduct,
			ProductID: domain.ProductID(v.ProductId),
		}, nil
	default:
		return domain.Mutation{}, fmt.Errorf("unknown mutation %T", value)
	}
}

func toAPIMutationResult(r domain.MutationResult) (api.MutationResult, error) {
	var item api.MutationResult

	switch r.Status {
	case domain.MutationApplied:
		err := item.FromMutationApplied(api.MutationApplied{
			MutationId: openapi_types.UUID(r.MutationID),
			Seq:        r.Seq,
		})
		return item, err
	case domain.MutationRejected:
		err := item.FromMutationRejected(api.MutationRejected{
			MutationId: openapi_types.UUID(r.MutationID),
			Reason:     api.MutationRejectedReason(r.Reason),
			Message:    r.Message,
		})
		return item, err
	default:
		return item, fmt.Errorf("unknown mutation status %q", r.Status)
	}
}

func toDomainProductData(d api.ProductData) *domain.ProductData {
	data := &domain.ProductData{
		Name:                  d.Name,
		Description:           d.Description,
		Quantity:              d.Quantity,
		ImageLink:             d.ImageLink,
		ManufacturedAt:        fromAPIDate(d.ManufacturedAt),
		ExpiresAt:             d.ExpiresAt.Time,
		OpenedAt:              fromAPIDate(d.OpenedAt),
		ShelfLifeAfterOpening: d.ShelfLifeAfterOpening,
	}
	if d.CategoryId != nil {
		data.CategoryID = new(domain.CategoryID(*d.CategoryId))
	}
	return data
}

func toAPIProductData(d *domain.ProductData) api.ProductData {
	data := api.ProductData{
		Name:                  d.Name,
		Description:           d.Description,
		Quantity:              d.Quantity,
		ImageLink:             d.ImageLink,
		ManufacturedAt:        toAPIDate(d.ManufacturedAt),
		ExpiresAt:             openapi_types.Date{Time: d.ExpiresAt},
		OpenedAt:              toAPIDate(d.OpenedAt),
		ShelfLifeAfterOpening: d.ShelfLifeAfterOpening,
	}
	if d.CategoryID != nil {
		data.CategoryId = new(openapi_types.UUID(*d.CategoryID))
	}
	return data
}

func fromAPIDate(d *openapi_types.Date) *time.Time {
	if d == nil {
		return nil
	}
	return &d.Time
}

func toAPIDate(t *time.Time) *openapi_types.Date {
	if t == nil {
		return nil
	}
	return &openapi_types.Date{Time: *t}
}
