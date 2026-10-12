package middleware

import (
	"strings"

	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
)

var publicPrefixes = []string{"/api/v1/auth/", "/swagger/ui", "/api/v1/openapi.json"}

func JWT(signingKey string) echo.MiddlewareFunc {
	return echojwt.WithConfig(echojwt.Config{
		SigningKey: []byte(signingKey),
		Skipper:    isPublic,
		ErrorHandler: func(_ *echo.Context, err error) error {
			return echo.ErrUnauthorized.Wrap(err)
		},
	})
}

func isPublic(c *echo.Context) bool {
	path := c.Request().URL.Path
	for _, p := range publicPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}

	return false
}
