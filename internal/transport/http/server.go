package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/middleware"
	"smart-fridge-backend/internal/transport/http/swagger"
	"time"

	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
	"go.uber.org/fx"
)

type Server struct {
	echo *echo.Echo

	address         string
	gracefulTimeout time.Duration

	cancel context.CancelFunc
	done   chan struct{}
}

func NewServer(cfg *config.Config, h api.StrictServerInterface) (*Server, error) {
	// get open-api spec
	spec, err := api.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("get spec: %w", err)
	}

	e := echo.New()
	e.HTTPErrorHandler = middleware.HTTPErrorHandler

	e.Use(echoMiddleware.RequestID())
	e.Use(echoMiddleware.RequestLogger())
	e.Use(echoMiddleware.Recover())

	if len(cfg.HTTP.CORSOrigins) > 0 {
		e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
			AllowOrigins:     cfg.HTTP.CORSOrigins,
			AllowCredentials: true,
		}))
	}
	e.Use(echoMiddleware.Gzip())

	// prod mode addons
	if cfg.App.Env == config.PROD {
		e.Use(echoMiddleware.Secure())
	}

	// routes and controllers registration
	apiGroup := e.Group("/api/v1", middleware.JWT(cfg.Auth.JWTSecret, spec, swagger.SpecPath, "/api/v1"))
	api.RegisterHandlers(apiGroup, api.NewStrictHandler(h, nil))
	e.GET("/favicon.ico", noContentHandler)

	if cfg.HTTP.SwaggerEnabled {
		swagger.UseSwagger(e)
	}

	return &Server{
		echo:            e,
		address:         cfg.HTTP.Addr,
		gracefulTimeout: cfg.HTTP.ShutdownTimeout,
	}, nil
}

//nolint:contextcheck
func (s *Server) Start(ctx context.Context, onFatal func(error)) error {
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", s.address)
	if err != nil {
		return fmt.Errorf("http listen %s: %w", s.address, err)
	}

	serveCtx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})

	sc := echo.StartConfig{
		Listener:        ln,
		HideBanner:      true,
		GracefulTimeout: s.gracefulTimeout,
		OnShutdownError: func(err error) {
			s.echo.Logger.Error("http graceful shutdown failed", slog.Any("error", err))
		},
	}

	go func() {
		defer close(s.done)
		if err := sc.Start(serveCtx, s.echo); err != nil {
			onFatal(fmt.Errorf("http server: %w", err))
		}
	}()

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.cancel == nil {
		return nil
	}
	s.cancel()

	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return errors.New("http server: shutdown did not finish in time")
	}
}

func RegisterLifecycle(lc fx.Lifecycle, s *Server, shutdowner fx.Shutdowner) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return s.Start(ctx, func(err error) {
				s.echo.Logger.Error("http server stopped unexpectedly", slog.Any("error", err))
				_ = shutdowner.Shutdown(fx.ExitCode(1))
			})
		},
		OnStop: s.Stop,
	})
}
