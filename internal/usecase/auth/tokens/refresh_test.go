package tokens_test

import (
	"smart-fridge-backend/internal/usecase/auth/tokens"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRefreshTokenManager(t *testing.T) {
	type args struct {
		ttl time.Duration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid ttl",
			args: args{ttl: time.Hour},
		},
		{
			name:    "zero ttl",
			args:    args{ttl: 0},
			wantErr: true,
		},
		{
			name:    "negative ttl",
			args:    args{ttl: -time.Minute},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tokens.NewRefreshTokenManager(tt.args.ttl)

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

func TestRefreshTokenManager_Generate(t *testing.T) {
	const ttl = time.Hour

	m, _ := tokens.NewRefreshTokenManager(ttl)
	got := m.Generate()

	assert.NotEmpty(t, got.Body)
	assert.Equal(t, ttl, got.TTL)
}
