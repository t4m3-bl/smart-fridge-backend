package domain

import (
	"time"
)

type Session struct {
	ID               ID
	UserID           ID
	RefreshTokenHash string
	ExpiresIn        time.Duration
	CreatedAt        time.Time
}

func NewSession(userID ID, tokenHash string, expiresIn time.Duration) (*Session, error) {
	id, err := NewID()
	if err != nil {
		return nil, err
	}

	if tokenHash == "" {
		return nil, NewValidationErr("token", "empty token")
	}

	return &Session{
		ID:               id,
		UserID:           userID,
		RefreshTokenHash: tokenHash,
		ExpiresIn:        expiresIn,
		CreatedAt:        time.Now().UTC(),
	}, nil
}

func (s *Session) IsExpired() bool {
	now := time.Now().UTC()
	return s.CreatedAt.Add(s.ExpiresIn).Before(now)
}
