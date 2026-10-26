package tokens_test

import (
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/usecase/auth/tokens"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewTokenManager(t *testing.T) {
	t.Parallel()

	const secret = "a-string-secret-at-least-256-bits-long"

	type args struct {
		secret string
		ttl    time.Duration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid secret and ttl",
			args: args{secret: secret, ttl: time.Hour},
		},
		{
			name:    "empty secret",
			args:    args{secret: "", ttl: time.Hour},
			wantErr: true,
		},
		{
			name:    "secret too short",
			args:    args{secret: "short", ttl: time.Hour},
			wantErr: true,
		},
		{
			name:    "zero ttl",
			args:    args{secret: secret, ttl: 0},
			wantErr: true,
		},
		{
			name:    "negative ttl",
			args:    args{secret: secret, ttl: -time.Minute},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tokens.NewTokenManager(tt.args.secret, tt.args.ttl)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestJWTTokenManager_Generate(t *testing.T) {
	t.Parallel()

	const (
		secret        = "a-string-secret-at-least-256-bits-long"
		issuer        = "smart-fridge-auth"
		audienceClaim = "smart-fridge-api"
	)

	m, err := tokens.NewTokenManager(secret, time.Hour)
	assert.NoError(t, err)

	user, _ := domain.NewUser("example@test.com")

	type args struct {
		user *domain.User
	}
	tests := []struct {
		name   string
		args   args
		want   dto.Token
		errStr string
	}{
		{
			name: "successful token generation",
			args: args{user: user},
			want: dto.Token{TTL: time.Hour},
		},
		{
			name:   "nil user",
			args:   args{user: nil},
			errStr: "generate jwt token: user is nil",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			now := time.Now().UTC()
			got, err := m.Generate(tt.args.user)

			if tt.errStr != "" {
				assert.ErrorContains(t, err, tt.errStr)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, got.Body)
			assert.Equal(t, tt.want.TTL, got.TTL)

			parsed, parseErr := jwt.ParseWithClaims(got.Body, &tokens.Claims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})
			assert.NoError(t, parseErr)
			assert.NotNil(t, parsed)
			assert.True(t, parsed.Valid)

			claims, ok := parsed.Claims.(*tokens.Claims)
			assert.True(t, ok)

			assert.Equal(t, tt.args.user.ID, claims.UserID)
			assert.Equal(t, tt.args.user.Email, claims.Email)
			assert.Equal(t, issuer, claims.Issuer)
			assert.Equal(t, jwt.ClaimStrings{audienceClaim}, claims.Audience)
			assert.Equal(t, uuid.UUID(tt.args.user.ID).String(), claims.Subject)
			assert.NotEmpty(t, claims.ID)

			assert.WithinDuration(t, now.Add(time.Hour), claims.ExpiresAt.Time, time.Second)
			assert.WithinDuration(t, now, claims.IssuedAt.Time, time.Second)
			assert.WithinDuration(t, now, claims.NotBefore.Time, time.Second)
		})
	}
}
