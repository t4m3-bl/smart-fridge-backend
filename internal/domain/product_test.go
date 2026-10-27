package domain_test

import (
	"smart-fridge-backend/internal/domain"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func date(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func validProductData() domain.ProductData {
	return domain.ProductData{
		Name:      "Молоко",
		ExpiresAt: date("2026-11-10"),
	}
}

func TestProductData_Normalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		modify func(d *domain.ProductData)
		errStr string
	}{
		{
			name:   "minimal valid data",
			modify: func(*domain.ProductData) {},
		},
		{
			name: "all fields filled",
			modify: func(d *domain.ProductData) {
				d.Description = new("2.5%")
				d.Quantity = new("1 л")
				d.ImageLink = new("https://example.com/milk.jpg")
				d.ManufacturedAt = new(date("2026-11-01"))
				d.OpenedAt = new(date("2026-11-02"))
				d.ShelfLifeAfterOpening = new(int32(3))
			},
		},
		{
			name:   "name of spaces only",
			modify: func(d *domain.ProductData) { d.Name = "   " },
			errStr: "name validation error: string is empty or longer than 100 characters",
		},
		{
			name:   "missing expiration date",
			modify: func(d *domain.ProductData) { d.ExpiresAt = time.Time{} },
			errStr: "expiresAt validation error: required",
		},
		{
			name:   "name longer than limit",
			modify: func(d *domain.ProductData) { d.Name = strings.Repeat("я", domain.MaxProductNameLen+1) },
			errStr: "name validation error: string is empty or longer than 100 characters",
		},
		{
			name:   "name exactly at limit in runes",
			modify: func(d *domain.ProductData) { d.Name = strings.Repeat("я", domain.MaxProductNameLen) },
		},
		{
			name: "description longer than limit",
			modify: func(d *domain.ProductData) {
				d.Description = new(strings.Repeat("a", domain.MaxProductDescriptionLen+1))
			},
			errStr: "description validation error: string is longer than 500 characters",
		},
		{
			name:   "quantity longer than limit",
			modify: func(d *domain.ProductData) { d.Quantity = new(strings.Repeat("a", domain.MaxProductQuantityLen+1)) },
			errStr: "quantity validation error: string is longer than 50 characters",
		},
		{
			name:   "non-positive shelf life",
			modify: func(d *domain.ProductData) { d.ShelfLifeAfterOpening = new(int32(0)) },
			errStr: "shelfLifeAfterOpening validation error: must be from 1 to 3650 days",
		},
		{
			name:   "shelf life above maximum",
			modify: func(d *domain.ProductData) { d.ShelfLifeAfterOpening = new(int32(domain.MaxShelfLifeDays + 1)) },
			errStr: "shelfLifeAfterOpening validation error: must be from 1 to 3650 days",
		},
		{
			name:   "manufactured after expiration",
			modify: func(d *domain.ProductData) { d.ManufacturedAt = new(date("2026-11-11")) },
			errStr: "manufacturedAt validation error: must not be after expiresAt",
		},
		{
			name: "opened before manufacture",
			modify: func(d *domain.ProductData) {
				d.ManufacturedAt = new(date("2026-11-05"))
				d.OpenedAt = new(date("2026-11-04"))
			},
			errStr: "openedAt validation error: must not be before manufacturedAt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := validProductData()
			tt.modify(&d)

			err := d.Normalize()

			if tt.errStr == "" {
				require.NoError(t, err)
				return
			}
			assert.EqualError(t, err, tt.errStr)
		})
	}
}

func TestProductData_NormalizeTrimsName(t *testing.T) {
	t.Parallel()

	d := validProductData()
	d.Name = "  Сыр  "

	require.NoError(t, d.Normalize())
	assert.Equal(t, "Сыр", d.Name)
}
