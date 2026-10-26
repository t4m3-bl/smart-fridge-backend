//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"smart-fridge-backend/internal/domain"
	db "smart-fridge-backend/internal/repository/postgres"
)

func TestUserRepository_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		prepare func(t *testing.T) *domain.User
		wantErr error
	}{
		{
			name:    "user without identities",
			prepare: func(*testing.T) *domain.User { return newUser() },
		},
		{
			name:    "user with native identity",
			prepare: func(*testing.T) *domain.User { return newUser(nativeIdentity()) },
		},
		{
			name: "user with yandex identity",
			prepare: func(*testing.T) *domain.User {
				return newUser(yandexIdentity(uuid.NewString()))
			},
		},
		{
			name: "user with several identities",
			prepare: func(*testing.T) *domain.User {
				return newUser(nativeIdentity(), yandexIdentity(uuid.NewString()))
			},
		},
		{
			name: "duplicate email",
			prepare: func(t *testing.T) *domain.User {
				existing := seedUser(t)

				user := newUser()
				user.Email = existing.Email

				return user
			},
			wantErr: domain.ErrAlreadyExists,
		},
		{
			name: "duplicate user id",
			prepare: func(t *testing.T) *domain.User {
				existing := seedUser(t)

				user := newUser()
				user.ID = existing.ID

				return user
			},
			wantErr: domain.ErrAlreadyExists,
		},
		{
			name: "duplicate external identity",
			prepare: func(t *testing.T) *domain.User {
				providerUserID := uuid.NewString()
				seedUser(t, yandexIdentity(providerUserID))

				return newUser(yandexIdentity(providerUserID))
			},
			wantErr: domain.ErrAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			repo := db.NewUserRepository(testPool)
			user := tt.prepare(t)

			err := repo.Create(ctx, user)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)

			got, err := repo.FindByID(ctx, user.ID)
			require.NoError(t, err)
			requireUserEqual(t, user, got)
		})
	}
}

func TestUserRepository_Create_RollsBackOnIdentityConflict(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := db.NewUserRepository(testPool)

	providerUserID := uuid.NewString()
	seedUser(t, yandexIdentity(providerUserID))

	user := newUser(yandexIdentity(providerUserID))

	require.ErrorIs(t, repo.Create(ctx, user), domain.ErrAlreadyExists)

	_, err := repo.FindByID(ctx, user.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)

	_, err = repo.FindByEmail(ctx, user.Email)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserRepository_FindByEmail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := db.NewUserRepository(testPool)

	existing := seedUser(t, nativeIdentity(), yandexIdentity(uuid.NewString()))
	withoutIdentities := seedUser(t)

	tests := []struct {
		name    string
		email   domain.Email
		want    *domain.User
		wantErr error
	}{
		{
			name:  "user with identities",
			email: existing.Email,
			want:  existing,
		},
		{
			name:  "user without identities",
			email: withoutIdentities.Email,
			want:  withoutIdentities,
		},
		{
			name:    "unknown email",
			email:   domain.Email(uuid.NewString() + "@example.com"),
			wantErr: domain.ErrNotFound,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: domain.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := repo.FindByEmail(ctx, tt.email)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			requireUserEqual(t, tt.want, got)
		})
	}
}

func TestUserRepository_FindByID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := db.NewUserRepository(testPool)

	existing := seedUser(t, nativeIdentity(), yandexIdentity(uuid.NewString()))
	withoutIdentities := seedUser(t)

	tests := []struct {
		name    string
		id      domain.UserID
		want    *domain.User
		wantErr error
	}{
		{
			name: "user with identities",
			id:   existing.ID,
			want: existing,
		},
		{
			name: "user without identities",
			id:   withoutIdentities.ID,
			want: withoutIdentities,
		},
		{
			name:    "unknown id",
			id:      domain.UserID(newUUID()),
			wantErr: domain.ErrNotFound,
		},
		{
			name:    "nil uuid",
			id:      domain.UserID(uuid.Nil),
			wantErr: domain.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := repo.FindByID(ctx, tt.id)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			requireUserEqual(t, tt.want, got)
		})
	}
}

func TestUserRepository_FindByIdentity(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := db.NewUserRepository(testPool)

	yandexUID := uuid.NewString()
	existing := seedUser(t, nativeIdentity(), yandexIdentity(yandexUID))

	tests := []struct {
		name     string
		provider domain.Provider
		uid      string
		want     *domain.User
		wantErr  error
	}{
		{
			name:     "existing external identity",
			provider: domain.ProviderYandex,
			uid:      yandexUID,
			want:     existing,
		},
		{
			name:     "unknown provider user id",
			provider: domain.ProviderYandex,
			uid:      uuid.NewString(),
			wantErr:  domain.ErrNotFound,
		},
		{
			name:     "provider user id belongs to another provider",
			provider: domain.ProviderNative,
			uid:      yandexUID,
			wantErr:  domain.ErrNotFound,
		},
		{
			name:     "empty provider user id",
			provider: domain.ProviderYandex,
			uid:      "",
			wantErr:  domain.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := repo.FindByIdentity(ctx, tt.provider, tt.uid)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			requireUserEqual(t, tt.want, got)
		})
	}
}

func TestUserRepository_ClosedPool(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := db.NewUserRepository(closedPool(t))

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "Create",
			call: func() error { return repo.Create(ctx, newUser()) },
		},
		{
			name: "FindByEmail",
			call: func() error { _, err := repo.FindByEmail(ctx, "user@example.com"); return err },
		},
		{
			name: "FindByID",
			call: func() error { _, err := repo.FindByID(ctx, domain.UserID(newUUID())); return err },
		},
		{
			name: "FindByIdentity",
			call: func() error {
				_, err := repo.FindByIdentity(ctx, domain.ProviderYandex, "uid")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.call()

			require.Error(t, err)
			assert.NotErrorIs(t, err, domain.ErrNotFound)
			assert.NotErrorIs(t, err, domain.ErrAlreadyExists)
		})
	}
}
