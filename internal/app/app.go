package app

import (
	"context"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/repository/postgres"
	"smart-fridge-backend/internal/repository/postgres/sqlc"
	http2 "smart-fridge-backend/internal/transport/http"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/controllers"
	"smart-fridge-backend/internal/transport/http/cookie"
	"smart-fridge-backend/internal/usecase/auth/oauth"
	"smart-fridge-backend/internal/usecase/auth/tokens"
	"smart-fridge-backend/internal/usecase/service"
	"smart-fridge-backend/pkg/crypto/argon2id"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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
		newAuthController,
		fx.Annotate(http2.NewHandler, fx.As(new(api.StrictServerInterface))),
		http2.NewServer,
	),
	fx.Invoke(http2.RegisterLifecycle),
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

func newPool(cfg *config.Config) (*pgxpool.Pool, error) {
	return postgres.NewPool(context.Background(), cfg.Database.URL)
}

func newAccessTokenManager(cfg *config.Config) (*tokens.JWTTokenManager, error) {
	return tokens.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.AccessTTL)
}

func newRefreshTokenManager(cfg *config.Config) (*tokens.RefreshTokenManager, error) {
	return tokens.NewRefreshTokenManager(cfg.Auth.RefreshTTL)
}

func newOAuthProviders(cfg *config.Config) *oauth.BaseProvider {
	return oauth.New(cfg.OAuth)
}

func newOAuthTokensManager(cfg *config.Config) (*tokens.OauthTokensManger, error) {
	return tokens.NewOauthTokensManger(cfg.Auth.OAuthTTl)
}

func newAuthController(
	cfg *config.Config,
	cookies *cookie.Factory,
	auth *service.AuthService,
	session *service.SessionService,
) *controllers.AuthController {
	return controllers.NewAuthController(cookies, auth, session, cfg.OAuth.FrontendURL)
}

func newCookieFactory(cfg *config.Config) *cookie.Factory {
	return cookie.NewFactory(cfg.Mode == config.PROD)
}
