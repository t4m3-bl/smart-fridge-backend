package tokens

import (
	"fmt"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/pkg/auth/token"
	"time"

	"golang.org/x/oauth2"
)

type OauthTokensManger struct {
	ttl time.Duration
}

func NewOauthTokensManger(ttl time.Duration) (*OauthTokensManger, error) {
	if ttl <= 0 {
		return nil, fmt.Errorf("oauth token manager: ttl must be positive, got %v", ttl)
	}

	return &OauthTokensManger{ttl}, nil
}

func (o *OauthTokensManger) Generate() dto.OAuthTokenPair {
	state := token.GenerateNonce(16)
	verifier := oauth2.GenerateVerifier()

	return dto.OAuthTokenPair{
		Verifier: verifier,
		State:    state,
		TTL:      o.ttl,
	}
}
