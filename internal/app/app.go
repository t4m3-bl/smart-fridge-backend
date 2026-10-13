package app

import (
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/repository/postgres"
	"smart-fridge-backend/internal/repository/postgres/sqlc"
	httpserver "smart-fridge-backend/internal/transport/http"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/controllers"
	"smart-fridge-backend/internal/transport/http/cookie"
	"smart-fridge-backend/internal/usecase/auth/tokens"
	"smart-fridge-backend/internal/usecase/service"
	"smart-fridge-backend/pkg/crypto/argon2id"
	"time"

	"go.uber.org/fx"
)

func New(cfg *config.Config) *fx.App {
	return fx.New(
		fx.Supply(cfg),
		fx.StopTimeout(cfg.HTTP.ShutdownTimeout+5*time.Second),

		infrastructure,
		usecases,
		transport,
	)
}

var infrastructure = fx.Module("infrastructure",
	fx.Provide(
		newPasswordHasher,
		fx.Annotate(postgres.NewPool,
			fx.As(fx.Self()),
			fx.As(new(sqlc.DBTX)),
			fx.As(new(postgres.TxDB)),
		),

		fx.Annotate(postgres.NewUserRepository,
			fx.As(new(service.UserRepository)),
			fx.As(new(service.UserRepository1)),
		),
		fx.Annotate(postgres.NewSessionRepository,
			fx.As(new(service.SessionRepository)),
		),
	),
	fx.Invoke(postgres.RegisterLifecycle),
)

var usecases = fx.Module("usecases",
	fx.Provide(
		newAccessTokenManager,
		newRefreshTokenManager,
		service.NewSessionService,
		service.NewAuthService,
	),
)

var transport = fx.Module("transport",
	fx.Provide(
		newCookieFactory,
		controllers.NewAuthController,
		fx.Annotate(httpserver.NewHandler, fx.As(new(api.StrictServerInterface))),
		httpserver.NewServer,
	),
	fx.Invoke(httpserver.RegisterLifecycle),
)

func newPasswordHasher(cfg *config.Config) service.PasswordHasher {
	return argon2id.NewArgon2idHasher(argon2id.Params{
		Memory:  cfg.Argon2.MemoryKiB,
		Time:    cfg.Argon2.Time,
		Threads: cfg.Argon2.Threads,
		SaltLen: cfg.Argon2.SaltLen,
		KeyLen:  cfg.Argon2.KeyLen,
	})
}

func newAccessTokenManager(cfg *config.Config) *tokens.JWTTokenManager {
	return tokens.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.AccessTTL)
}

func newRefreshTokenManager(cfg *config.Config) *tokens.RefreshTokenManager {
	return tokens.NewRefreshTokenManager(cfg.Auth.RefreshTTL)
}

func newCookieFactory(cfg *config.Config) *cookie.Factory {
	return cookie.NewFactory(cfg.Mode == config.PROD)
}
