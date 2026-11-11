package tokens

import (
	"errors"
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

func NewTokenManager(secret string, ttl time.Duration) (*JWTTokenManager, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("jwt token manager: secret must be at least 32 bytes, got %d", len(secret))
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("jwt token manager: ttl must be positive, got %v", ttl)
	}
	return &JWTTokenManager{
		secret: []byte(secret),
		ttl:    ttl,
	}, nil
}

type Claims struct {
	jwt.RegisteredClaims
}

func (m *JWTTokenManager) Generate(user *domain.User) (dto.Token, error) {
	if user == nil {
		return dto.Token{}, errors.New("generate jwt token: user is nil")
	}

	now := time.Now().UTC()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   uuid.UUID(user.ID).String(),
			Audience:  jwt.ClaimStrings{audienceClaim},
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
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

func (m *JWTTokenManager) Verify(tokenString string) (domain.UserID, error) {
	if tokenString == "" {
		return domain.UserID{}, domain.ErrInvalidCredentials
	}

	claims := new(Claims)
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing algorithm: %v", token.Header["alg"])
		}

		return m.secret, nil
	},
		jwt.WithValidMethods(
			[]string{jwt.SigningMethodHS256.Alg()},
		),
		jwt.WithIssuer(issuer),
		jwt.WithAudience(audienceClaim),
		jwt.WithExpirationRequired(),
		jwt.WithNotBeforeRequired(),
		jwt.WithIssuedAt(),
		jwt.WithStrictDecoding(),
	)
	if err != nil || token == nil || !token.Valid {
		return domain.UserID{}, domain.ErrInvalidCredentials
	}

	if claims.ExpiresAt.Before(claims.IssuedAt.Time) {
		return domain.UserID{}, domain.ErrInvalidCredentials
	}

	id, err := uuid.Parse(claims.Subject)
	if err != nil || id == uuid.Nil {
		return domain.UserID{}, domain.ErrInvalidCredentials
	}

	return domain.UserID(id), nil
}
