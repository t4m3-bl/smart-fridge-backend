package middleware

import (
	"context"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type userIDKey struct{}

func SetUserID(c *echo.Context, id uuid.UUID) {
	req := c.Request()
	ctx := context.WithValue(req.Context(), userIDKey{}, id)
	c.SetRequest(req.WithContext(ctx))
}

func GetUserID(ctx context.Context) (uuid.UUID, error) {
	id, ok := ctx.Value(userIDKey{}).(uuid.UUID)
	if !ok || id == uuid.Nil {
		return uuid.Nil, echo.ErrUnauthorized
	}
	return id, nil
}
