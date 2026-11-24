package service

import (
	"context"
	"errors"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/domain"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type notificationRepoStub struct {
	schedules  []domain.NotificationSchedule
	summary    domain.DailySummary
	prepareErr error
	revoked    []uuid.UUID
}

func (r *notificationRepoStub) ListSchedules(context.Context) ([]domain.NotificationSchedule, error) {
	return r.schedules, nil
}

func (r *notificationRepoStub) Prepare(context.Context, domain.NotificationSchedule, time.Time) (domain.DailySummary, error) {
	return r.summary, r.prepareErr
}

func (r *notificationRepoStub) CanSend(context.Context, domain.NotificationSchedule, uuid.UUID) (bool, error) {
	return true, nil
}

func (r *notificationRepoStub) Revoke(_ context.Context, _ domain.UserID, id uuid.UUID) error {
	r.revoked = append(r.revoked, id)
	return nil
}

type senderStub struct {
	calls  []uuid.UUID
	onSend func()
}

func (s *senderStub) Send(_ context.Context, message domain.PushMessage) (domain.PushResult, error) {
	s.calls = append(s.calls, message.Subscription.ID)
	if s.onSend != nil {
		s.onSend()
	}
	if len(s.calls) == 1 {
		return domain.PushResult{Status: 410}, errors.New("gone")
	}
	return domain.PushResult{Status: 201}, nil
}

func TestCronScheduleAndSequentialSending(t *testing.T) {
	t.Parallel()
	id := domain.UserID(uuid.New())
	now := time.Date(2026, time.November, 24, 18, 59, 59, 0, time.UTC)
	schedule := domain.NotificationSchedule{UserID: id, NotificationTime: 9 * time.Hour, Timezone: "Pacific/Kiritimati"}
	first, second := uuid.New(), uuid.New()
	repo := &notificationRepoStub{schedules: []domain.NotificationSchedule{schedule}, summary: domain.DailySummary{Payload: []byte(`{"title":"Test"}`), Subscriptions: []domain.PushSubscription{{ID: first}, {ID: second}}, ExpiresAt: time.Now().Add(time.Hour)}}
	sender := &senderStub{}
	worker := NewNotificationWorker(repo, sender, config.Push{RequestTimeout: time.Second})
	require.NoError(t, worker.refresh(t.Context(), now))
	require.Equal(t, now.Add(time.Second), worker.jobs[id].next)
	// Refreshing unchanged settings must not postpone the scheduled invocation.
	require.NoError(t, worker.refresh(t.Context(), now.Add(500*time.Millisecond)))
	require.Equal(t, now.Add(time.Second), worker.jobs[id].next)
	worker.runDue(t.Context(), now)
	require.Empty(t, sender.calls)
	worker.runDue(t.Context(), now.Add(time.Second))
	require.Equal(t, []uuid.UUID{first, second}, sender.calls)
	require.Equal(t, []uuid.UUID{first}, repo.revoked)
	worker.runDue(t.Context(), now.Add(2*time.Second))
	require.Len(t, sender.calls, 2)
	require.Equal(t, now.Add(time.Second).AddDate(0, 0, 1), worker.jobs[id].next)
	// Updating settings reschedules; disabling removes the cron job.
	repo.schedules[0].NotificationTime = 10 * time.Hour
	require.NoError(t, worker.refresh(t.Context(), now))
	require.Equal(t, now.Add(time.Hour+time.Second), worker.jobs[id].next)
	repo.schedules = nil
	require.NoError(t, worker.refresh(t.Context(), now))
	require.Empty(t, worker.jobs)
}

func TestCronDoesNotCatchUpMissedRuns(t *testing.T) {
	t.Parallel()
	id := domain.UserID(uuid.New())
	now := time.Date(2026, time.November, 24, 10, 0, 0, 0, time.UTC)
	repo := &notificationRepoStub{schedules: []domain.NotificationSchedule{{UserID: id, Timezone: "UTC", NotificationTime: 9 * time.Hour}}}
	worker := NewNotificationWorker(repo, &senderStub{}, config.Push{})
	require.NoError(t, worker.refresh(t.Context(), now))
	require.Equal(t, time.Date(2026, time.November, 25, 9, 0, 0, 0, time.UTC), worker.jobs[id].next)
}

func TestSynchronousSendStopsOnCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	repo := &notificationRepoStub{summary: domain.DailySummary{Subscriptions: []domain.PushSubscription{{ID: uuid.New()}, {ID: uuid.New()}}, ExpiresAt: time.Now().Add(time.Hour)}}
	sender := &senderStub{onSend: cancel}
	worker := NewNotificationWorker(repo, sender, config.Push{RequestTimeout: time.Second})
	require.ErrorIs(t, worker.sendSummary(ctx, domain.NotificationSchedule{}), context.Canceled)
	require.Len(t, sender.calls, 1)
}

func TestEmptySummaryNotSent(t *testing.T) {
	t.Parallel()
	repo := &notificationRepoStub{prepareErr: domain.ErrNotFound}
	sender := &senderStub{}
	worker := NewNotificationWorker(repo, sender, config.Push{})
	require.NoError(t, worker.sendSummary(t.Context(), domain.NotificationSchedule{}))
	require.Empty(t, sender.calls)
}
