package controllers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/middleware"
	"smart-fridge-backend/internal/transport/http/utils/header"

	"github.com/labstack/echo/v5"
)

func (c *ProductController) UploadProductImage(ctx context.Context, request api.UploadProductImageRequestObject) (api.UploadProductImageResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}
	version, err := header.ResolveVersion(request.Params.IfMatch, nil)
	if err != nil {
		return nil, err
	}
	data, contentType, err := readImage(request.Body, c.maxImageSize)
	if err != nil {
		return nil, err
	}
	version, err = c.service.UploadImage(ctx, domain.UserID(userID), domain.ProductID(request.Id), version, bytes.NewReader(data), contentType)
	if err != nil {
		return nil, err
	}
	return api.UploadProductImage200Response{
		Headers: api.UploadProductImage200ResponseHeaders{ETag: new(fmt.Sprintf(`"%d"`, version))},
	}, nil
}

func (c *ProductController) DeleteProductImage(ctx context.Context, request api.DeleteProductImageRequestObject) (api.DeleteProductImageResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}
	version, err := header.ResolveVersion(request.Params.IfMatch, nil)
	if err != nil {
		return nil, err
	}
	version, err = c.service.DeleteImage(ctx, domain.UserID(userID), domain.ProductID(request.Id), version)
	if err != nil {
		return nil, err
	}
	return api.DeleteProductImage200Response{
		Headers: api.DeleteProductImage200ResponseHeaders{ETag: new(fmt.Sprintf(`"%d"`, version))},
	}, nil
}

func readImage(body *multipart.Reader, limit int64) ([]byte, string, error) {
	if body == nil {
		return nil, "", domain.NewValidationErr("image", "file is required")
	}
	part, err := body.NextPart()
	if err != nil || part.FormName() != "image" || part.FileName() == "" {
		return nil, "", domain.NewValidationErr("image", "one image file is required")
	}
	data, err := io.ReadAll(io.LimitReader(part, limit+1))
	if errors.Is(err, echo.ErrStatusRequestEntityTooLarge) || int64(len(data)) > limit {
		return nil, "", echo.ErrStatusRequestEntityTooLarge
	}
	if err != nil {
		return nil, "", domain.NewValidationErr("image", "invalid multipart body")
	}
	if _, err := body.NextPart(); !errors.Is(err, io.EOF) {
		return nil, "", domain.NewValidationErr("image", "only one file is allowed")
	}
	info, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") || info.Width <= 0 || info.Height <= 0 {
		return nil, "", domain.NewValidationErr("image", "valid JPEG or PNG file required")
	}
	return data, "image/" + format, nil
}
