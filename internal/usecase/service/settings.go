package service

import (
	"context"
	"github.com/google/uuid"
	"regexp"
	"smart-fridge-backend/internal/domain"
	"strings"
	"time"
)

type SettingsRepository interface {
	Get(context.Context, uuid.UUID) (domain.Settings, error)
	Patch(context.Context, uuid.UUID, domain.SettingsPatch, *int64) (domain.Settings, error)
}
type SettingsService struct{ repo SettingsRepository }

func NewSettingsService(repo SettingsRepository) *SettingsService { return &SettingsService{repo} }
func (s *SettingsService) Get(ctx context.Context, user uuid.UUID) (domain.Settings, error) {
	return s.repo.Get(ctx, user)
}

var indicatorColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (s *SettingsService) Patch(ctx context.Context, user uuid.UUID, p domain.SettingsPatch, base *int64) (domain.Settings, error) {
	invalid := func(field string) (domain.Settings, error) {
		return domain.Settings{}, domain.NewValidationErr(field, "invalid value")
	}
	if base == nil || *base < 1 {
		return invalid("baseVersion")
	}
	if p.NotificationTime != nil {
		if _, err := time.Parse("15:04:05", *p.NotificationTime); err != nil {
			return invalid("notificationTime")
		}
	}
	if p.Timezone != nil {
		if *p.Timezone == "" {
			return invalid("timezone")
		}
		if _, err := time.LoadLocation(*p.Timezone); err != nil {
			return invalid("timezone")
		}
	}
	if p.Indicators != nil {
		colors := map[string]bool{}
		days := map[int32]bool{}
		hasNull := false
		for _, i := range *p.Indicators {
			color := strings.ToLower(i.Color)
			if !indicatorColor.MatchString(color) || colors[color] {
				return invalid("indicators")
			}
			colors[color] = true
			if i.MaxDays == nil {
				if hasNull {
					return invalid("indicators")
				}
				hasNull = true
			} else {
				if *i.MaxDays < 0 || days[*i.MaxDays] {
					return invalid("indicators")
				}
				days[*i.MaxDays] = true
			}
		}
	}
	return s.repo.Patch(ctx, user, p, base)
}
