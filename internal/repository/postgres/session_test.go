package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/repository/postgres"
)

func createTestUser(t *testing.T, ctx context.Context, tx pgx.Tx) domain.ID {
	t.Helper()

	user := &domain.User{
		ID:           domain.ID(uuid.New()),
		Email:        domain.Email(uuid.NewString() + "@example.com"),
		PasswordHash: "hashed_password",
	}
	require.NoError(t, postgres.NewUserRepository(tx).Create(ctx, user))

	return user.ID
}

func newTestSession(userID domain.ID, tokenHash string, createdAt time.Time, expiresIn time.Duration) *domain.Session {
	return &domain.Session{
		ID:               domain.ID(uuid.New()),
		UserID:           userID,
		RefreshTokenHash: tokenHash,
		ExpiresIn:        expiresIn,
		CreatedAt:        createdAt,
	}
}

func TestSessionRepository_Create(t *testing.T) {
	ctx := context.Background()

	tx, err := testPool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(ctx) })

	userID := createTestUser(t, ctx, tx)
	now := time.Now().UTC()

	repo := postgres.NewSessionRepository(tx)
	require.NoError(t, repo.Create(ctx, newTestSession(userID, "existing_hash", now, time.Hour)))

	type TestCase struct {
		name    string
		session *domain.Session
		wantErr bool
	}

	tests := []TestCase{
		{
			name:    "successful creation",
			session: newTestSession(userID, "new_hash", now, time.Hour),
			wantErr: false,
		},
		{
			name:    "refresh token hash already exists",
			session: newTestSession(userID, "existing_hash", now, time.Hour),
			wantErr: true,
		},
		{
			name:    "unknown user",
			session: newTestSession(domain.ID(uuid.New()), "orphan_hash", now, time.Hour),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := require.New(t)

			subTx, err := tx.Begin(ctx)
			is.NoError(err)
			t.Cleanup(func() { _ = subTx.Rollback(ctx) })

			repo := postgres.NewSessionRepository(subTx)

			err = repo.Create(ctx, tc.session)
			if tc.wantErr {
				is.Error(err)
			} else {
				is.NoError(err)
			}
		})
	}
}

func TestSessionRepository_Rotate(t *testing.T) {
	ctx := context.Background()

	tx, err := testPool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(ctx) })

	userID := createTestUser(t, ctx, tx)
	now := time.Now().UTC()

	repo := postgres.NewSessionRepository(tx)
	require.NoError(t, repo.Create(ctx, newTestSession(userID, "active_hash", now, time.Hour)))
	require.NoError(t, repo.Create(ctx, newTestSession(userID, "expired_hash", now.Add(-2*time.Hour), time.Hour)))
	require.NoError(t, repo.Create(ctx, newTestSession(userID, "revoked_hash", now, time.Hour)))
	require.NoError(t, repo.Revoke(ctx, "revoked_hash"))

	type TestCase struct {
		name       string
		oldRefresh string
		err        error
	}

	tests := []TestCase{
		{
			name:       "successful rotation",
			oldRefresh: "active_hash",
			err:        nil,
		},
		{
			name:       "unknown refresh token",
			oldRefresh: "unknown_hash",
			err:        domain.ErrInvalidSession,
		},
		{
			name:       "expired session",
			oldRefresh: "expired_hash",
			err:        domain.ErrInvalidSession,
		},
		{
			name:       "revoked session",
			oldRefresh: "revoked_hash",
			err:        domain.ErrInvalidSession,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := require.New(t)

			subTx, err := tx.Begin(ctx)
			is.NoError(err)
			t.Cleanup(func() { _ = subTx.Rollback(ctx) })

			repo := postgres.NewSessionRepository(subTx)

			got, err := repo.Rotate(ctx, tc.oldRefresh, "rotated_hash", time.Hour)
			if tc.err != nil {
				is.ErrorIs(err, tc.err)
				is.Nil(got)
				return
			}

			is.NoError(err)
			is.NotNil(got)
			is.Equal(userID, got.UserID)
			is.Equal("rotated_hash", got.RefreshTokenHash)
			is.Equal(time.Hour, got.ExpiresIn)

			_, err = repo.Rotate(ctx, tc.oldRefresh, "rotated_again_hash", time.Hour)
			is.ErrorIs(err, domain.ErrInvalidSession)

			_, err = repo.Rotate(ctx, "rotated_hash", "rotated_twice_hash", time.Hour)
			is.NoError(err)
		})
	}
}

func TestSessionRepository_Revoke(t *testing.T) {
	ctx := context.Background()

	tx, err := testPool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(ctx) })

	userID := createTestUser(t, ctx, tx)
	now := time.Now().UTC()

	repo := postgres.NewSessionRepository(tx)
	require.NoError(t, repo.Create(ctx, newTestSession(userID, "active_hash", now, time.Hour)))
	require.NoError(t, repo.Create(ctx, newTestSession(userID, "revoked_hash", now, time.Hour)))
	require.NoError(t, repo.Revoke(ctx, "revoked_hash"))

	type TestCase struct {
		name      string
		tokenHash string
	}

	tests := []TestCase{
		{
			name:      "active session",
			tokenHash: "active_hash",
		},
		{
			name:      "already revoked session",
			tokenHash: "revoked_hash",
		},
		{
			name:      "unknown refresh token",
			tokenHash: "unknown_hash",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := require.New(t)

			subTx, err := tx.Begin(ctx)
			is.NoError(err)
			t.Cleanup(func() { _ = subTx.Rollback(ctx) })

			repo := postgres.NewSessionRepository(subTx)

			is.NoError(repo.Revoke(ctx, tc.tokenHash))

			_, err = repo.Rotate(ctx, tc.tokenHash, "rotated_hash", time.Hour)
			is.ErrorIs(err, domain.ErrInvalidSession)
		})
	}
}
