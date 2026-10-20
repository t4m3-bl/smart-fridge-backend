package service

import (
	"context"
	"errors"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/usecase/auth/oauth"
)

//go:generate go run go.uber.org/mock/mockgen -source=auth.go -destination=mocks/auth_mock.go

type SessionCreator interface {
	Create(ctx context.Context, user *domain.User) (dto.Token, error)
}

type PasswordHasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) error
}

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error)
	FindByIdentity(ctx context.Context, provider domain.Provider, uid string) (*domain.User, error)
}

type OauthTokensGenerator interface {
	Generate() dto.OAuthTokenPair
}

type AuthService struct {
	user   UserRepository
	hasher PasswordHasher

	session SessionCreator

	provider oauth.BaseProvider
	oAuth    OauthTokensGenerator
	access   AccessTokenGenerator
}

func NewAuthService(auth UserRepository, hasher PasswordHasher, session SessionCreator) *AuthService {
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

	user, err := domain.NewUser(input.Email)
	if err != nil {
		return dto.TokenPair{}, fmt.Errorf("sign up: %w", err)
	}

	if err := user.AddNativeIdentity(pwHash); err != nil {
		return dto.TokenPair{}, fmt.Errorf("sign up: %w", err)
	}

	if err = s.user.Create(ctx, user); err != nil {
		return dto.TokenPair{}, fmt.Errorf("sign up: %w", err)
	}

	refresh, err := s.session.Create(ctx, user)
	if err != nil {
		return dto.TokenPair{}, fmt.Errorf("sign up: %w", err)
	}

	access, err := s.access.Generate(user)
	if err != nil {
		return dto.TokenPair{}, fmt.Errorf("create session: %w", err)
	}

	return dto.TokenPair{
		Refresh: refresh,
		Access:  access,
	}, nil
}

func (s *AuthService) SignIn(ctx context.Context, input dto.CredentialsInput) (dto.TokenPair, error) {
	email, err := domain.NewEmail(input.Email)
	if err != nil {
		return dto.TokenPair{}, fmt.Errorf("sign in: %w", err)
	}

	user, err := s.user.FindByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		return dto.TokenPair{}, fmt.Errorf("sign in: %w", domain.ErrInvalidCredentials)
	} else if err != nil {
		return dto.TokenPair{}, fmt.Errorf("sign in: %w", err)
	}

	pwHash, err := user.GetPasswordHash()
	if err != nil {
		return dto.TokenPair{}, fmt.Errorf("sign in: %w", domain.ErrAlreadyExists)
	}

	if err := s.hasher.Compare(pwHash, input.Password); err != nil {
		return dto.TokenPair{}, fmt.Errorf("sign in: %w: %w", domain.ErrInvalidCredentials, err)
	}

	refresh, err := s.session.Create(ctx, user)
	if err != nil {
		return dto.TokenPair{}, fmt.Errorf("sign in: %w", err)
	}

	access, err := s.access.Generate(user)
	if err != nil {
		return dto.TokenPair{}, fmt.Errorf("create session: %w", err)
	}

	return dto.TokenPair{
		Refresh: refresh,
		Access:  access,
	}, nil
}

func (s *AuthService) OauthSignIn(providerName string) (dto.OAuthOutput, error) {
	provider, err := s.provider.Use(providerName)
	if err != nil {
		return dto.OAuthOutput{}, fmt.Errorf("oauth sign in: %w", err)
	}

	tokens := s.oAuth.Generate()
	url := provider.AuthCodeURL(tokens.State, tokens.Verifier)
	return dto.OAuthOutput{
		State:    tokens.State,
		Verifier: tokens.Verifier,
		TTL:      tokens.TTL,
		URL:      url,
	}, nil
}

func (s *AuthService) OauthCallback(ctx context.Context, input dto.OAuthCallbackInput) (dto.Token, error) {
	p, err := domain.ToProvider(input.ProviderName)

	provider, err := s.provider.Use(input.ProviderName)
	if err != nil {
		return dto.Token{}, fmt.Errorf("oauth callback: %w", err)
	}

	info, err := provider.Exchange(ctx, input.Code, input.Verifier)
	if err != nil {
		return dto.Token{}, fmt.Errorf("oauth callback: %w", err)
	}

	// User already with this identity exists
	user, err := s.user.FindByIdentity(ctx, p, info.ProviderUserID)
	if err == nil {
		pair, err := s.session.Create(ctx, user)
		if err != nil {
			return dto.Token{}, fmt.Errorf("oauth callback: %w", err)
		}

		return pair, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return dto.Token{}, fmt.Errorf("oauth callback: %w", err)
	}

	// User with this email exists
	email, err := domain.NewEmail(info.Email)
	if err != nil {
		return dto.Token{}, fmt.Errorf("oauth callback: %w", err)
	}

	if _, err = s.user.FindByEmail(ctx, email); err != nil {
		return dto.Token{}, fmt.Errorf("oauth callback: %w", domain.ErrAlreadyExists)
	} else if !errors.Is(err, domain.ErrNotFound) {
		return dto.Token{}, fmt.Errorf("oauth callback: %w", err)
	}

	// Create new user
	user, err = domain.NewUser(info.Email)
	if err != nil {
		return dto.Token{}, fmt.Errorf("oauth callback: %w", err)
	}

	if err := user.AddExternalIdentity(p, input.ProviderName); err != nil {
		return dto.Token{}, fmt.Errorf("oauth callback: %w", err)
	}

	if err = s.user.Create(ctx, user); err != nil {
		return dto.Token{}, fmt.Errorf("oauth callback: %w", err)
	}

	refresh, err := s.session.Create(ctx, user)
	if err != nil {
		return dto.Token{}, fmt.Errorf("oauth callback: %w", err)
	}

	return refresh, nil
}
