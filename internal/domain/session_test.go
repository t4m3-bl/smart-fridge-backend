package domain_test

import (
	"smart-fridge-backend/internal/domain"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewSession(t *testing.T) {
	type args struct {
		userID    domain.ID
		tokenHash string
		expiresIn time.Duration
	}

	type testCase struct {
		name      string
		args      args
		errString string
	}

	baseID := domain.ID(uuid.New())

	tests := []testCase{
		{
			name: "successful session creation",
			args: args{
				userID:    baseID,
				tokenHash: "valid_refresh_token_hash",
				expiresIn: time.Hour,
			},
			errString: "",
		},
		{
			name: "empty token hash",
			args: args{
				userID:    baseID,
				tokenHash: "",
				expiresIn: time.Hour,
			},
			errString: "token validation error: empty token",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := assert.New(t)

			now := time.Now().UTC()
			got, err := domain.NewSession(tc.args.userID, tc.args.tokenHash, tc.args.expiresIn)

			if tc.errString != "" {
				if is.Error(err) {
					is.Equal(tc.errString, err.Error())
				}
				is.Nil(got)
				return
			}

			is.NoError(err)
			if is.NotNil(got) {
				is.NotEmpty(got.ID)
				is.Equal(tc.args.userID, got.UserID)
				is.Equal(tc.args.tokenHash, got.RefreshTokenHash)
				is.Equal(tc.args.expiresIn, got.ExpiresIn)
				is.WithinDuration(now, got.CreatedAt, time.Second)
			}
		})
	}
}

func TestSession_IsExpired(t *testing.T) {
	is := assert.New(t)
	userID := domain.ID(uuid.New())

	activeSession, err := domain.NewSession(userID, "hash", time.Hour)
	is.NoError(err)
	is.False(activeSession.IsExpired())

	expiredSession, err := domain.NewSession(userID, "hash", -time.Minute)
	is.NoError(err)
	is.True(expiredSession.IsExpired())
}
