//go:build integration

package postgres_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	db "smart-fridge-backend/internal/repository/postgres"
	"smart-fridge-backend/internal/workerapp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSynchronousNotifications(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	user := seedUser(t, nativeIdentity())
	uid := uuid.UUID(user.ID)
	_, err := testPool.Exec(ctx, `UPDATE settings SET timezone='UTC',notification_time='00:00' WHERE user_id=$1`, uid)
	require.NoError(t, err)
	now := time.Now().UTC()
	today := now.Format(time.DateOnly)
	_, err = testPool.Exec(ctx, `INSERT INTO products(id,user_id,category_id,name,expires_at,opened_at,shelf_life_after_opening)
 VALUES ($1,$4,1,'Today',$5::date,NULL,NULL),($2,$4,1,'Expired',$5::date-1,NULL,NULL),($3,$4,1,'Opened',$5::date+30,$5::date,1)`, uuid.New(), uuid.New(), uuid.New(), uid, today)
	require.NoError(t, err)
	subscriptions := db.NewPushSubscriptionRepository(testPool)
	first, err := subscriptions.Upsert(ctx, user.ID, dto.PushSubscriptionInput{Endpoint: "https://example.com/" + uuid.NewString(), P256DH: "key", Auth: "auth"})
	require.NoError(t, err)
	_, err = subscriptions.Upsert(ctx, user.ID, dto.PushSubscriptionInput{Endpoint: "https://example.com/" + uuid.NewString(), P256DH: "key", Auth: "auth"})
	require.NoError(t, err)
	other := seedUser(t, nativeIdentity())
	var endpoint string
	require.NoError(t, testPool.QueryRow(ctx, `SELECT endpoint FROM push_subscriptions WHERE id=$1`, first).Scan(&endpoint))
	_, err = subscriptions.Upsert(ctx, other.ID, dto.PushSubscriptionInput{Endpoint: endpoint, P256DH: "other", Auth: "other"})
	require.ErrorIs(t, err, domain.ErrAlreadyExists)
	repo := db.NewNotificationRepository(testPool)
	schedules, err := repo.ListSchedules(ctx)
	require.NoError(t, err)
	schedule := domain.NotificationSchedule{UserID: user.ID, NotificationTime: 0, Timezone: "UTC"}
	require.Contains(t, schedules, schedule)
	summary, err := repo.Prepare(ctx, schedule, now)
	require.NoError(t, err)
	require.Len(t, summary.Subscriptions, 2)
	var payload map[string]string
	require.NoError(t, json.Unmarshal(summary.Payload, &payload))
	require.Contains(t, payload["body"], "Today")
	require.Contains(t, payload["body"], "Opened")
	require.NotContains(t, payload["body"], "Expired")
	require.Less(t, len(summary.Payload), 3000)
	require.NoError(t, repo.Revoke(ctx, user.ID, first))
	active, err := repo.CanSend(ctx, schedule, first)
	require.NoError(t, err)
	require.False(t, active)
	_, err = testPool.Exec(ctx, `UPDATE settings SET has_notifications=false WHERE user_id=$1`, uid)
	require.NoError(t, err)
	_, err = repo.Prepare(ctx, schedule, now)
	require.ErrorIs(t, err, domain.ErrNotFound)
	// No queued jobs and no last-run column exist in the final schema.
	var noQueue bool
	require.NoError(t, testPool.QueryRow(ctx, `SELECT to_regclass('notification_batches') IS NULL AND to_regclass('notification_deliveries') IS NULL`).Scan(&noQueue))
	require.True(t, noQueue)
	var markerCount int
	require.NoError(t, testPool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_name='settings' AND column_name='last_notification_date'`).Scan(&markerCount))
	require.Zero(t, markerCount)
	// Worker boots without HTTP, JWT, OAuth or S3 configuration.
	application := workerapp.New(&config.WorkerConfig{App: config.WorkerApp{ShutdownTimeout: time.Second}, Database: config.Database{URL: testPool.Config().ConnString()}, Push: config.Push{Enabled: true, PollInterval: time.Hour, RequestTimeout: time.Second}})
	require.NoError(t, application.Start(ctx))
	require.NoError(t, application.Stop(ctx))
}

func TestNotificationIndicatorBoundaries(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		bounds   []*int32
		included []int
	}{
		{name: "first two sorted thresholds", bounds: []*int32{nil, new(int32(5)), new(int32(2)), new(int32(1))}, included: []int{0, 1, 2}},
		{name: "third finite threshold skipped", bounds: []*int32{new(int32(9)), new(int32(1)), nil, new(int32(0))}, included: []int{0, 1}},
		{name: "unlimited only", bounds: []*int32{nil}},
		{name: "no indicators"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			user := seedUser(t, nativeIdentity())
			now := time.Now().UTC()
			_, err := testPool.Exec(ctx, `UPDATE settings SET timezone='UTC',notification_time='00:00' WHERE user_id=$1`, uuid.UUID(user.ID))
			require.NoError(t, err)
			_, err = testPool.Exec(ctx, `INSERT INTO products(id,user_id,category_id,name,expires_at)
   SELECT uuidv7(),$1,1,'day '||days,$2::date+days FROM generate_series(-1,5) AS days`, uuid.UUID(user.ID), now.Format(time.DateOnly))
			require.NoError(t, err)
			indicators := make([]domain.Indicator, 0, len(tc.bounds))
			for index, bound := range tc.bounds {
				indicators = append(indicators, domain.Indicator{Color: fmt.Sprintf("#%06X", index+1), MaxDays: bound})
			}
			patch, err := domain.NewSettingsPatch(user.ID, domain.SettingsChanges{Indicators: &indicators})
			require.NoError(t, err)
			_, err = db.NewSettingsRepository(testPool).Patch(ctx, patch)
			require.NoError(t, err)
			summary, err := db.NewNotificationRepository(testPool).Prepare(ctx, domain.NotificationSchedule{UserID: user.ID, Timezone: "UTC"}, now)
			if len(tc.included) == 0 {
				require.ErrorIs(t, err, domain.ErrNotFound)
				return
			}
			require.NoError(t, err)
			var payload map[string]string
			require.NoError(t, json.Unmarshal(summary.Payload, &payload))
			for days := -1; days <= 5; days++ {
				if slices.Contains(tc.included, days) {
					require.Contains(t, payload["body"], fmt.Sprintf("day %d", days))
				} else {
					require.NotContains(t, payload["body"], fmt.Sprintf("day %d", days))
				}
			}
		})
	}
}
