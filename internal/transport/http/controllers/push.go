package controllers

import (
	"context"
	"crypto/rand"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/middleware"
	"smart-fridge-backend/internal/transport/http/utils/cookie"
	"smart-fridge-backend/internal/usecase/service"
	"smart-fridge-backend/pkg/crypto/sha256"
	"time"
)

type PushController struct {
	push    *service.PushService
	cookies *cookie.Factory
}

func NewPushController(push *service.PushService, cookies *cookie.Factory) *PushController {
	return &PushController{push: push, cookies: cookies}
}

func (c *PushController) GetVapidPublicKey(_ context.Context, _ api.GetVapidPublicKeyRequestObject) (api.GetVapidPublicKeyResponseObject, error) {
	key, err := c.push.PublicKey()
	if err != nil {
		return nil, err
	}
	return api.GetVapidPublicKey200JSONResponse{VapidPublicKeySuccessJSONResponse: api.VapidPublicKeySuccessJSONResponse{PublicKey: key}}, nil
}

func (c *PushController) UpsertPushSubscription(ctx context.Context, request api.UpsertPushSubscriptionRequestObject) (api.UpsertPushSubscriptionResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, domain.NewValidationErr("body", "required")
	}
	token := rand.Text()
	hash := sha256.Hash(token)
	id, err := c.push.Upsert(ctx, domain.UserID(userID), dto.PushSubscriptionInput{Endpoint: request.Body.Endpoint, P256DH: request.Body.Keys.P256dh, Auth: request.Body.Keys.Auth, LogoutTokenHash: &hash})
	if err != nil {
		return nil, err
	}
	return api.UpsertPushSubscription200JSONResponse{
		Body:    api.PushSubscriptionRegistered{Id: id},
		Headers: api.UpsertPushSubscription200ResponseHeaders{SetCookie: []string{c.cookies.CreatePushBindingCookie(token, 365*24*time.Hour).String()}},
	}, nil
}

func (c *PushController) DeletePushSubscription(ctx context.Context, request api.DeletePushSubscriptionRequestObject) (api.DeletePushSubscriptionResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err = c.push.Revoke(ctx, domain.UserID(userID), request.SubscriptionId); err != nil {
		return nil, err
	}
	return api.DeletePushSubscription204Response{}, nil
}
