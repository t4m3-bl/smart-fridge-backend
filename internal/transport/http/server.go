package http

import (
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/middleware"
	"smart-fridge-backend/internal/transport/http/swagger"
	"time"

	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
)

type Params struct {
	mode config.Mode

	address         string
	gracefulTimeout time.Duration

	CORSOrigins []string
	signingKey  string
}

type Server struct {
	echo *echo.Echo

	address         string
	gracefulTimeout time.Duration
}

func NewServer(p Params, h api.StrictServerInterface) (*Server, error) {
	e := echo.New()
	e.HTTPErrorHandler = middleware.HTTPErrorHandler

	e.Use(echoMiddleware.RequestID())
	e.Use(echoMiddleware.RequestLogger())
	e.Use(echoMiddleware.Recover())

	e.Use(echoMiddleware.CORS(p.CORSOrigins...))
	e.Use(echoMiddleware.Gzip())

	e.Use(middleware.JWT(p.signingKey))

	api.RegisterHandlersWithBaseURL(e, api.NewStrictHandler(h, nil), "/api/v1")

	if p.mode == config.DEV {
		swagger.UseSwagger(e)
	}

	if p.mode == config.PROD {
		e.Use(echoMiddleware.Secure())
	}

	return &Server{
		echo:            e,
		address:         p.address,
		gracefulTimeout: p.gracefulTimeout,
	}, nil
}
