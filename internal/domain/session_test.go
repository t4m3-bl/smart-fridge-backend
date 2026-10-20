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
		userID    domain.UserID
		tokenHash string
		expiresIn time.Duration
	}
	baseID := domain.UserID(uuid.New())

	tests := []struct {
		name   string
		args   args
		want   args
		errStr string
	}{
		{
			name: "successful session creation",
			args: args{
				userID:    baseID,
				tokenHash: "valid_refresh_token_hash",
				expiresIn: time.Hour,
			},
			want: args{
				userID:    baseID,
				tokenHash: "valid_refresh_token_hash",
				expiresIn: time.Hour,
			},
		},
		{
			name: "empty token hash",
			args: args{
				userID:    baseID,
				tokenHash: "",
				expiresIn: time.Hour,
			},
			errStr: "token validation error: empty token",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now().UTC()

			got, err := domain.NewSession(tt.args.userID, tt.args.tokenHash, tt.args.expiresIn)

			if tt.errStr != "" {
				assert.ErrorContains(t, err, tt.errStr)
				assert.Nil(t, got)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.NotEmpty(t, got.ID)
			assert.Equal(t, tt.want, args{
				userID:    got.UserID,
				tokenHash: got.RefreshTokenHash,
				expiresIn: got.ExpiresIn,
			})
			assert.WithinDuration(t, now, got.CreatedAt, time.Second)
		})
	}
}

func TestSession_IsExpired(t *testing.T) {
	type args struct {
		expiresIn time.Duration
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "active session",
			args: args{expiresIn: time.Hour},
			want: false,
		},
		{
			name: "expired session",
			args: args{expiresIn: -time.Minute},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := domain.UserID(uuid.New())

			session, err := domain.NewSession(userID, "hash", tt.args.expiresIn)

			assert.NoError(t, err)
			assert.Equal(t, tt.want, session.IsExpired())
		})
	}
}
