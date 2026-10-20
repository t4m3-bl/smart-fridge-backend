package domain

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID               SessionID
	UserID           UserID
	RefreshTokenHash string
	ExpiresIn        time.Duration
	CreatedAt        time.Time
}

func NewSession(userID UserID, tokenHash string, expiresIn time.Duration) (*Session, error) {
	id := uuid.Must(uuid.NewV7())

	if tokenHash == "" {
		return nil, NewValidationErr("token", "empty token")
	}

	return &Session{
		ID:               SessionID(id),
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

type SessionID uuid.UUID
