package service

import (
	"context"
	"github.com/google/uuid"
	"smart-fridge-backend/internal/domain"
	"strings"
)

type CategoryRepository interface {
	List(context.Context, uuid.UUID, int32, int32) ([]domain.Category, error)
	Save(context.Context, uuid.UUID, domain.Category, *int64) (domain.Category, error)
	Delete(context.Context, uuid.UUID, uuid.UUID, *int64) error
}
type CategoryService struct{ repo CategoryRepository }

func NewCategoryService(repo CategoryRepository) *CategoryService { return &CategoryService{repo} }
func (s *CategoryService) List(ctx context.Context, user uuid.UUID, limit, offset int32) ([]domain.Category, error) {
	if limit < 1 || limit > 500 || offset < 0 {
		return nil, domain.NewValidationErr("pagination", "invalid limit or offset")
	}
	return s.repo.List(ctx, user, limit, offset)
}
func (s *CategoryService) Save(ctx context.Context, user uuid.UUID, item domain.Category, base *int64) (domain.Category, error) {
	if item.ID == uuid.Nil || strings.TrimSpace(item.Name) == "" {
		return item, domain.NewValidationErr("id/name", "required")
	}
	if base != nil && *base < 1 {
		return item, domain.NewValidationErr("baseVersion", "must be positive")
	}
	if len(item.Name) > 100 {
		return item, domain.NewValidationErr("name", "too long")
	}
	return s.repo.Save(ctx, user, item, base)
}
func (s *CategoryService) Delete(ctx context.Context, user, id uuid.UUID, base *int64) error {
	if id == uuid.Nil || (base != nil && *base < 1) {
		return domain.NewValidationErr("id/baseVersion", "invalid value")
	}
	return s.repo.Delete(ctx, user, id, base)
}
