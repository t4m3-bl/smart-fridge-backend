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

	if _, ok := errors.AsType[domain.ValidationErr](err); ok {
		problem.Status = http.StatusBadRequest
		problem.Detail = new(err.Error())
		problem.Type = "https://smart-fridge.local"
	} else if errors.Is(err, domain.ErrAlreadyExists) {
		problem.Status = http.StatusConflict
	} else if errors.Is(err, domain.ErrInvalidCredentials) || errors.Is(err, domain.ErrInvalidSession) {
		problem.Status = http.StatusUnauthorized
	} else if errors.Is(err, domain.ErrNotFound) {
		problem.Status = http.StatusNotFound
	} else if code := echo.StatusCode(err); code != 0 {
		problem.Status = code
	}

	problem.Title = http.StatusText(problem.Status)

	c.Response().Header().Set(echo.HeaderContentType, MIMEApplicationProblemJSON)
	if errJSON := c.JSON(problem.Status, problem); errJSON != nil {
		c.Logger().Error(errJSON.Error())
	}
}
