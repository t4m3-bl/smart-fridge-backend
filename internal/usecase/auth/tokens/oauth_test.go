package tokens_test

import (
	"regexp"
	"smart-fridge-backend/internal/usecase/auth/tokens"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var urlSafeRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func TestNewOauthTokensManger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ttl     time.Duration
		wantErr bool
	}{
		{
			name: "valid ttl",
			ttl:  10 * time.Minute,
		},
		{
			name: "minimal positive ttl",
			ttl:  time.Nanosecond,
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

			got, err := tokens.NewOauthTokensManger(tt.ttl)

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

func TestOauthTokensManger_Generate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{
			name: "ttl is passed through",
			ttl:  10 * time.Minute,
		},
		{
			name: "different ttl",
			ttl:  time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m, err := tokens.NewOauthTokensManger(tt.ttl)
			require.NoError(t, err)

			got := m.Generate()

			assert.Equal(t, tt.ttl, got.TTL)

			assert.Len(t, got.State, 22)
			assert.Regexp(t, urlSafeRe, got.State)

			assert.GreaterOrEqual(t, len(got.Verifier), 43)
			assert.LessOrEqual(t, len(got.Verifier), 128)
			assert.Regexp(t, urlSafeRe, got.Verifier)
		})
	}
}

func TestOauthTokensManger_Generate_Unique(t *testing.T) {
	t.Parallel()

	const iterations = 100

	m, err := tokens.NewOauthTokensManger(time.Minute)
	require.NoError(t, err)

	states := make(map[string]struct{}, iterations)
	verifiers := make(map[string]struct{}, iterations)

	for range iterations {
		got := m.Generate()

		states[got.State] = struct{}{}
		verifiers[got.Verifier] = struct{}{}
	}

	assert.Len(t, states, iterations, "state must be unique per call")
	assert.Len(t, verifiers, iterations, "verifier must be unique per call")
}
