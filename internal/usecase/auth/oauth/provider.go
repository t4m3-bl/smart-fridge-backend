package oauth

import (
	"context"
	"errors"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/dto"
)

var NotFound = errors.New("unknown OAuth provider")

type Provider interface {
	AuthCodeURL(state, verifier string) string
	Exchange(ctx context.Context, code string, verifier string) (dto.OAuthIdentity, error)
}

type BaseProvider struct {
	yandex *YandexProvider
}

func New(cfg config.Providers) *BaseProvider {
	return &BaseProvider{
		yandex: NewYandexProvider(cfg.RedirectURL, cfg.Yandex),
	}
}

func (p *BaseProvider) Use(providerName string) (Provider, error) {
	switch providerName {
	case "yandex":
		return p.yandex, nil
	default:
		return nil, NotFound
	}
}
