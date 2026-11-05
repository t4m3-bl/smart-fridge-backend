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
	*controllers.SettingController
	*controllers.ProductController
	*controllers.CategoryController
}

func NewHandler(
	auth *controllers.AuthController,
	setting *controllers.SettingController,
	product *controllers.ProductController,
	category *controllers.CategoryController,
) *Handler {
	return &Handler{
		AuthController:     auth,
		SettingController:  setting,
		ProductController:  product,
		CategoryController: category,
	}
}

func noContentHandler(c *echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}
