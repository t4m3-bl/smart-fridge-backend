package controllers

import (
	"context"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/middleware"
	"smart-fridge-backend/internal/usecase/service"
)

type ImageController struct {
	service *service.ProductImageService
}

func NewImageController(s *service.ProductImageService) *ImageController {
	return &ImageController{
		service: s,
	}
}

func (c *ImageController) PrepareProductImageUploads(
	ctx context.Context,
	request api.PrepareProductImageUploadsRequestObject,
) (api.PrepareProductImageUploadsResponseObject, error) {
	userID, err := middleware.GetUserID(ctx)
	if err != nil {
		return nil, err
	}

	if len(request.Body.Operations) == 0 {
		return nil, domain.NewValidationErr("operations", "at least one operation is required")
	}

	inputs := make([]dto.ImageUploadInput, 0, len(request.Body.Operations))
	for _, operation := range request.Body.Operations {
		inputs = append(inputs, dto.ImageUploadInput{
			OperationID: operation.OperationId,
			ContentType: string(operation.ContentType),
		})
	}

	uploads, err := c.service.PrepareUploads(ctx, domain.UserID(userID), inputs)
	if err != nil {
		return nil, err
	}

	response := make([]api.ProductImageUploadTarget, 0, len(uploads))
	for _, upload := range uploads {
		response = append(response, api.ProductImageUploadTarget{
			OperationId:     upload.OperationID,
			UploadUrl:       upload.UploadURL,
			ObjectId:        upload.ObjectID,
			ExpiresAt:       upload.ExpiresAt,
			RequiredHeaders: upload.RequiredHeaders,
		})
	}

	return api.PrepareProductImageUploads200JSONResponse{
		Uploads: response,
	}, nil
}
