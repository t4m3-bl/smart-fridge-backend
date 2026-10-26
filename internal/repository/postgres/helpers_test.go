//go:build integration

package postgres_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"smart-fridge-backend/internal/domain"
	db "smart-fridge-backend/internal/repository/postgres"
)

// errorIs returns an ErrorAssertionFunc, allowing both success and error cases
// to be described in a consistent manner within table-driven tests.
func errorIs(target error) require.ErrorAssertionFunc {
	return func(t require.TestingT, err error, _ ...any) {
		if h, ok := t.(interface{ Helper() }); ok {
			h.Helper()
		}
		require.ErrorIs(t, err, target)
	}
}

// newPool creates a separate pool for the test database, ensuring that tests
// which close the pool or hold onto connections do not interfere with the shared testPool.
func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	pool, err := db.NewPool(context.Background(), testPool.Config().ConnString())
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool
}

func closedPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	pool := newPool(t)
	pool.Close()

	return pool
}

// newEmptyDatabasePool creates a new, empty database (without migrations) in the same container
// and returns a pool connected to it. The database is deleted once the test concludes.
func newEmptyDatabasePool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()
	name := "t_" + strings.ReplaceAll(uuid.NewString(), "-", "")

	_, err := testPool.Exec(ctx, "CREATE DATABASE "+name)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
	})

	cfg := testPool.Config()
	cfg.ConnConfig.Database = name

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool
}

func newUUID() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}

// newUser assembles a user with a unique ID and email, and associates identities with them.
func newUser(identities ...domain.Identity) *domain.User {
	id := domain.UserID(newUUID())

	for i := range identities {
		identities[i].IdentityID = domain.IdentityID(newUUID())
		identities[i].UserID = id
	}

	return &domain.User{
		ID:         id,
		Email:      domain.Email(fmt.Sprintf("%s@example.com", uuid.NewString())),
		Identities: identities,
	}
}

func nativeIdentity() domain.Identity {
	return domain.Identity{
		Provider:     domain.ProviderNative,
		PasswordHash: new("password-hash-" + uuid.NewString()),
	}
}

func yandexIdentity(providerUserID string) domain.Identity {
	return domain.Identity{
		Provider:       domain.ProviderYandex,
		ProviderUserID: &providerUserID,
	}
}

// seedUser saves a new user via the repository and returns it.
func seedUser(t *testing.T, identities ...domain.Identity) *domain.User {
	t.Helper()

	user := newUser(identities...)
	require.NoError(t, db.NewUserRepository(testPool).Create(context.Background(), user))

	return user
}

// requireUserEqual compares users without regard for the order of identities:
// the repository does not guarantee the order in which they are returned.
func requireUserEqual(t *testing.T, want, got *domain.User) {
	t.Helper()

	require.NotNil(t, got)
	assert.Equal(t, want.ID, got.ID)
	assert.Equal(t, want.Email, got.Email)
	assert.ElementsMatch(t, want.Identities, got.Identities)
}
