package postgres

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/repository/postgres/sqlc"
)

type SettingsRepository struct{ pool *pgxpool.Pool }

func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository { return &SettingsRepository{pool} }
func readSettings(ctx context.Context, q *sqlc.Queries, user uuid.UUID) (domain.Settings, error) {
	var item domain.Settings
	if err := q.EnsureSettings(ctx, user); err != nil {
		return item, err
	}
	raw, err := q.GetSettingsDocument(ctx, user)
	if err != nil {
		return item, err
	}
	data, ok := raw.([]byte)
	if !ok {
		data, err = json.Marshal(raw)
		if err != nil {
			return item, err
		}
	}
	err = json.Unmarshal(data, &item)
	return item, err
}
func (r *SettingsRepository) Get(ctx context.Context, user uuid.UUID) (domain.Settings, error) {
	var item domain.Settings
	err := ownerTransaction(ctx, r.pool, user, func(q *sqlc.Queries) error { var err error; item, err = readSettings(ctx, q, user); return err })
	return item, err
}
func (r *SettingsRepository) Patch(ctx context.Context, user uuid.UUID, patch domain.SettingsPatch, base *int64) (domain.Settings, error) {
	var item domain.Settings
	err := ownerTransaction(ctx, r.pool, user, func(q *sqlc.Queries) error {
		var err error
		item, err = readSettings(ctx, q, user)
		if err != nil {
			return err
		}
		if base == nil || *base != int64(item.Version) {
			return domain.ErrVersionConflict
		}
		if patch.HasNotifications != nil {
			item.HasNotifications = *patch.HasNotifications
		}
		if patch.NotificationTime != nil {
			item.NotificationTime = *patch.NotificationTime
		}
		if patch.Timezone != nil {
			item.Timezone = *patch.Timezone
		}
		if patch.Indicators != nil {
			item.Indicators = *patch.Indicators
		}
		item.Version++
		data, err := document(item, user)
		if err != nil {
			return err
		}
		if _, err = q.SaveSettingsDocument(ctx, data); err != nil {
			return err
		}
		if patch.Indicators != nil {
			if err = q.ClearIndicators(ctx, user); err != nil {
				return err
			}
			for _, i := range item.Indicators {
				if err = q.AddIndicator(ctx, sqlc.AddIndicatorParams{ID: uuid.New(), UserID: user, Color: i.Color, MaxDays: i.MaxDays}); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return item, err
}
