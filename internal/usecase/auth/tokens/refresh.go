package tokens

import (
	"fmt"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/pkg/auth/token"
	"time"
)

type RefreshTokenManager struct {
	ttl time.Duration
}

func NewRefreshTokenManager(ttl time.Duration) (*RefreshTokenManager, error) {
	if ttl <= 0 {
		return nil, fmt.Errorf("refresh token manager: ttl must be positive, got %v", ttl)
	}

	return &RefreshTokenManager{ttl: ttl}, nil
}

func (m *RefreshTokenManager) Generate() dto.Token {
	raw := token.GenerateNonce(32)
	return dto.Token{
		Body: raw,
		TTL:  m.ttl,
	}
}
