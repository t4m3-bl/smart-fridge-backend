//go:build integration

package postgres_test

import (
	"smart-fridge-backend/internal/domain"
	db "smart-fridge-backend/internal/repository/postgres"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUserIndicatorDefaultsAndCustomization(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	first, second := seedUser(t, nativeIdentity()), seedUser(t, nativeIdentity())
	repo := db.NewSettingsRepository(testPool)
	firstSettings, err := repo.Get(ctx, first.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []domain.Indicator{
		{Color: "#EF4444", MaxDays: new(int32(1))},
		{Color: "#EAB308", MaxDays: new(int32(3))},
		{Color: "#22C55E"},
	}, firstSettings.Indicators)
	custom := []domain.Indicator{
		{Color: "#112233", MaxDays: new(int32(2))},
		{Color: "#445566", MaxDays: new(int32(5))},
		{Color: "#778899"},
	}
	patch, err := domain.NewSettingsPatch(first.ID, domain.SettingsChanges{Indicators: &custom})
	require.NoError(t, err)
	_, err = repo.Patch(ctx, patch)
	require.NoError(t, err)
	// Patching unrelated settings calls the default-settings path again and must preserve customization.
	patch, err = domain.NewSettingsPatch(first.ID, domain.SettingsChanges{NotificationTime: new(10 * time.Hour)})
	require.NoError(t, err)
	firstSettings, err = repo.Patch(ctx, patch)
	require.NoError(t, err)
	require.ElementsMatch(t, custom, firstSettings.Indicators)
	secondSettings, err := repo.Get(ctx, second.ID)
	require.NoError(t, err)
	require.Len(t, secondSettings.Indicators, 3)
	require.NotEqual(t, firstSettings.Indicators, secondSettings.Indicators)
}
