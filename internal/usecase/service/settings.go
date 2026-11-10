package service

import (
	"context"
	"fmt"
	"smart-fridge-backend/internal/domain"
)

type SettingsRepository interface {
	Get(ctx context.Context, userID domain.UserID) (*domain.Settings, error)
	Patch(ctx context.Context, p *domain.SettingsPatch) (*domain.Settings, error)
}

type SettingsService struct{ repo SettingsRepository }

func NewSettingsService(repo SettingsRepository) *SettingsService {
	return &SettingsService{repo: repo}
}

func (s *SettingsService) Get(ctx context.Context, userID domain.UserID) (*domain.Settings, error) {
	res, err := s.repo.Get(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}

	return res, nil
}

func (s *SettingsService) Patch(ctx context.Context, userID domain.UserID, base int32, c domain.SettingsChanges) (*domain.Settings, error) {
	p, err := domain.NewSettingsPatch(userID, base, c)
	if err != nil {
		return nil, err
	}

	return s.repo.Patch(ctx, p)
}
