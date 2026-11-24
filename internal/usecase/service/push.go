package service

import (
	"context"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/pkg/crypto/sha256"

	"github.com/google/uuid"
)

type PushSubscriptionRepository interface {
	Upsert(context.Context, domain.UserID, dto.PushSubscriptionInput) (uuid.UUID, error)
	Revoke(context.Context, domain.UserID, uuid.UUID) error
	RevokeBrowser(context.Context, string) error
}

type PushService struct {
	repo PushSubscriptionRepository
	cfg  config.Push
}

func NewPushService(repo PushSubscriptionRepository, cfg *config.Config) *PushService {
	return &PushService{repo: repo, cfg: cfg.Push}
}

func (s *PushService) PublicKey() (string, error) {
	if !s.cfg.Enabled {
		return "", domain.ErrPushNotConfigured
	}
	return s.cfg.PublicKey, nil
}

func (s *PushService) Upsert(ctx context.Context, userID domain.UserID, in dto.PushSubscriptionInput) (uuid.UUID, error) {
	if !s.cfg.Enabled {
		return uuid.Nil, domain.ErrPushNotConfigured
	}
	if err := domain.ValidatePushSubscription(in.Endpoint, in.P256DH, in.Auth); err != nil {
		return uuid.Nil, err
	}
	return s.repo.Upsert(ctx, userID, in)
}

func (s *PushService) Revoke(ctx context.Context, userID domain.UserID, id uuid.UUID) error {
	return s.repo.Revoke(ctx, userID, id)
}

func (s *PushService) RevokeBrowser(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.repo.RevokeBrowser(ctx, sha256.Hash(token))
}
