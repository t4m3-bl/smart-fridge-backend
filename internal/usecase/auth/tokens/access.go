package tokens

import (
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	issuer        = "smart-fridge-auth"
	audienceClaim = "smart-fridge-api"
)

type JWTTokenManager struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenManager(secret string, ttl time.Duration) *JWTTokenManager {
	return &JWTTokenManager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

type Claims struct {
	UserID domain.ID    `json:"uid"`
	Email  domain.Email `json:"email"`
	jwt.RegisteredClaims
}

func (m *JWTTokenManager) Generate(user *domain.User) (dto.Token, error) {
	now := time.Now().UTC()

	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   user.ID.String(),
			Audience:  jwt.ClaimStrings{audienceClaim},
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return dto.Token{}, fmt.Errorf("error signing token: %w", err)
	}

	return dto.Token{
		Body: signed,
		TTL:  m.ttl,
	}, nil
}
