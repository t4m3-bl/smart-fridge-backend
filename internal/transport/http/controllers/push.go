package controllers

import (
	"context"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/middleware"

	"github.com/google/uuid"
)

type PushController struct {
	repo           PushSubRepository
	vapidPublicKey string
}

type PushSubRepository interface {
	Upsert(ctx context.Context, userID domain.UserID, input dto.PushSubscriptionInput) (uuid.UUID, error)
	Revoke(ctx context.Context, userID domain.UserID, subscriptionID uuid.UUID) error
}

func NewPushController(repo PushSubRepository, key string) *PushController {
	return &PushController{
		repo:           repo,
		vapidPublicKey: key,
	}
}

func (c *PushController) GetVapidPublicKey(_ context.Context, _ api.GetVapidPublicKeyRequestObject) (api.GetVapidPublicKeyResponseObject, error) {
	return api.GetVapidPublicKey200JSONResponse{
		VapidPublicKeySuccessJSONResponse: api.VapidPublicKeySuccessJSONResponse{
			PublicKey: c.vapidPublicKey,
		},
	}, nil
}

func (c *PushController) UpsertPushSubscription(ctx context.Context, request api.UpsertPushSubscriptionRequestObject) (api.UpsertPushSubscriptionResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := c.repo.Upsert(ctx, domain.UserID(userID), dto.PushSubscriptionInput{
		Endpoint: request.Body.Endpoint,
		P256DH:   request.Body.Keys.P256dh,
		Auth:     request.Body.Keys.Auth,
	},
	)
	if err != nil {
		return nil, err
	}

	return api.UpsertPushSubscription200JSONResponse{
		PushSubscriptionSuccessJSONResponse: api.PushSubscriptionSuccessJSONResponse{
			Id: id,
		},
	}, nil
}

func (c *PushController) DeletePushSubscription(ctx context.Context, request api.DeletePushSubscriptionRequestObject) (api.DeletePushSubscriptionResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	if err := c.repo.Revoke(ctx, domain.UserID(userID), request.SubscriptionId); err != nil {
		return nil, err
	}

	return api.DeletePushSubscription204Response{}, nil
}
