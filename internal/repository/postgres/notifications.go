package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository struct{ pool *pgxpool.Pool }

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

func (r *NotificationRepository) ListSchedules(ctx context.Context) ([]domain.NotificationSchedule, error) {
	const query = `SELECT s.user_id,s.notification_time,s.timezone FROM settings s
 WHERE s.has_notifications AND EXISTS(SELECT 1 FROM push_subscriptions sub WHERE sub.user_id=s.user_id AND sub.revoked_at IS NULL)
 ORDER BY s.user_id`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list notification schedules: %w", err)
	}
	defer rows.Close()
	schedules := make([]domain.NotificationSchedule, 0)
	for rows.Next() {
		var schedule domain.NotificationSchedule
		var userID uuid.UUID
		var clock pgtype.Time
		if err = rows.Scan(&userID, &clock, &schedule.Timezone); err != nil {
			return nil, fmt.Errorf("scan notification schedule: %w", err)
		}
		schedule.UserID = domain.UserID(userID)
		schedule.NotificationTime = time.Duration(clock.Microseconds) * time.Microsecond
		schedules = append(schedules, schedule)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("read notification schedules: %w", err)
	}
	return schedules, nil
}

func (r *NotificationRepository) Prepare(ctx context.Context, schedule domain.NotificationSchedule, now time.Time) (domain.DailySummary, error) {
	const query = `WITH notification_indicators AS (
 SELECT color,max_days FROM indicators WHERE user_id=$1
 ORDER BY max_days NULLS LAST LIMIT 2
 )
 SELECT summary.names,summary.total,($3::timestamptz AT TIME ZONE s.timezone)::date,
 ((($3::timestamptz AT TIME ZONE s.timezone)::date+1)::timestamp AT TIME ZONE s.timezone) FROM settings s
 CROSS JOIN LATERAL (
 SELECT array_agg(name ORDER BY effective_expires_at,id) AS names,max(total) AS total FROM (
 SELECT p.id,p.name,p.effective_expires_at,count(*) OVER() AS total
 FROM products p
 JOIN LATERAL (
  SELECT i.color,i.max_days FROM notification_indicators i
  WHERE i.max_days IS NOT NULL
   AND p.effective_expires_at-($3::timestamptz AT TIME ZONE s.timezone)::date <= i.max_days
  ORDER BY i.max_days LIMIT 1
 ) active_indicator ON true
 WHERE p.user_id=s.user_id
 AND p.effective_expires_at >= ($3::timestamptz AT TIME ZONE s.timezone)::date
 ORDER BY p.effective_expires_at,p.id LIMIT 8
 ) items ) summary
 WHERE s.user_id=$1 AND s.has_notifications
 AND s.notification_time=$2::time AND s.timezone=$4
 AND ($3::timestamptz AT TIME ZONE s.timezone)::time>=s.notification_time AND summary.total>0`
	var summary domain.DailySummary
	var names []string
	var total int64
	var localDate time.Time
	err := r.pool.QueryRow(ctx, query, uuid.UUID(schedule.UserID), pgtype.Time{Microseconds: schedule.NotificationTime.Microseconds(), Valid: true}, now, schedule.Timezone).Scan(&names, &total, &localDate, &summary.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return summary, domain.ErrNotFound
	}
	if err != nil {
		return summary, fmt.Errorf("prepare notification summary: %w", err)
	}
	for i, name := range names {
		letters := []rune(name)
		if len(letters) > 40 {
			names[i] = string(letters[:40]) + "…"
		}
	}
	body := strings.Join(names, ", ")
	if total > int64(len(names)) {
		body += fmt.Sprintf(" и ещё %d", total-int64(len(names)))
	}
	summary.Payload, err = json.Marshal(struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		Tag   string `json:"tag"`
		URL   string `json:"url"`
	}{
		Title: fmt.Sprintf("Скоро истекает срок: %d", total), Body: body, Tag: "fridge-daily-" + uuid.UUID(schedule.UserID).String() + "-" + localDate.Format(time.DateOnly), URL: "/fridge",
	})
	if err != nil {
		return summary, fmt.Errorf("encode notification summary: %w", err)
	}
	rows, err := r.pool.Query(ctx, `SELECT id,endpoint,p256dh,auth FROM push_subscriptions WHERE user_id=$1 AND revoked_at IS NULL ORDER BY id`, uuid.UUID(schedule.UserID))
	if err != nil {
		return summary, fmt.Errorf("list push subscriptions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var subscription domain.PushSubscription
		if err = rows.Scan(&subscription.ID, &subscription.Endpoint, &subscription.P256DH, &subscription.Auth); err != nil {
			return summary, err
		}
		summary.Subscriptions = append(summary.Subscriptions, subscription)
	}
	if err = rows.Err(); err != nil {
		return summary, fmt.Errorf("read push subscriptions: %w", err)
	}
	return summary, nil
}

func (r *NotificationRepository) CanSend(ctx context.Context, schedule domain.NotificationSchedule, subscriptionID uuid.UUID) (bool, error) {
	var active bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM push_subscriptions sub JOIN settings s ON s.user_id=sub.user_id
 WHERE sub.id=$1 AND s.user_id=$2 AND sub.revoked_at IS NULL AND s.has_notifications
 AND s.notification_time=$3::time AND s.timezone=$4)`, subscriptionID, uuid.UUID(schedule.UserID), pgtype.Time{Microseconds: schedule.NotificationTime.Microseconds(), Valid: true}, schedule.Timezone).Scan(&active)
	if err != nil {
		return false, fmt.Errorf("check push subscription: %w", err)
	}
	return active, nil
}

func (r *NotificationRepository) Revoke(ctx context.Context, userID domain.UserID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE push_subscriptions SET revoked_at=now(),updated_at=now() WHERE id=$1 AND user_id=$2`, id, uuid.UUID(userID))
	if err != nil {
		return fmt.Errorf("revoke invalid push subscription: %w", err)
	}
	return nil
}
