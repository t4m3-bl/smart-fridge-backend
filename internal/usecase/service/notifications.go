package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/domain"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/fx"
)

type NotificationRepository interface {
	ListSchedules(context.Context) ([]domain.NotificationSchedule, error)
	Prepare(context.Context, domain.NotificationSchedule, time.Time) (domain.DailySummary, error)
	CanSend(context.Context, domain.NotificationSchedule, uuid.UUID) (bool, error)
	Revoke(context.Context, domain.UserID, uuid.UUID) error
}
type (
	PushSender interface {
		Send(context.Context, domain.PushMessage) (domain.PushResult, error)
	}
	scheduledNotification struct {
		settings domain.NotificationSchedule
		cron     cron.Schedule
		next     time.Time
	}
	NotificationWorker struct {
		repo   NotificationRepository
		sender PushSender
		cfg    config.Push
		jobs   map[domain.UserID]scheduledNotification
		cancel context.CancelFunc
		done   chan struct{}
	}
)

func NewNotificationWorker(repo NotificationRepository, sender PushSender, cfg config.Push) *NotificationWorker {
	return &NotificationWorker{repo: repo, sender: sender, cfg: cfg, jobs: make(map[domain.UserID]scheduledNotification)}
}

func RegisterNotificationLifecycle(lc fx.Lifecycle, w *NotificationWorker) {
	lc.Append(fx.Hook{OnStart: w.Start, OnStop: w.Stop})
}

//nolint:contextcheck // The scheduler must outlive the Fx startup context.
func (w *NotificationWorker) Start(startCtx context.Context) error {
	if !w.cfg.Enabled {
		slog.Info("push worker disabled")
		return nil
	}
	if err := w.refresh(startCtx, time.Now()); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.done = make(chan struct{})
	go w.run(ctx)
	return nil
}

func (w *NotificationWorker) run(ctx context.Context) {
	defer close(w.done)
	slog.Info("push cron worker started", "schedule_refresh_interval", w.cfg.PollInterval)
	refresh := time.NewTicker(w.cfg.PollInterval)
	defer refresh.Stop()
	for {
		timer := time.NewTimer(w.nextWait(time.Now()))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-refresh.C:
			if err := w.refresh(ctx, time.Now()); err != nil && ctx.Err() == nil {
				slog.Error("push schedule refresh failed", "error", err)
			}
		case <-timer.C:
			w.runDue(ctx, time.Now())
		}
		timer.Stop()
	}
}

func (w *NotificationWorker) Stop(ctx context.Context) error {
	if w.cancel == nil {
		return nil
	}
	w.cancel()
	select {
	case <-w.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *NotificationWorker) refresh(ctx context.Context, now time.Time) error {
	schedules, err := w.repo.ListSchedules(ctx)
	if err != nil {
		return err
	}
	active := make(map[domain.UserID]struct{}, len(schedules))
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	for _, settings := range schedules {
		active[settings.UserID] = struct{}{}
		if old, exists := w.jobs[settings.UserID]; exists && old.settings == settings {
			continue
		}
		seconds := int(settings.NotificationTime / time.Second)
		spec := fmt.Sprintf("CRON_TZ=%s %d %d %d * * *", settings.Timezone, seconds%60, (seconds/60)%60, seconds/3600)
		parsed, parseErr := parser.Parse(spec)
		if parseErr != nil {
			delete(w.jobs, settings.UserID)
			slog.Error("invalid push cron schedule", "user_id", uuid.UUID(settings.UserID), "error", parseErr)
			continue
		}
		w.jobs[settings.UserID] = scheduledNotification{settings: settings, cron: parsed, next: parsed.Next(now)}
		slog.Info("push schedule updated", "user_id", uuid.UUID(settings.UserID), "cron", spec)
	}
	for userID := range w.jobs {
		if _, exists := active[userID]; !exists {
			delete(w.jobs, userID)
		}
	}
	return nil
}

func (w *NotificationWorker) nextWait(now time.Time) time.Duration {
	wait := time.Hour
	for _, job := range w.jobs {
		wait = min(wait, max(0, job.next.Sub(now)))
	}
	return wait
}

func (w *NotificationWorker) runDue(ctx context.Context, now time.Time) {
	for userID, job := range w.jobs {
		if ctx.Err() != nil {
			return
		}
		if job.next.After(now) {
			continue
		}
		job.next = job.cron.Next(now)
		w.jobs[userID] = job
		if err := w.sendSummary(ctx, job.settings); err != nil && ctx.Err() == nil {
			slog.Error("push summary failed", "user_id", uuid.UUID(userID), "error", err)
		}
	}
}

func (w *NotificationWorker) sendSummary(ctx context.Context, schedule domain.NotificationSchedule) error {
	summary, err := w.repo.Prepare(ctx, schedule, time.Now())
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(summary.Subscriptions) == 0 {
		return nil
	}
	for _, subscription := range summary.Subscriptions {
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = w.sendSubscription(ctx, schedule, summary, subscription); err != nil {
			slog.Error("push subscription failed", "subscription_id", subscription.ID, "error", err)
		}
	}
	return nil
}

func (w *NotificationWorker) sendSubscription(ctx context.Context, schedule domain.NotificationSchedule, summary domain.DailySummary, subscription domain.PushSubscription) error {
	if !summary.ExpiresAt.After(time.Now()) {
		return nil
	}
	active, checkErr := w.repo.CanSend(ctx, schedule, subscription.ID)
	if checkErr != nil {
		slog.Error("push subscription check failed", "subscription_id", subscription.ID, "error", checkErr)
		return checkErr
	}
	if !active {
		return nil
	}
	sendCtx, cancel := context.WithTimeout(ctx, w.cfg.RequestTimeout)
	result, sendErr := w.sender.Send(sendCtx, domain.PushMessage{Subscription: subscription, Payload: summary.Payload, TTL: min(time.Hour, time.Until(summary.ExpiresAt))})
	cancel()
	if sendErr != nil || result.Status < 200 || result.Status >= 300 {
		slog.Error("push send failed", "user_id", uuid.UUID(schedule.UserID), "subscription_id", subscription.ID, "provider_status", result.Status, "error", sendErr)
	} else {
		slog.Info("push accepted", "user_id", uuid.UUID(schedule.UserID), "subscription_id", subscription.ID, "provider_status", result.Status)
	}
	if result.Status == 404 || result.Status == 410 {
		if err := w.repo.Revoke(ctx, schedule.UserID, subscription.ID); err != nil {
			slog.Error("push subscription revoke failed", "subscription_id", subscription.ID, "error", err)
		}
	}
	return nil
}
