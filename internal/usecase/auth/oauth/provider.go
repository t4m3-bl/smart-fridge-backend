package oauth

import (
	"context"
	"net/http"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"time"
)

//go:generate go run go.uber.org/mock/mockgen -source=provider.go -destination=mocks/provider_mock.go

const httpTimeout = 10 * time.Second

type Provider interface {
	AuthCodeURL(state, verifier string) string
	Exchange(ctx context.Context, code string, verifier string) (dto.OAuthIdentity, error)
}

type BaseProvider struct {
	providers map[domain.Provider]Provider
}

func New(cfg config.OAuth) *BaseProvider {
	client := &http.Client{Timeout: httpTimeout}
	providers := make(map[domain.Provider]Provider)

	if pCfg, ok := cfg.Providers[domain.ProviderYandex]; ok {
		providers[domain.ProviderYandex] = NewYandexProvider(pCfg, client)
	}

	return &BaseProvider{providers: providers}
}

func (p *BaseProvider) Use(providerName string) (Provider, error) {
	providerType, err := domain.ToProvider(providerName)
	if err != nil {
		return nil, domain.ErrUnknownProvider
	}

	provider, ok := p.providers[providerType]
	if !ok {
		return nil, domain.ErrUnknownProvider
	}

	return provider, nil
}
