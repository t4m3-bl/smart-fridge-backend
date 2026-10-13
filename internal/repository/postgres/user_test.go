package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/repository/postgres"
)

func TestUserRepository_Create(t *testing.T) {
	ctx := context.Background()

	tx, err := testPool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(ctx) })

	existingID := domain.ID(uuid.New())
	existingEmail := domain.Email("existing@example.com")
	passwordHash := "hashed_password"

	repo := postgres.NewUserRepository(tx)
	require.NoError(t, repo.Create(ctx, &domain.User{
		ID:           existingID,
		Email:        existingEmail,
		PasswordHash: passwordHash,
	}))

	type TestCase struct {
		name string
		user *domain.User
		err  error
	}

	tests := []TestCase{
		{
			name: "successful creation",
			user: &domain.User{
				ID:           domain.ID(uuid.New()),
				Email:        domain.Email("new@example.com"),
				PasswordHash: passwordHash,
			},
			err: nil,
		},
		{
			name: "email already exists",
			user: &domain.User{
				ID:           domain.ID(uuid.New()),
				Email:        existingEmail,
				PasswordHash: passwordHash,
			},
			err: domain.ErrAlreadyExists,
		},
		{
			name: "id already exists",
			user: &domain.User{
				ID:           existingID,
				Email:        domain.Email("another@example.com"),
				PasswordHash: passwordHash,
			},
			err: domain.ErrAlreadyExists,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := require.New(t)

			subTx, err := tx.Begin(ctx)
			is.NoError(err)
			t.Cleanup(func() { _ = subTx.Rollback(ctx) })

			repo := postgres.NewUserRepository(subTx)

			err = repo.Create(ctx, tc.user)
			if tc.err != nil {
				is.ErrorIs(err, tc.err)
			} else {
				is.NoError(err)
			}
		})
	}
}

func TestUserRepository_FindByEmail(t *testing.T) {
	ctx := context.Background()

	tx, err := testPool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(ctx) })

	existing := &domain.User{
		ID:           domain.ID(uuid.New()),
		Email:        domain.Email("find-by-email@example.com"),
		PasswordHash: "hashed_password",
	}

	repo := postgres.NewUserRepository(tx)
	require.NoError(t, repo.Create(ctx, existing))

	type TestCase struct {
		name  string
		email string
		want  *domain.User
		err   error
	}

	tests := []TestCase{
		{
			name:  "existing user",
			email: "find-by-email@example.com",
			want:  existing,
			err:   nil,
		},
		{
			name:  "email in different case",
			email: "Find-By-Email@Example.COM",
			want:  existing,
			err:   nil,
		},
		{
			name:  "unknown email",
			email: "unknown@example.com",
			want:  nil,
			err:   domain.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := require.New(t)

			got, err := repo.FindByEmail(ctx, tc.email)
			if tc.err != nil {
				is.ErrorIs(err, tc.err)
				is.Nil(got)
				return
			}

			is.NoError(err)
			is.Equal(tc.want, got)
		})
	}
}

func TestUserRepository_FindByID(t *testing.T) {
	ctx := context.Background()

	tx, err := testPool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(ctx) })

	existing := &domain.User{
		ID:           domain.ID(uuid.New()),
		Email:        domain.Email("find-by-id@example.com"),
		PasswordHash: "hashed_password",
	}

	repo := postgres.NewUserRepository(tx)
	require.NoError(t, repo.Create(ctx, existing))

	type TestCase struct {
		name string
		id   domain.ID
		want *domain.User
		err  error
	}

	tests := []TestCase{
		{
			name: "existing user",
			id:   existing.ID,
			want: existing,
			err:  nil,
		},
		{
			name: "unknown id",
			id:   domain.ID(uuid.New()),
			want: nil,
			err:  domain.ErrNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := require.New(t)

			got, err := repo.FindByID(ctx, tc.id)
			if tc.err != nil {
				is.ErrorIs(err, tc.err)
				is.Nil(got)
				return
			}

			is.NoError(err)
			is.Equal(tc.want, got)
		})
	}
}
