package oauth

import (
	"context"
	"fmt"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/transport/httpclient"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/yandex"
)

type YandexProvider struct {
	cfg *oauth2.Config
}

type userInfo struct {
	ID    string `json:"id"`
	Email string `json:"default_email"`
}

func NewYandexProvider(redirectURL string, cfg config.Provider) *YandexProvider {
	return &YandexProvider{
		cfg: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint:     yandex.Endpoint,
			RedirectURL:  redirectURL,
			Scopes:       cfg.Scopes,
		},
	}
}

func (p *YandexProvider) AuthCodeURL(state, verifier string) string {
	return p.cfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
}

func (p *YandexProvider) Exchange(ctx context.Context, code, verifier string) (dto.OAuthIdentity, error) {
	token, err := p.cfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	client := p.cfg.Client(ctx, token)
	info, err := httpclient.GetUserInfo[userInfo](ctx, client, "https://login.yandex.ru/info?format=json", token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}

	return dto.OAuthIdentity{
		ProviderUserID: info.ID,
		Email:          info.Email,
	}, nil
	0
}
