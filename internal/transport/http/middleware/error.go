package middleware

import (
	"errors"
	"net/http"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/transport/http/api"

	"github.com/labstack/echo/v5"
)

const MIMEApplicationProblemJSON = "application/problem+json"

func HTTPErrorHandler(c *echo.Context, err error) {
	if resp, uErr := echo.UnwrapResponse(c.Response()); uErr == nil && resp.Committed {
		return
	}

	status := http.StatusInternalServerError
	problem := api.Error{
		Code:    "internal_error",
		Message: "internal server error",
	}

	if vErr, ok := errors.AsType[domain.ValidationError](err); ok {
		status = http.StatusBadRequest
		problem.Code = "validation_error"
		problem.Message = vErr.Error()
	} else {
		switch {
		case errors.Is(err, domain.ErrAlreadyExists):
			status = http.StatusConflict
			problem.Code = "already_exists"
			problem.Message = err.Error()

		case errors.Is(err, domain.ErrInvalidCredentials):
			status = http.StatusUnauthorized
			problem.Code = "invalid_credentials"
			problem.Message = err.Error()

		case errors.Is(err, domain.ErrInvalidSession):
			status = http.StatusUnauthorized
			problem.Code = "invalid_session"
			problem.Message = err.Error()

		case errors.Is(err, domain.ErrNotFound):
			status = http.StatusNotFound
			problem.Code = "not_found"
			problem.Message = err.Error()

		case errors.Is(err, domain.ErrUnknownProvider):
			status = http.StatusNotFound
			problem.Code = "unknown_provider"
			problem.Message = err.Error()

		case errors.Is(err, domain.ErrUnprocessableEntity):
			status = http.StatusUnprocessableEntity
			problem.Code = "unprocessable_entity"
			problem.Message = err.Error()

		case errors.Is(err, domain.ErrProviderUnavailable):
			status = http.StatusBadGateway
			problem.Code = "provider_unavailable"
			problem.Message = err.Error()

		default:
			if s := echo.StatusCode(err); s != 0 {
				status = s
				problem.Code = http.StatusText(s)
				problem.Message = err.Error()
			}
		}
	}

	c.Response().Header().Set(echo.HeaderContentType, MIMEApplicationProblemJSON)
	if jsonErr := c.JSON(status, problem); jsonErr != nil {
		c.Logger().Error(jsonErr.Error())
	}
}
