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
	if resp, uErr := echo.UnwrapResponse(c.Response()); uErr == nil {
		if resp.Committed {
			return
		}
	}

	problem := api.Problem{
		RequestId: new(c.Response().Header().Get(echo.HeaderXRequestID)),
		Status:    http.StatusInternalServerError,
		Type:      "about:blank",
	}

	// determine the status code for the error
	if _, ok := errors.AsType[domain.ValidationErr](err); ok {
		problem.Status = http.StatusBadRequest
	} else {
		switch {
		case errors.Is(err, domain.ErrAlreadyExists):
			problem.Status = http.StatusConflict

		case errors.Is(err, domain.ErrInvalidCredentials),
			errors.Is(err, domain.ErrInvalidSession):
			problem.Status = http.StatusUnauthorized

		case errors.Is(err, domain.ErrNotFound),
			errors.Is(err, domain.ErrUnknownProvider):
			problem.Status = http.StatusNotFound

		case errors.Is(err, domain.ErrProviderUnavailable):
			problem.Status = http.StatusBadGateway

		case echo.StatusCode(err) != 0:
			problem.Status = echo.StatusCode(err)
		}
	}

	problem.Title = http.StatusText(problem.Status)
	if problem.Status < http.StatusInternalServerError {
		problem.Detail = new(err.Error())
	}

	c.Response().Header().Set(echo.HeaderContentType, MIMEApplicationProblemJSON)
	if errJSON := c.JSON(problem.Status, problem); errJSON != nil {
		c.Logger().Error(errJSON.Error())
	}
}
