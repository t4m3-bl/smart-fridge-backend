//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxtest"

	db "smart-fridge-backend/internal/repository/postgres"
)

func TestNewPool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		dsn             string
		wantErrContains string
	}{
		{
			name: "valid dsn",
			dsn:  testPool.Config().ConnString(),
		},
		{
			name:            "malformed dsn",
			dsn:             "postgres://user:pass@localhost:not-a-port/db",
			wantErrContains: "parse postgres dsn",
		},
		{
			name:            "invalid pool option",
			dsn:             testPool.Config().ConnString() + "&pool_max_conns=0",
			wantErrContains: "parse postgres dsn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			pool, err := db.NewPool(ctx, tt.dsn)

			if tt.wantErrContains != "" {
				require.ErrorContains(t, err, tt.wantErrContains)
				assert.Nil(t, pool)
				return
			}

			require.NoError(t, err)
			t.Cleanup(pool.Close)
			assert.NoError(t, pool.Ping(ctx))
		})
	}
}

func TestMigrate(t *testing.T) {
	t.Parallel()

	appliedTables := []string{"users", "identities", "sessions"}

	tests := []struct {
		name        string
		pool        func(t *testing.T) *pgxpool.Pool
		canceledCtx bool
		wantErr     require.ErrorAssertionFunc
		wantTables  []string
	}{
		{
			name:       "applies all migrations to an empty database",
			pool:       newEmptyDatabasePool,
			wantErr:    require.NoError,
			wantTables: appliedTables,
		},
		{
			name:       "already migrated database is a no-op",
			pool:       newPool,
			wantErr:    require.NoError,
			wantTables: appliedTables,
		},
		{
			name: "repeated run is idempotent",
			pool: func(t *testing.T) *pgxpool.Pool {
				t.Helper()

				pool := newEmptyDatabasePool(t)
				require.NoError(t, db.Migrate(context.Background(), pool))
				return pool
			},
			wantErr:    require.NoError,
			wantTables: appliedTables,
		},
		{
			name:    "closed pool",
			pool:    closedPool,
			wantErr: require.Error,
		},
		{
			name:        "canceled context",
			pool:        newPool,
			canceledCtx: true,
			wantErr:     require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pool := tt.pool(t)

			ctx := context.Background()
			if tt.canceledCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			err := db.Migrate(ctx, pool)

			tt.wantErr(t, err)

			for _, table := range tt.wantTables {
				var exists bool
				require.NoError(t, pool.QueryRow(context.Background(),
					"SELECT to_regclass($1) IS NOT NULL", "public."+table).Scan(&exists))
				assert.True(t, exists, "table %q must exist", table)
			}
		})
	}
}

func TestRegisterLifecycle(t *testing.T) {
	t.Parallel()

	unreachablePool := func(t *testing.T) *pgxpool.Pool {
		t.Helper()

		pool, err := db.NewPool(context.Background(),
			"postgres://user:pass@127.0.0.1:1/db?sslmode=disable&connect_timeout=1")
		require.NoError(t, err)
		t.Cleanup(pool.Close)

		return pool
	}

	tests := []struct {
		name         string
		pool         func(t *testing.T) *pgxpool.Pool
		wantStartErr string
	}{
		{
			name: "healthy database",
			pool: newPool,
		},
		{
			name:         "unreachable database",
			pool:         unreachablePool,
			wantStartErr: "ping postgres",
		},
		{
			name:         "closed pool",
			pool:         closedPool,
			wantStartErr: "ping postgres",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pool := tt.pool(t)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			lc := fxtest.NewLifecycle(t)
			db.RegisterLifecycle(lc, pool)

			err := lc.Start(ctx)

			if tt.wantStartErr != "" {
				require.ErrorContains(t, err, tt.wantStartErr)
				return
			}

			require.NoError(t, err)
			require.NoError(t, lc.Stop(ctx))
			assert.Error(t, pool.Ping(ctx), "pool must be closed after stop")
		})
	}
}
