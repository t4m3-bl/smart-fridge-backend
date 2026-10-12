package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/migrations"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.uber.org/fx"
)

func NewPool(lc fx.Lifecycle, cfg *config.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := pool.Ping(ctx); err != nil {
				return fmt.Errorf("ping postgres: %w", err)
			}
			return migrate(ctx, pool)
		},
		OnStop: func(context.Context) error {
			pool.Close()
			return nil
		},
	})

	return pool, nil
}

func migrate(ctx context.Context, pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	for _, r := range results {
		slog.Info("migration applied", "version", r.Source.Version, "file", r.Source.Path, "duration", r.Duration)
	}

	return nil
}
