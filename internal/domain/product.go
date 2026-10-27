package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	MaxProductNameLen        = 100
	MaxProductDescriptionLen = 500
	MaxProductQuantityLen    = 50
	MaxProductImageLinkLen   = 2048
	MaxShelfLifeDays         = 3650
)

type ProductID uuid.UUID

type CategoryID uuid.UUID

type ProductData struct {
	Name                  string
	CategoryID            *CategoryID
	Description           *string
	Quantity              *string
	ImageLink             *string
	ManufacturedAt        *time.Time
	ExpiresAt             time.Time
	OpenedAt              *time.Time
	ShelfLifeAfterOpening *int32
}

func (d *ProductData) Normalize() error {
	d.Name = strings.TrimSpace(d.Name)
	if d.Name == "" || utf8.RuneCountInString(d.Name) > MaxProductNameLen {
		return NewValidationErr("name", fmt.Sprintf("string is empty or longer than %d characters", MaxProductNameLen))
	}

	if d.ExpiresAt.IsZero() {
		return NewValidationErr("expiresAt", "required")
	}

	if err := checkLen("description", d.Description, MaxProductDescriptionLen); err != nil {
		return err
	}
	if err := checkLen("quantity", d.Quantity, MaxProductQuantityLen); err != nil {
		return err
	}
	if err := checkLen("imageLink", d.ImageLink, MaxProductImageLinkLen); err != nil {
		return err
	}

	if d.ShelfLifeAfterOpening != nil && (*d.ShelfLifeAfterOpening < 1 || *d.ShelfLifeAfterOpening > MaxShelfLifeDays) {
		return NewValidationErr("shelfLifeAfterOpening", fmt.Sprintf("must be from 1 to %d days", MaxShelfLifeDays))
	}

	if d.ManufacturedAt != nil {
		if d.ManufacturedAt.After(d.ExpiresAt) {
			return NewValidationErr("manufacturedAt", "must not be after expiresAt")
		}
		if d.OpenedAt != nil && d.OpenedAt.Before(*d.ManufacturedAt) {
			return NewValidationErr("openedAt", "must not be before manufacturedAt")
		}
	}

	return nil
}

func checkLen(field string, value *string, maxLen int) error {
	if value != nil && utf8.RuneCountInString(*value) > maxLen {
		return NewValidationErr(field, fmt.Sprintf("string is longer than %d characters", maxLen))
	}
	return nil
}
