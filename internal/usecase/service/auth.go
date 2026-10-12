package service

import (
	"context"
	"errors"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
)

type PasswordHasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) error
}

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
}

type AuthService struct {
	user   UserRepository
	hasher PasswordHasher

	session *SessionService
}

func NewAuthService(auth UserRepository, hasher PasswordHasher, session *SessionService) *AuthService {
	return &AuthService{
		user:    auth,
		hasher:  hasher,
		session: session,
	}
}

func (s *AuthService) SignUp(ctx context.Context, input dto.CredentialsInput) (dto.TokenPair, error) {
	pwHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return dto.TokenPair{}, fmt.Errorf("sign up: %w", err)
	}

	user, err := domain.NewUser(input.Email, pwHash)
	if err != nil {
		return dto.TokenPair{}, fmt.Errorf("sign up: %w", err)
	}

	if err = s.user.Create(ctx, user); err != nil {
		return dto.TokenPair{}, fmt.Errorf("sign up: %w", err)
	}

	pair, err := s.session.Create(ctx, user)
	if err != nil {
		return dto.TokenPair{}, fmt.Errorf("sign up: %w", err)
	}

	return pair, nil
}

func (s *AuthService) SignIn(ctx context.Context, input dto.CredentialsInput) (dto.TokenPair, error) {
	email, err := domain.NewEmail(input.Email)
	if err != nil {
		return dto.TokenPair{}, fmt.Errorf("sign in: %w", domain.ErrInvalidCredentials)
	}

	user, err := s.user.FindByEmail(ctx, string(email))
	if errors.Is(err, domain.ErrNotFound) {
		return dto.TokenPair{}, fmt.Errorf("sign in: %w", domain.ErrInvalidCredentials)
	}
	if err != nil {
		return dto.TokenPair{}, fmt.Errorf("sign in: %w", err)
	}

	if err := s.hasher.Compare(user.PasswordHash, input.Password); err != nil {
		return dto.TokenPair{}, fmt.Errorf("sign in: %w: %w", domain.ErrInvalidCredentials, err)
	}

	pair, err := s.session.Create(ctx, user)
	if err != nil {
		return dto.TokenPair{}, fmt.Errorf("sign in: %w", err)
	}

	return pair, nil
}
