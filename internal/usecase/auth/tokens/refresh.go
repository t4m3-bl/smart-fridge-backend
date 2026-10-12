package tokens

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"smart-fridge-backend/internal/dto"
	"time"
)

type RefreshTokenManager struct {
	ttl time.Duration
}

func NewRefreshTokenManager(ttl time.Duration) *RefreshTokenManager {
	return &RefreshTokenManager{ttl: ttl}
}

func (m *RefreshTokenManager) Generate() (dto.Token, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return dto.Token{}, fmt.Errorf("error generating refresh token: %w", err)
	}

	raw := base64.RawURLEncoding.EncodeToString(b)
	return dto.Token{
		Body: raw,
		TTL:  m.ttl,
	}, nil
}
