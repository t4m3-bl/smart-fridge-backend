package oauth

import (
	"context"
	"net/http"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"time"
)

const httpTimeout = 10 * time.Second

type Provider interface {
	AuthCodeURL(state, verifier string) string
	Exchange(ctx context.Context, code string, verifier string) (dto.OAuthIdentity, error)
}

type BaseProvider struct {
	providers map[string]Provider
}

func New(cfg config.OAuth) *BaseProvider {
	client := &http.Client{Timeout: httpTimeout}
	providers := make(map[string]Provider)

	if cfg.Yandex.Enabled() {
		providers[domain.ProviderYandex.String()] = NewYandexProvider(cfg.Yandex, client)
	}

	return &BaseProvider{providers: providers}
}

func (p *BaseProvider) Use(providerName string) (Provider, error) {
	provider, ok := p.providers[providerName]
	if !ok {
		return nil, domain.ErrUnknownProvider
	}

	return provider, nil
}
