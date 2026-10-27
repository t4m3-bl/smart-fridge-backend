package middleware

import (
	"context"
	"errors"
	"smart-fridge-backend/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type userIDKey struct{}

var errNoUserID = errors.New("user id is missing in request context")

func UserID(ctx context.Context) (domain.UserID, error) {
	id, ok := ctx.Value(userIDKey{}).(domain.UserID)
	if !ok {
		return domain.UserID{}, echo.ErrUnauthorized.Wrap(errNoUserID)
	}
	return id, nil
}

func putUserID(c *echo.Context) error {
	token, ok := c.Get("user").(*jwt.Token)
	if !ok {
		return echo.ErrUnauthorized
	}

	sub, err := token.Claims.GetSubject()
	if err != nil {
		return echo.ErrUnauthorized.Wrap(err)
	}

	id, err := uuid.Parse(sub)
	if err != nil {
		return echo.ErrUnauthorized.Wrap(err)
	}

	req := c.Request()
	c.SetRequest(req.WithContext(context.WithValue(req.Context(), userIDKey{}, domain.UserID(id))))
	return nil
}
