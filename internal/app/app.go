package app

import (
	"context"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/repository/postgres"
	"smart-fridge-backend/internal/repository/postgres/sqlc"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/controllers"
	"smart-fridge-backend/internal/transport/http/cookie"
	"smart-fridge-backend/internal/usecase/auth/oauth"
	"smart-fridge-backend/internal/usecase/auth/tokens"
	"smart-fridge-backend/internal/usecase/service"
	"smart-fridge-backend/pkg/crypto/argon2id"

	httpserver "smart-fridge-backend/internal/transport/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

func New(cfg *config.Config) *fx.App {
	return fx.New(
		fx.Supply(cfg),
		fx.Supply(cfg.App.FrontendURL),
		fx.StopTimeout(cfg.App.ShutdownTimeout),

		infrastructure,
		usecases,
		transport,
	)
}

var infrastructure = fx.Module("infrastructure",
	fx.Provide(
		newPasswordHasher,
		fx.Annotate(newPool,
			fx.As(fx.Self()),
			fx.As(new(sqlc.DBTX)),
		),

		fx.Annotate(postgres.NewUserRepository,
			fx.As(new(service.UserRepository)),
			fx.As(new(service.UserFinder)),
		),
		fx.Annotate(postgres.NewSessionRepository,
			fx.As(new(service.SessionRepository)),
		),
	),
	fx.Invoke(postgres.RegisterLifecycle),
)

var usecases = fx.Module("usecases",
	fx.Provide(
		fx.Annotate(newAccessTokenManager, fx.As(new(service.AccessTokenGenerator))),
		fx.Annotate(newRefreshTokenManager, fx.As(new(service.RefreshTokenGenerator))),
		fx.Annotate(service.NewSessionService,
			fx.As(fx.Self()),
			fx.As(new(service.SessionCreator)),
		),
		fx.Annotate(newOAuthProviders, fx.As(new(service.OAuthProviders))),
		fx.Annotate(newOAuthTokensManager, fx.As(new(service.OauthTokensGenerator))),
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
		Memory:  cfg.Crypto.Argon2.MemoryKiB,
		Time:    cfg.Crypto.Argon2.Time,
		Threads: cfg.Crypto.Argon2.Threads,
		SaltLen: cfg.Crypto.Argon2.SaltLen,
		KeyLen:  cfg.Crypto.Argon2.KeyLen,
	})
}

func newPool(cfg *config.Config) (*pgxpool.Pool, error) {
	return postgres.NewPool(context.Background(), cfg.Database.URL)
}

func newAccessTokenManager(cfg *config.Config) (*tokens.JWTTokenManager, error) {
	return tokens.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.JWT.AccessTTL)
}

func newRefreshTokenManager(cfg *config.Config) (*tokens.RefreshTokenManager, error) {
	return tokens.NewRefreshTokenManager(cfg.Auth.JWT.RefreshTTL)
}

func newOAuthProviders(cfg *config.Config) *oauth.BaseProvider {
	return oauth.New(cfg.Auth.OAuth)
}

func newOAuthTokensManager(cfg *config.Config) (*tokens.OauthTokensManger, error) {
	return tokens.NewOauthTokensManger(cfg.Auth.OAuth.TokensTTL)
}

func newCookieFactory(cfg *config.Config) *cookie.Factory {
	return cookie.NewFactory(cfg.App.Env == config.PROD)
}
