package service

import (
	"context"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
)

//go:generate go run go.uber.org/mock/mockgen -source=sync.go -destination=mocks/sync_mock.go

const (
	MaxPushMutations = 100
	DefaultPullLimit = 100
	MaxPullLimit     = 500
)

type SyncRepository interface {
	Apply(ctx context.Context, userID domain.UserID, m domain.Mutation) (domain.MutationResult, error)
	Reject(ctx context.Context, userID domain.UserID, m domain.Mutation, reason domain.RejectReason, message string) (domain.MutationResult, error)
	ListApplied(ctx context.Context, userID domain.UserID, since int64, limit int32) ([]domain.AppliedMutation, error)
}

type SyncService struct {
	repo SyncRepository
}

func NewSyncService(repo SyncRepository) *SyncService {
	return &SyncService{repo: repo}
}

func (s *SyncService) Push(ctx context.Context, userID domain.UserID, mutations []domain.Mutation) ([]domain.MutationResult, error) {
	if len(mutations) == 0 || len(mutations) > MaxPushMutations {
		return nil, domain.NewValidationErr("mutations", fmt.Sprintf("must contain from 1 to %d items", MaxPushMutations))
	}

	for i, m := range mutations {
		if m.ID.IsZero() {
			return nil, domain.NewValidationErr(fmt.Sprintf("mutations[%d].mutationId", i), "required")
		}
	}

	results := make([]domain.MutationResult, 0, len(mutations))
	for _, m := range mutations {
		result, err := s.push(ctx, userID, m)
		if err != nil {
			return nil, fmt.Errorf("push: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

func (s *SyncService) push(ctx context.Context, userID domain.UserID, m domain.Mutation) (domain.MutationResult, error) {
	if err := m.Validate(); err != nil {
		reason, ok := domain.RejectReasonOf(err)
		if !ok {
			return domain.MutationResult{}, err
		}
		return s.repo.Reject(ctx, userID, m, reason, err.Error())
	}

	return s.repo.Apply(ctx, userID, m)
}

func (s *SyncService) Pull(ctx context.Context, userID domain.UserID, since int64, limit *int) (dto.PullPage, error) {
	if since < 0 {
		return dto.PullPage{}, domain.NewValidationErr("since", "must not be negative")
	}

	n := DefaultPullLimit
	if limit != nil {
		n = *limit
	}
	if n < 1 || n > MaxPullLimit {
		return dto.PullPage{}, domain.NewValidationErr("limit", fmt.Sprintf("must be from 1 to %d", MaxPullLimit))
	}

	mutations, err := s.repo.ListApplied(ctx, userID, since, int32(n+1))
	if err != nil {
		return dto.PullPage{}, fmt.Errorf("pull: %w", err)
	}

	page := dto.PullPage{Cursor: since}
	if len(mutations) > n {
		mutations = mutations[:n]
		page.HasMore = true
	}
	if len(mutations) > 0 {
		page.Cursor = mutations[len(mutations)-1].Seq
	}
	page.Mutations = mutations

	return page, nil
}
