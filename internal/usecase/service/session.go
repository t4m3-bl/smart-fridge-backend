package service

import (
	"context"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/pkg/crypto/sha256"
	"time"
)

//go:generate go run go.uber.org/mock/mockgen -source=session.go -destination=mocks/session_mock.go

type UserFinder interface {
	FindByID(ctx context.Context, id domain.UserID) (*domain.User, error)
}

type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	Rotate(ctx context.Context, oldRefresh, newRefresh string, expiresIn time.Duration) (*domain.Session, error)
	Revoke(ctx context.Context, tokenHash string) error
}

type AccessTokenGenerator interface {
	Generate(user *domain.User) (dto.Token, error)
}

type RefreshTokenGenerator interface {
	Generate() dto.Token
}

type SessionService struct {
	sessions SessionRepository
	users    UserFinder

	access  AccessTokenGenerator
	refresh RefreshTokenGenerator
}

func NewSessionService(
	sessions SessionRepository,
	users UserFinder,
	access AccessTokenGenerator,
	refresh RefreshTokenGenerator,
) *SessionService {
	return &SessionService{
		sessions: sessions,
		users:    users,
		access:   access,
		refresh:  refresh,
	}
}

func (s *SessionService) Create(ctx context.Context, user *domain.User) (dto.Token, error) {
	refresh := s.refresh.Generate()

	session, err := domain.NewSession(user.ID, sha256.Hash(refresh.Body), refresh.TTL)
	if err != nil {
		return dto.Token{}, fmt.Errorf("create session: %w", err)
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return dto.Token{}, fmt.Errorf("create session: save: %w", err)
	}

	return refresh, nil
}

func (s *SessionService) Rotate(ctx context.Context, oldRefresh string) (dto.TokenPair, error) {
	newRefresh := s.refresh.Generate()

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
