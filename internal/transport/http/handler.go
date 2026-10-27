package http

import (
	"net/http"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/controllers"

	"github.com/labstack/echo/v5"
)

var _ api.StrictServerInterface = (*Handler)(nil)

type Handler struct {
	*controllers.AuthController
	*controllers.SyncController
}

func NewHandler(auth *controllers.AuthController, sync *controllers.SyncController) *Handler {
	return &Handler{
		AuthController: auth,
		SyncController: sync,
	}
}

func noContentHandler(c *echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}
