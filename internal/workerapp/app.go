package workerapp

import (
	"context"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/repository/postgres"
	"smart-fridge-backend/internal/transport/httpclient/webpush"
	"smart-fridge-backend/internal/usecase/service"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

func New(cfg *config.WorkerConfig) *fx.App { return fx.New(options(cfg)) }

func options(cfg *config.WorkerConfig) fx.Option {
	return fx.Options(
		fx.Supply(cfg), fx.Supply(cfg.Push), fx.StopTimeout(cfg.App.ShutdownTimeout),
		fx.Provide(newPool,
			fx.Annotate(postgres.NewNotificationRepository, fx.As(new(service.NotificationRepository))),
			fx.Annotate(webpush.New, fx.As(new(service.PushSender))),
			service.NewNotificationWorker,
		),
		fx.Invoke(postgres.RegisterLifecycle),
		fx.Invoke(service.RegisterNotificationLifecycle),
	)
}

func newPool(cfg *config.WorkerConfig) (*pgxpool.Pool, error) {
	return postgres.NewPool(context.Background(), cfg.Database.URL)
}
