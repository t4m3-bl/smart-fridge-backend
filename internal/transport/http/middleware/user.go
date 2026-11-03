package middleware

import (
	"context"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type userIDKey struct{}

func UserID(ctx context.Context) (uuid.UUID, error) {
	id, ok := ctx.Value(userIDKey{}).(uuid.UUID)
	if !ok || id == uuid.Nil {
		return uuid.Nil, echo.ErrUnauthorized
	}
	return id, nil
}
