package postgres

import (
	"encoding/json"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"time"

	"github.com/google/uuid"
)

type productPayload struct {
	Name                  string     `json:"name"`
	CategoryID            *uuid.UUID `json:"categoryId,omitempty"`
	Description           *string    `json:"description,omitempty"`
	Quantity              *string    `json:"quantity,omitempty"`
	ImageLink             *string    `json:"imageLink,omitempty"`
	ManufacturedAt        *string    `json:"manufacturedAt,omitempty"`
	ExpiresAt             string     `json:"expiresAt"`
	OpenedAt              *string    `json:"openedAt,omitempty"`
	ShelfLifeAfterOpening *int32     `json:"shelfLifeAfterOpening,omitempty"`
}

func encodePayload(d *domain.ProductData) ([]byte, error) {
	p := productPayload{
		Name:                  d.Name,
		Description:           d.Description,
		Quantity:              d.Quantity,
		ImageLink:             d.ImageLink,
		ManufacturedAt:        formatDate(d.ManufacturedAt),
		ExpiresAt:             d.ExpiresAt.Format(time.DateOnly),
		OpenedAt:              formatDate(d.OpenedAt),
		ShelfLifeAfterOpening: d.ShelfLifeAfterOpening,
	}
	if d.CategoryID != nil {
		p.CategoryID = new(uuid.UUID(*d.CategoryID))
	}
	return json.Marshal(p)
}

func decodePayload(raw []byte) (*domain.ProductData, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var p productPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	expiresAt, err := time.Parse(time.DateOnly, p.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("decode payload: expiresAt: %w", err)
	}
	manufacturedAt, err := parseDate(p.ManufacturedAt)
	if err != nil {
		return nil, fmt.Errorf("decode payload: manufacturedAt: %w", err)
	}
	openedAt, err := parseDate(p.OpenedAt)
	if err != nil {
		return nil, fmt.Errorf("decode payload: openedAt: %w", err)
	}

	d := &domain.ProductData{
		Name:                  p.Name,
		Description:           p.Description,
		Quantity:              p.Quantity,
		ImageLink:             p.ImageLink,
		ManufacturedAt:        manufacturedAt,
		ExpiresAt:             expiresAt,
		OpenedAt:              openedAt,
		ShelfLifeAfterOpening: p.ShelfLifeAfterOpening,
	}
	if p.CategoryID != nil {
		d.CategoryID = new(domain.CategoryID(*p.CategoryID))
	}
	return d, nil
}

func formatDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	return new(t.Format(time.DateOnly))
}

func parseDate(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	t, err := time.Parse(time.DateOnly, *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
