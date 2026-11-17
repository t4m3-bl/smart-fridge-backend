package domain

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	MaxNameLen               = 200
	MaxDescriptionLen        = 2000
	MaxQuantityLen           = 100
	MinShelfAfterOpeningDays = 1
	MaxShelfAfterOpeningDays = 3650
)

type ProductID uuid.UUID

type Product struct {
	ID       ProductID
	UserID   UserID
	Version  int32
	ImageKey *string
	ProductData
}

type ProductData struct {
	Name                  string
	CategoryID            CategoryID
	Description           *string
	Quantity              *string
	ManufacturedAt        *time.Time
	ExpiresAt             time.Time
	OpenedAt              *time.Time
	ShelfLifeAfterOpening *int32
}

func (d ProductData) Validate() error {
	if err := d.validateStrings(); err != nil {
		return err
	}
	if d.CategoryID <= 0 {
		return NewValidationErr("categoryId", "must be greater than 0")
	}
	return d.validateDates()
}

func (d ProductData) validateStrings() error {
	if n := utf8.RuneCountInString(d.Name); n < 1 || n > MaxNameLen {
		return NewValidationErr("name", fmt.Sprintf("must be 1..%d characters", MaxNameLen))
	}
	if d.Description != nil && utf8.RuneCountInString(*d.Description) > MaxDescriptionLen {
		return NewValidationErr("description", fmt.Sprintf("must be at most %d characters", MaxDescriptionLen))
	}
	if d.Quantity != nil && utf8.RuneCountInString(*d.Quantity) > MaxQuantityLen {
		return NewValidationErr("quantity", fmt.Sprintf("must be at most %d characters", MaxQuantityLen))
	}
	return nil
}

func (d ProductData) validateDates() error {
	if d.ExpiresAt.IsZero() {
		return NewValidationErr(
			"expiresAt",
			"expiration date is required",
		)
	}

	if d.ManufacturedAt != nil &&
		d.ManufacturedAt.After(d.ExpiresAt) {
		return NewValidationErr(
			"manufacturedAt",
			"manufactured at is after expiration",
		)
	}

	if d.OpenedAt != nil {
		if d.OpenedAt.After(d.ExpiresAt) {
			return NewValidationErr(
				"openedAt",
				"opened at is after expiration",
			)
		}

		if d.ManufacturedAt != nil &&
			d.OpenedAt.Before(*d.ManufacturedAt) {
			return NewValidationErr(
				"openedAt",
				"opened at is before manufactured at",
			)
		}
	}

	if s := d.ShelfLifeAfterOpening; s != nil &&
		(*s < MinShelfAfterOpeningDays ||
			*s > MaxShelfAfterOpeningDays) {
		return NewValidationErr(
			"shelfLifeAfterOpening",
			fmt.Sprintf(
				"must be between %d and %d days",
				MinShelfAfterOpeningDays,
				MaxShelfAfterOpeningDays,
			),
		)
	}

	return nil
}

func NewProduct(id ProductID, userID UserID, version int32, imageKey *string, data ProductData) (*Product, error) {
	if uuid.UUID(id) == uuid.Nil {
		return nil, NewValidationErr("id", "product id must not be nil")
	}
	if uuid.UUID(userID) == uuid.Nil {
		return nil, NewValidationErr("userId", "user id must not be nil")
	}
	if version < 0 {
		return nil, NewValidationErr("version", "version must not be negative")
	}

	if err := data.Validate(); err != nil {
		return nil, err
	}

	return &Product{
		ID:          id,
		UserID:      userID,
		Version:     version,
		ImageKey:    imageKey,
		ProductData: data,
	}, nil
}
