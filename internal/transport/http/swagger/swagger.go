package swagger

import (
	_ "embed"
	"net/http"
	"smart-fridge-backend/internal/transport/http/api"

	"github.com/labstack/echo/v5"
)

//go:embed index.html
var swaggerUI string

const (
	SpecPath = "/api/v1/openapi.json"
	DocsPath = "/swagger/ui"
)

func UseSwagger(e *echo.Echo) {
	spec, err := api.GetSpecJSON()
	if err != nil {
		e.Logger.Error("get open-api spec", "error", err)
		return
	}

	e.GET(SpecPath, func(c *echo.Context) error {
		return c.Blob(http.StatusOK, echo.MIMEApplicationJSON, spec)
	})

	e.GET(DocsPath, func(c *echo.Context) error {
		return c.HTML(http.StatusOK, swaggerUI)
	})
}
