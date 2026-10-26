package tokens_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"smart-fridge-backend/internal/usecase/auth/tokens"
)

func TestNewRefreshTokenManager(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ttl     time.Duration
		wantErr bool
	}{
		{
			name: "valid ttl",
			ttl:  time.Hour,
		},
		{
			name:    "zero ttl",
			ttl:     0,
			wantErr: true,
		},
		{
			name:    "negative ttl",
			ttl:     -time.Minute,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tokens.NewRefreshTokenManager(tt.ttl)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)

				return
			}

			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func TestRefreshTokenManager_Generate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{
			name: "one hour ttl",
			ttl:  time.Hour,
		},
		{
			name: "thirty days ttl",
			ttl:  720 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m, err := tokens.NewRefreshTokenManager(tt.ttl)
			require.NoError(t, err)

			got := m.Generate()

			// 32 random bytes -> 43 chars in raw base64url
			assert.Len(t, got.Body, 43)
			assert.Regexp(t, urlSafeRe, got.Body)
			assert.Equal(t, tt.ttl, got.TTL)
		})
	}
}

func TestRefreshTokenManager_Generate_Unique(t *testing.T) {
	t.Parallel()

	const iterations = 100

	m, err := tokens.NewRefreshTokenManager(time.Hour)
	require.NoError(t, err)

	seen := make(map[string]struct{}, iterations)

	for range iterations {
		seen[m.Generate().Body] = struct{}{}
	}

	assert.Len(t, seen, iterations, "refresh token must be unique per call")
}
