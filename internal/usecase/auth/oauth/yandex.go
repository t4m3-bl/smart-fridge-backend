package oauth

import (
	"context"
	"fmt"
	"net/http"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/transport/httpclient"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/yandex"
)

const yandexUserInfoURL = "https://login.yandex.ru/info?format=json"

type YandexProvider struct {
	cfg    *oauth2.Config
	client *http.Client
}

type userInfo struct {
	ID    string `json:"id"`
	Email string `json:"default_email"`
}

func NewYandexProvider(cfg config.OAuthProvider, client *http.Client) *YandexProvider {
	return &YandexProvider{
		cfg: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint:     yandex.Endpoint,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       cfg.Scopes,
		},
		client: client,
	}
}

func (p *YandexProvider) AuthCodeURL(state, verifier string) string {
	return p.cfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
}

func (p *YandexProvider) Exchange(ctx context.Context, code, verifier string) (dto.OAuthIdentity, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.client)

	token, err := p.cfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return dto.OAuthIdentity{}, fmt.Errorf("%w: get token: %w", domain.ErrProviderUnavailable, err)
	}

	info, err := httpclient.GetUserInfo[userInfo](ctx, p.client, yandexUserInfoURL, token.AccessToken)
	if err != nil {
		return dto.OAuthIdentity{}, fmt.Errorf("%w: get user info: %w", domain.ErrProviderUnavailable, err)
	}

	if info.ID == "" {
		return dto.OAuthIdentity{}, fmt.Errorf("%w: empty user id", domain.ErrProviderUnavailable)
	}

	return dto.OAuthIdentity{
		ProviderUserID: info.ID,
		Email:          info.Email,
	}, nil
}
