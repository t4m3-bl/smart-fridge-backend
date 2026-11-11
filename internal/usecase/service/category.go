package service

import (
	"context"
	"errors"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
)

type CategoryRepository interface {
	List(ctx context.Context, userID domain.UserID) ([]domain.Category, error)
	Create(ctx context.Context, c *domain.Category) error
	Update(ctx context.Context, c *domain.Category) error
	Delete(ctx context.Context, in dto.CategoryDeleteInput) error
}

type CategoryService struct{ repo CategoryRepository }

func NewCategoryService(repo CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) List(ctx context.Context, userID domain.UserID) ([]domain.Category, error) {
	items, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return items, nil
}

func (s *CategoryService) Save(ctx context.Context, in dto.CategorySaveInput) (*domain.Category, error) {
	c, err := domain.NewCategory(domain.CategoryID(in.CategoryID), domain.UserID(in.UserID), in.BaseVersion, in.Name)
	if err != nil {
		return nil, err
	}

	if c.Version == 0 {
		err = s.repo.Create(ctx, c)
	} else {
		err = s.repo.Update(ctx, c)
	}
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			err = domain.ErrVersionConflict
		}
		return nil, fmt.Errorf("save category: %w", err)
	}

	c.UpdateVersion()
	return c, nil
}

func (s *CategoryService) Delete(ctx context.Context, in dto.CategoryDeleteInput) error {
	if in.BaseVersion < 1 {
		return domain.NewValidationErr("baseVersion", "must be >= 1")
	}
	if err := s.repo.Delete(ctx, in); err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	return nil
}
