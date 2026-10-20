package controllers

import (
	"context"
	"net/http"
	"smart-fridge-backend/internal/transport/http/api"

	"github.com/labstack/echo/v5"
)

var errNotImplemented = echo.NewHTTPError(http.StatusNotImplemented, "not implemented")

type Unimplemented struct{}

func (Unimplemented) SyncPush(context.Context, api.SyncPushRequestObject) (api.SyncPushResponseObject, error) {
	return nil, errNotImplemented
}

func (Unimplemented) SyncPull(context.Context, api.SyncPullRequestObject) (api.SyncPullResponseObject, error) {
	return nil, errNotImplemented
}

func (Unimplemented) SyncSnapshot(context.Context, api.SyncSnapshotRequestObject) (api.SyncSnapshotResponseObject, error) {
	return nil, errNotImplemented
}

func (Unimplemented) ScanCode(context.Context, api.ScanCodeRequestObject) (api.ScanCodeResponseObject, error) {
	return nil, errNotImplemented
}

func (Unimplemented) ScanPhoto(context.Context, api.ScanPhotoRequestObject) (api.ScanPhotoResponseObject, error) {
	return nil, errNotImplemented
}

func (Unimplemented) CreatePhotoUploadUrl(context.Context, api.CreatePhotoUploadUrlRequestObject) (api.CreatePhotoUploadUrlResponseObject, error) {
	return nil, errNotImplemented
}

func (Unimplemented) GetPhotoUrl(context.Context, api.GetPhotoUrlRequestObject) (api.GetPhotoUrlResponseObject, error) {
	return nil, errNotImplemented
}
