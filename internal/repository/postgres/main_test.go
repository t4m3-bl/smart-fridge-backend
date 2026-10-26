//go:build integration

package postgres_test

import (
	"context"
	"log"
	"os"
	db "smart-fridge-backend/internal/repository/postgres"

	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("test-user"),
		postgres.WithPassword("test-password"),
		postgres.WithSQLDriver("pgx"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Printf("failed to start postgres container: %v", err)
		return 1
	}

	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %v", err)
		}
	}()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Printf("failed to get connection string: %v", err)
		return 1
	}

	testPool, err = pgxpool.New(ctx, connStr)
	if err != nil {
		log.Printf("failed to create pool: %v", err)
		return 1
	}
	defer testPool.Close()

	if err := db.Migrate(ctx, testPool); err != nil {
		log.Printf("failed to apply migrations: %v", err)
		return 1
	}

	return m.Run()
}
