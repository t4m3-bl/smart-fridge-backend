//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"smart-fridge-backend/internal/domain"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "smart-fridge-backend/internal/repository/postgres"
)

type sessionRow struct {
	UserID    uuid.UUID
	ExpiresAt time.Time
	RevokedAt *time.Time
}

func findSession(t *testing.T, refreshTokenHash string) *sessionRow {
	t.Helper()

	var row sessionRow
	err := testPool.QueryRow(context.Background(),
		`SELECT user_id, expires_at, revoked_at FROM sessions WHERE refresh_token_hash = $1`,
		refreshTokenHash,
	).Scan(&row.UserID, &row.ExpiresAt, &row.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	require.NoError(t, err)

	return &row
}

func newSession(t *testing.T, userID domain.UserID) *domain.Session {
	t.Helper()

	session, err := domain.NewSession(userID, "refresh-"+uuid.NewString(), time.Hour)
	require.NoError(t, err)

	return session
}

func seedSession(t *testing.T, repo *db.SessionRepository, userID domain.UserID) *domain.Session {
	t.Helper()

	session := newSession(t, userID)
	require.NoError(t, repo.Create(context.Background(), session))

	return session
}

func TestSessionRepository_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		prepare func(t *testing.T, repo *db.SessionRepository, userID domain.UserID) *domain.Session
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "valid session",
			prepare: func(t *testing.T, _ *db.SessionRepository, userID domain.UserID) *domain.Session {
				t.Helper()
				return newSession(t, userID)
			},
			wantErr: require.NoError,
		},
		{
			name: "user does not exist",
			prepare: func(t *testing.T, _ *db.SessionRepository, _ domain.UserID) *domain.Session {
				t.Helper()
				return newSession(t, domain.UserID(newUUID()))
			},
			wantErr: require.Error,
		},
		{
			name: "duplicate session id",
			prepare: func(t *testing.T, repo *db.SessionRepository, userID domain.UserID) *domain.Session {
				t.Helper()

				existing := seedSession(t, repo, userID)
				duplicate := *existing
				duplicate.RefreshTokenHash = "refresh-" + uuid.NewString()

				return &duplicate
			},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := db.NewSessionRepository(testPool)
			user := seedUser(t)
			session := tt.prepare(t, repo, user.ID)

			err := repo.Create(context.Background(), session)

			tt.wantErr(t, err)
			if err != nil {
				return
			}

			row := findSession(t, session.RefreshTokenHash)
			require.NotNil(t, row)
			assert.Equal(t, uuid.UUID(session.UserID), row.UserID)
			assert.WithinDuration(t, session.CreatedAt.Add(session.ExpiresIn), row.ExpiresAt, time.Millisecond)
			assert.Nil(t, row.RevokedAt)
		})
	}
}

func TestSessionRepository_Rotate(t *testing.T) {
	t.Parallel()

	const expiresIn = 24 * time.Hour

	tests := []struct {
		name        string
		prepare     func(t *testing.T, repo *db.SessionRepository, userID domain.UserID) (oldHash string)
		expiresIn   time.Duration
		canceledCtx bool
		wantErr     require.ErrorAssertionFunc
	}{
		{
			name: "active session",
			prepare: func(t *testing.T, repo *db.SessionRepository, userID domain.UserID) string {
				t.Helper()
				return seedSession(t, repo, userID).RefreshTokenHash
			},
			expiresIn: expiresIn,
			wantErr:   require.NoError,
		},
		{
			name: "unknown token",
			prepare: func(*testing.T, *db.SessionRepository, domain.UserID) string {
				t.Helper()
				return "missing-" + uuid.NewString()
			},
			expiresIn: expiresIn,
			wantErr:   errorIs(domain.ErrInvalidSession),
		},
		{
			name: "already revoked session",
			prepare: func(t *testing.T, repo *db.SessionRepository, userID domain.UserID) string {
				t.Helper()

				session := seedSession(t, repo, userID)
				require.NoError(t, repo.Revoke(context.Background(), session.RefreshTokenHash))

				return session.RefreshTokenHash
			},
			expiresIn: expiresIn,
			wantErr:   errorIs(domain.ErrInvalidSession),
		},
		{
			name: "expired session",
			prepare: func(t *testing.T, repo *db.SessionRepository, userID domain.UserID) string {
				t.Helper()

				session := newSession(t, userID)
				session.CreatedAt = time.Now().Add(-2 * time.Hour)
				require.NoError(t, repo.Create(context.Background(), session))

				return session.RefreshTokenHash
			},
			expiresIn: expiresIn,
			wantErr:   errorIs(domain.ErrInvalidSession),
		},
		{
			name: "replayed token",
			prepare: func(t *testing.T, repo *db.SessionRepository, userID domain.UserID) string {
				t.Helper()

				session := seedSession(t, repo, userID)
				_, err := repo.Rotate(context.Background(), session.RefreshTokenHash, "first-"+uuid.NewString(), expiresIn)
				require.NoError(t, err)

				return session.RefreshTokenHash
			},
			expiresIn: expiresIn,
			wantErr:   errorIs(domain.ErrInvalidSession),
		},
		{
			name: "invalid new session leaves old session untouched",
			prepare: func(t *testing.T, repo *db.SessionRepository, userID domain.UserID) string {
				t.Helper()
				return seedSession(t, repo, userID).RefreshTokenHash
			},
			expiresIn: -time.Minute,
			wantErr:   require.Error,
		},
		{
			name: "canceled context",
			prepare: func(t *testing.T, repo *db.SessionRepository, userID domain.UserID) string {
				t.Helper()
				return seedSession(t, repo, userID).RefreshTokenHash
			},
			expiresIn:   expiresIn,
			canceledCtx: true,
			wantErr:     require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := db.NewSessionRepository(testPool)
			user := seedUser(t)
			oldHash := tt.prepare(t, repo, user.ID)
			before := findSession(t, oldHash)

			ctx := context.Background()
			if tt.canceledCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			got, err := repo.Rotate(ctx, oldHash, "new-"+uuid.NewString(), tt.expiresIn)

			tt.wantErr(t, err)

			after := findSession(t, oldHash)

			if err != nil {
				assert.Nil(t, got)
				if before != nil {
					require.NotNil(t, after)
					assert.Equal(t, before.RevokedAt, after.RevokedAt)
				}
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, user.ID, got.UserID)
			assert.Equal(t, tt.expiresIn, got.ExpiresIn)

			require.NotNil(t, after)
			assert.NotNil(t, after.RevokedAt, "old session must be revoked")

			row := findSession(t, got.RefreshTokenHash)
			require.NotNil(t, row, "new session must be stored")
			assert.Equal(t, uuid.UUID(user.ID), row.UserID)
			assert.WithinDuration(t, got.CreatedAt.Add(got.ExpiresIn), row.ExpiresAt, time.Millisecond)
			assert.Nil(t, row.RevokedAt)
		})
	}
}

func TestSessionRepository_Rotate_ConcurrentReuse(t *testing.T) {
	t.Parallel()

	const attempts = 4

	repo := db.NewSessionRepository(testPool)
	session := seedSession(t, repo, seedUser(t).ID)

	errs := make([]error, attempts)

	var wg sync.WaitGroup
	for i := range attempts {
		wg.Go(func() {
			_, errs[i] = repo.Rotate(context.Background(), session.RefreshTokenHash, "new-"+uuid.NewString(), time.Hour)
		})
	}
	wg.Wait()

	succeeded := 0
	for _, err := range errs {
		if err == nil {
			succeeded++
			continue
		}
		assert.ErrorIs(t, err, domain.ErrInvalidSession)
	}

	assert.Equal(t, 1, succeeded)
}

func TestSessionRepository_Revoke(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		prepare     func(t *testing.T, repo *db.SessionRepository, userID domain.UserID) (hash string)
		canceledCtx bool
		wantErr     require.ErrorAssertionFunc
	}{
		{
			name: "active session",
			prepare: func(t *testing.T, repo *db.SessionRepository, userID domain.UserID) string {
				t.Helper()
				return seedSession(t, repo, userID).RefreshTokenHash
			},
			wantErr: require.NoError,
		},
		{
			name: "unknown token is not an error",
			prepare: func(*testing.T, *db.SessionRepository, domain.UserID) string {
				t.Helper()
				return "missing-" + uuid.NewString()
			},
			wantErr: require.NoError,
		},
		{
			name: "already revoked session",
			prepare: func(t *testing.T, repo *db.SessionRepository, userID domain.UserID) string {
				t.Helper()

				session := seedSession(t, repo, userID)
				require.NoError(t, repo.Revoke(context.Background(), session.RefreshTokenHash))

				return session.RefreshTokenHash
			},
			wantErr: require.NoError,
		},
		{
			name: "canceled context",
			prepare: func(t *testing.T, repo *db.SessionRepository, userID domain.UserID) string {
				t.Helper()
				return seedSession(t, repo, userID).RefreshTokenHash
			},
			canceledCtx: true,
			wantErr:     require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := db.NewSessionRepository(testPool)
			hash := tt.prepare(t, repo, seedUser(t).ID)
			before := findSession(t, hash)

			ctx := context.Background()
			if tt.canceledCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			err := repo.Revoke(ctx, hash)

			tt.wantErr(t, err)

			after := findSession(t, hash)
			if before == nil {
				assert.Nil(t, after)
				return
			}
			require.NotNil(t, after)

			switch {
			case err != nil:
				assert.Equal(t, before.RevokedAt, after.RevokedAt)
			case before.RevokedAt == nil:
				assert.NotNil(t, after.RevokedAt, "session must be revoked")
			default:
				assert.Equal(t, before.RevokedAt, after.RevokedAt)
			}
		})
	}
}

func TestSessionRepository_ClosedPool(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := db.NewSessionRepository(closedPool(t))
	session := newSession(t, domain.UserID(newUUID()))

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "Create",
			call: func() error { return repo.Create(ctx, session) },
		},
		{
			name: "Rotate",
			call: func() error { _, err := repo.Rotate(ctx, "old", "new", time.Hour); return err },
		},
		{
			name: "Revoke",
			call: func() error { return repo.Revoke(ctx, "token") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.call()

			require.Error(t, err)
			assert.NotErrorIs(t, err, domain.ErrInvalidSession)
		})
	}
}
