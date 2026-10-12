package service

import (
	"context"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/usecase/auth/tokens"
	"smart-fridge-backend/pkg/crypto/sha256"
	"time"
)

type UserRepository1 interface {
	FindByID(ctx context.Context, id domain.ID) (*domain.User, error)
}

type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	Rotate(ctx context.Context, oldRefresh, newRefresh string, expiresIn time.Duration) (*domain.Session, error)
	Revoke(ctx context.Context, tokenHash string) error
}

type SessionService struct {
	sessions SessionRepository
	users    UserRepository1

	access  *tokens.JWTTokenManager
	refresh *tokens.RefreshTokenManager
}

func NewSessionService(
	sessions SessionRepository,
	users UserRepository1,
	access *tokens.JWTTokenManager,
	refresh *tokens.RefreshTokenManager,
) *SessionService {
	return &SessionService{
		sessions: sessions,
		users:    users,
		access:   access,
		refresh:  refresh,
	}
}

func (s *SessionService) Create(ctx context.Context, user *domain.User) (dto.TokenPair, error) {
	refresh, err := s.refresh.Generate()
	if err != nil {
		return dto.TokenPair{}, err
	}

	access, err := s.access.Generate(user)
	if err != nil {
		return dto.TokenPair{}, fmt.Errorf("create session: %w", err)
	}

	session, err := domain.NewSession(user.ID, sha256.Hash(refresh.Body), refresh.TTL)
	if err != nil {
		return dto.TokenPair{}, fmt.Errorf("create session: %w", err)
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return dto.TokenPair{}, fmt.Errorf("create session: save: %w", err)
	}

	return dto.TokenPair{
		Refresh: refresh,
		Access:  access,
	}, nil
}

func (s *SessionService) Rotate(ctx context.Context, oldRefresh string) (dto.TokenPair, error) {
	newRefresh, err := s.refresh.Generate()
	if err != nil {
		return dto.TokenPair{}, err
	}

	newSession, err := s.sessions.Rotate(ctx,
		sha256.Hash(oldRefresh),
		sha256.Hash(newRefresh.Body),
		newRefresh.TTL,
	)
	if err != nil {
		return dto.TokenPair{}, fmt.Errorf("rotate session: %w", err)
	}

	user, err := s.users.FindByID(ctx, newSession.UserID)
	if err != nil {
		return dto.TokenPair{}, fmt.Errorf("rotate session: %w", err)
	}

	access, err := s.access.Generate(user)
	if err != nil {
		return dto.TokenPair{}, fmt.Errorf("rotate session: %w", err)
	}

	return dto.TokenPair{
		Access:  access,
		Refresh: newRefresh,
	}, nil
}

func (s *SessionService) Revoke(ctx context.Context, refresh *string) error {
	if refresh == nil {
		return nil
	}

	if err := s.sessions.Revoke(ctx, sha256.Hash(*refresh)); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}

	return nil
}
