package postgres

import (
	"context"
	"errors"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/repository/postgres/sqlc"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SettingsRepository struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{pool: pool, q: sqlc.New(pool)}
}

// Get создаёт настройки по умолчанию при первом обращении.
func (r *SettingsRepository) Get(ctx context.Context, userID domain.UserID) (*domain.Settings, error) {
	s, err := r.read(ctx, userID)
	if errors.Is(err, domain.ErrNotFound) {
		if err = r.q.CreateDefaultSettings(ctx, uuid.UUID(userID)); err != nil {
			return nil, fmt.Errorf("create default settings: %w", err)
		}
		return r.read(ctx, userID)
	}
	return s, err
}

func (r *SettingsRepository) read(ctx context.Context, userID domain.UserID) (*domain.Settings, error) {
	var res *domain.Settings

	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadOnly,
	}, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)

		row, err := q.GetSettings(ctx, uuid.UUID(userID))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrNotFound
			}
			return err
		}
		inds, err := q.ListIndicators(ctx, uuid.UUID(userID))
		if err != nil {
			return err
		}

		res = toSettings(row, inds)
		return nil
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("get settings: %w", err)
	}
	return res, nil
}

func (r *SettingsRepository) Patch(ctx context.Context, p *domain.SettingsPatch) (*domain.Settings, error) {
	var res *domain.Settings

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		uid := uuid.UUID(p.UserID)

		row, err := q.PatchSettings(ctx, sqlc.PatchSettingsParams{
			UserID:           uid,
			HasNotifications: p.HasNotifications,
			NotificationTime: toPgTime(p.NotificationTime),
			Timezone:         p.Timezone,
			BaseVersion:      p.BaseVersion,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrVersionConflict // строки нет или версия не совпала
			}
			return err
		}

		if p.Indicators != nil {
			if err = q.DeleteIndicators(ctx, uid); err != nil {
				return err
			}
			for _, ind := range *p.Indicators {
				if err = q.CreateIndicator(ctx, sqlc.CreateIndicatorParams{
					ID:      uuid.New(),
					UserID:  uid,
					Color:   ind.Color,
					MaxDays: ind.MaxDays,
				}); err != nil {
					return err
				}
			}
		}

		inds, err := q.ListIndicators(ctx, uid) // внутри транзакции видны свои изменения
		if err != nil {
			return err
		}

		res = toSettings(row, inds)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("patch settings: %w", err)
	}
	return res, nil
}

func toPgTime(d *time.Duration) pgtype.Time {
	if d == nil {
		return pgtype.Time{} // Valid=false: NULL, COALESCE оставит прежнее значение
	}
	return pgtype.Time{Microseconds: d.Microseconds(), Valid: true}
}

func toSettings(row sqlc.Setting, inds []sqlc.Indicator) *domain.Settings {
	s := &domain.Settings{
		UserID:           domain.UserID(row.UserID),
		HasNotifications: row.HasNotifications,
		NotificationTime: time.Duration(row.NotificationTime.Microseconds) * time.Microsecond,
		Timezone:         row.Timezone,
		Indicators:       make([]domain.Indicator, 0, len(inds)),
		Version:          row.Version,
	}
	for _, i := range inds {
		s.Indicators = append(s.Indicators, domain.Indicator{Color: i.Color, MaxDays: i.MaxDays})
	}
	return s
}
