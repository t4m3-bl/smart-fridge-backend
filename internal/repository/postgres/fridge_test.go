//go:build integration

package postgres_test

import (
	"context"
	"smart-fridge-backend/internal/domain"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	db "smart-fridge-backend/internal/repository/postgres"
)

func TestFridgeRepositories(t *testing.T) {
	ctx := context.Background()
	user, other := newUser(), newUser()
	users := db.NewUserRepository(testPool)
	require.NoError(t, users.Create(ctx, user))
	require.NoError(t, users.Create(ctx, other))
	uid, oid := uuid.UUID(user.ID), uuid.UUID(other.ID)
	categories := db.NewCategoryRepository(testPool)
	products := db.NewProductRepository(testPool)
	settings := db.NewSettingsRepository(testPool)
	category, err := categories.Save(ctx, uid, domain.Category{ID: uuid.New(), Name: "Dairy"}, nil)
	require.NoError(t, err)
	require.EqualValues(t, 1, category.Version)
	product := domain.Product{ID: uuid.New(), Name: "Milk", CategoryID: &category.ID, ExpiresAt: "2027-01-01"}
	product, err = products.Save(ctx, uid, product, nil)
	require.NoError(t, err)
	retry, err := products.Save(ctx, uid, product, nil)
	require.NoError(t, err)
	require.Equal(t, product.Version, retry.Version)
	rows, err := products.List(ctx, oid, 200, 0)
	require.NoError(t, err)
	require.Empty(t, rows)
	foreign := product
	foreign.ID = uuid.New()
	_, err = products.Save(ctx, oid, foreign, nil)
	require.Error(t, err)
	version := int64(1)
	product.Name = "Fresh milk"
	product, err = products.Save(ctx, uid, product, &version)
	require.NoError(t, err)
	require.EqualValues(t, 2, product.Version)
	_, err = products.Save(ctx, uid, product, &version)
	require.ErrorIs(t, err, domain.ErrVersionConflict)
	require.ErrorIs(t, products.Delete(ctx, uid, product.ID, &version), domain.ErrVersionConflict)
	require.NoError(t, categories.Delete(ctx, uid, category.ID, &version))
	rows, err = products.List(ctx, uid, 200, 0)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Nil(t, rows[0].CategoryID)
	require.EqualValues(t, 3, rows[0].Version)
	state, err := settings.Get(ctx, uid)
	require.NoError(t, err)
	require.EqualValues(t, 1, state.Version)
	enabled := false
	clock := "10:30:00"
	days := int32(3)
	indicators := []domain.Indicator{{Color: "#FF0000", MaxDays: &days}}
	state, err = settings.Patch(ctx, uid, domain.SettingsPatch{HasNotifications: &enabled, NotificationTime: &clock, Indicators: &indicators}, &version)
	require.NoError(t, err)
	require.EqualValues(t, 2, state.Version)
	state, err = settings.Get(ctx, uid)
	require.NoError(t, err)
	require.False(t, state.HasNotifications)
	require.Equal(t, clock, state.NotificationTime)
	require.Len(t, state.Indicators, 1)
	_, err = settings.Patch(ctx, uid, domain.SettingsPatch{}, &version)
	require.ErrorIs(t, err, domain.ErrVersionConflict)
	currentSettings := int64(2)
	badIndicators := []domain.Indicator{{Color: "invalid"}}
	_, err = settings.Patch(ctx, uid, domain.SettingsPatch{Indicators: &badIndicators}, &currentSettings)
	require.Error(t, err)
	state, err = settings.Get(ctx, uid)
	require.NoError(t, err)
	require.EqualValues(t, 2, state.Version)
	require.Equal(t, "#FF0000", state.Indicators[0].Color)

	current := int64(3)
	require.NoError(t, products.Delete(ctx, uid, product.ID, &current))
	require.NoError(t, products.Delete(ctx, uid, product.ID, &current))
}
