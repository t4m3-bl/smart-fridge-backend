package oauth_test

import (
	"smart-fridge-backend/internal/usecase/auth/oauth"
	"testing"

	"github.com/stretchr/testify/require"

	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/domain"
)

func TestNew_Use(t *testing.T) {
	t.Parallel()

	yandexCfg := config.OAuth{
		Providers: map[domain.Provider]config.OAuthProvider{
			domain.ProviderYandex: {
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURL:  "https://app.example/callback",
			},
		},
	}

	tests := []struct {
		name         string
		cfg          config.OAuth
		providerName string
		wantErr      error
	}{
		{
			name:         "configured provider",
			cfg:          yandexCfg,
			providerName: "yandex",
		},
		{
			name:         "known provider is not configured: nil providers",
			cfg:          config.OAuth{},
			providerName: "yandex",
			wantErr:      domain.ErrUnknownProvider,
		},
		{
			name:         "known provider is not configured: empty providers",
			cfg:          config.OAuth{Providers: map[domain.Provider]config.OAuthProvider{}},
			providerName: "yandex",
			wantErr:      domain.ErrUnknownProvider,
		},
		{
			name:         "unknown provider name",
			cfg:          yandexCfg,
			providerName: "unknown",
			wantErr:      domain.ErrUnknownProvider,
		},
		{
			name:         "empty provider name",
			cfg:          yandexCfg,
			providerName: "",
			wantErr:      domain.ErrUnknownProvider,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := oauth.New(tt.cfg)
			require.NotNil(t, p)

			got, err := p.Use(tt.providerName)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.IsType(t, &oauth.YandexProvider{}, got)
		})
	}
}
