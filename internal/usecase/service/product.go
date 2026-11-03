package service

import (
	"context"
	"github.com/google/uuid"
	"smart-fridge-backend/internal/domain"
	"strings"
	"time"
)

type ProductRepository interface {
	List(context.Context, uuid.UUID, int32, int32) ([]domain.Product, error)
	Save(context.Context, uuid.UUID, domain.Product, *int64) (domain.Product, error)
	Delete(context.Context, uuid.UUID, uuid.UUID, *int64) error
}
type ProductService struct{ repo ProductRepository }

func NewProductService(repo ProductRepository) *ProductService { return &ProductService{repo} }
func (s *ProductService) List(ctx context.Context, user uuid.UUID, limit, offset int32) ([]domain.Product, error) {
	if limit < 1 || limit > 500 || offset < 0 {
		return nil, domain.NewValidationErr("pagination", "invalid limit or offset")
	}
	return s.repo.List(ctx, user, limit, offset)
}
func (s *ProductService) Save(ctx context.Context, user uuid.UUID, item domain.Product, base *int64) (domain.Product, error) {
	if item.ID == uuid.Nil || strings.TrimSpace(item.Name) == "" {
		return item, domain.NewValidationErr("id/name", "required")
	}
	if base != nil && *base < 1 {
		return item, domain.NewValidationErr("baseVersion", "must be positive")
	}
	if len([]rune(item.Name)) > 200 || item.ExpiresAt == "" {
		return item, domain.NewValidationErr("product", "name and expiresAt are required")
	}
	expires, err := time.Parse("2006-01-02", item.ExpiresAt)
	if err != nil || expires.IsZero() {
		return item, domain.NewValidationErr("expiresAt", "invalid date")
	}
	if item.OpenedAt != nil {
		if _, err := time.Parse("2006-01-02", *item.OpenedAt); err != nil {
			return item, domain.NewValidationErr("openedAt", "invalid date")
		}
	}
	if item.CategoryID != nil && *item.CategoryID == uuid.Nil {
		return item, domain.NewValidationErr("categoryId", "invalid UUID")
	}
	for _, field := range []struct {
		name  string
		value *string
		max   int
	}{{"description", item.Description, 2000}, {"quantity", item.Quantity, 100}, {"imageLink", item.ImageLink, 2048}} {
		if field.value != nil && len([]rune(*field.value)) > field.max {
			return item, domain.NewValidationErr(field.name, "too long")
		}
	}

	if item.ManufacturedAt != nil {
		made, e := time.Parse("2006-01-02", *item.ManufacturedAt)
		if e != nil || made.After(expires) {
			return item, domain.NewValidationErr("manufacturedAt", "invalid date order")
		}
		if item.OpenedAt != nil && *item.OpenedAt < *item.ManufacturedAt {
			return item, domain.NewValidationErr("openedAt", "before manufacture")
		}
	}
	if item.ShelfLifeAfterOpening != nil && (*item.ShelfLifeAfterOpening < 1 || *item.ShelfLifeAfterOpening > 3650) {
		return item, domain.NewValidationErr("shelfLifeAfterOpening", "must be 1..3650")
	}

	return s.repo.Save(ctx, user, item, base)
}
func (s *ProductService) Delete(ctx context.Context, user, id uuid.UUID, base *int64) error {
	if id == uuid.Nil || (base != nil && *base < 1) {
		return domain.NewValidationErr("id/baseVersion", "invalid value")
	}
	return s.repo.Delete(ctx, user, id, base)
}
