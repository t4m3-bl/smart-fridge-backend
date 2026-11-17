package middleware

import (
	"errors"
	"net/http"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/transport/http/api"

	"github.com/labstack/echo/v5"
)

const MIMEApplicationProblemJSON = "application/problem+json"

type errMapping struct {
	target  error
	status  int
	code    string
	message string
}

//nolint:gochecknoglobals
var domainErrMappings = []errMapping{
	{
		domain.ErrPreconditionRequired,
		http.StatusPreconditionRequired,
		"precondition_required",
		"a version precondition is required",
	},
	{
		domain.ErrImageStorageUnavailable,
		http.StatusServiceUnavailable,
		"image_storage_unavailable",
		"image storage is not configured",
	},
	{
		domain.ErrImageStorageFailed,
		http.StatusBadGateway,
		"image_storage_failed",
		"image storage request failed",
	},
	{
		domain.ErrVersionConflict,
		http.StatusPreconditionFailed,
		"version_conflict",
		"",
	},
	{
		domain.ErrAlreadyExists,
		http.StatusConflict,
		"already_exists",
		"",
	},
	{
		domain.ErrInvalidCredentials,
		http.StatusUnauthorized,
		"invalid_credentials",
		"",
	},
	{
		domain.ErrInvalidSession,
		http.StatusUnauthorized,
		"invalid_session",
		"",
	},
	{
		domain.ErrNotFound,
		http.StatusNotFound,
		"not_found",
		"",
	},
	{
		domain.ErrUnknownProvider,
		http.StatusNotFound,
		"unknown_provider",
		"",
	},
	{
		domain.ErrUnprocessableEntity,
		http.StatusUnprocessableEntity,
		"unprocessable_entity",
		"",
	},
	{
		domain.ErrProviderUnavailable,
		http.StatusBadGateway,
		"provider_unavailable",
		"",
	},
}

func HTTPErrorHandler(c *echo.Context, err error) {
	if resp, uErr := echo.UnwrapResponse(c.Response()); uErr == nil && resp.Committed {
		return
	}

	status, problem := mapError(err)

	c.Response().Header().Set(echo.HeaderContentType, MIMEApplicationProblemJSON)
	if jsonErr := c.JSON(status, problem); jsonErr != nil {
		c.Logger().Error(jsonErr.Error())
	}
}

func mapError(err error) (int, api.Error) {
	if vErr, ok := errors.AsType[domain.ValidationError](err); ok {
		return http.StatusBadRequest, api.Error{
			Code:    "validation_error",
			Message: vErr.Error(),
		}
	}

	for _, m := range domainErrMappings {
		if errors.Is(err, m.target) {
			msg := m.message
			if msg == "" {
				msg = err.Error()
			}
			return m.status, api.Error{Code: m.code, Message: msg}
		}
	}

	if s := echo.StatusCode(err); s != 0 {
		return s, api.Error{
			Code:    http.StatusText(s),
			Message: err.Error(),
		}
	}

	return http.StatusInternalServerError, api.Error{
		Code:    "internal_error",
		Message: "internal server error",
	}
}
