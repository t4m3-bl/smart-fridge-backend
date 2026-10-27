package middleware

import (
	"net/http"
	"regexp"

	"github.com/getkin/kin-openapi/openapi3"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
)

func JWT(signingKey string, spec *openapi3.T, specPath, basePath string) echo.MiddlewareFunc {
	public := buildPublicRoutes(spec, basePath)
	public[routeKey(http.MethodGet, specPath)] = struct{}{}

	return echojwt.WithConfig(echojwt.Config{
		SigningKey: []byte(signingKey),
		Skipper: func(c *echo.Context) bool {
			if c.RouteInfo().Method == echo.RouteNotFound {
				return true
			}
			_, ok := public[routeKey(c.Request().Method, c.Path())]
			return ok
		},
		SuccessHandler: putUserID,
		ErrorHandler: func(_ *echo.Context, err error) error {
			return echo.ErrUnauthorized.Wrap(err)
		},
	})
}

var pathParam = regexp.MustCompile(`\{([^}/]+)}`)

func buildPublicRoutes(spec *openapi3.T, basePath string) map[string]struct{} {
	public := make(map[string]struct{})

	for path, item := range spec.Paths.Map() {
		echoPath := basePath + pathParam.ReplaceAllString(path, ":$1")

		for method, op := range item.Operations() {
			sec := op.Security
			if sec == nil {
				sec = &spec.Security
			}
			if len(*sec) == 0 {
				public[routeKey(method, echoPath)] = struct{}{}
			}
		}
	}
	return public
}

func routeKey(method, path string) string {
	return method + " " + path
}
