package service_test

import (
	"context"
	"errors"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/usecase/service"
	"testing"
	"time"

	mockservice "smart-fridge-backend/internal/usecase/service/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newMutation(typ domain.MutationType, data *domain.ProductData) domain.Mutation {
	return domain.Mutation{
		ID:        domain.MutationID(uuid.New()),
		Type:      typ,
		ProductID: domain.ProductID(uuid.New()),
		Data:      data,
	}
}

func validData() *domain.ProductData {
	return &domain.ProductData{Name: "Молоко", ExpiresAt: time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)}
}

func TestSyncService_Push(t *testing.T) {
	t.Parallel()

	userID := domain.UserID(uuid.New())
	create := newMutation(domain.MutationCreateProduct, validData())
	invalid := newMutation(domain.MutationUpdateProduct, &domain.ProductData{Name: " "})
	remove := newMutation(domain.MutationDeleteProduct, nil)
	repoErr := errors.New("db is down")

	tests := []struct {
		name      string
		mutations []domain.Mutation
		setup     func(repo *mockservice.MockSyncRepository)
		want      []domain.MutationResult
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name:      "applies valid and rejects invalid mutations in order",
			mutations: []domain.Mutation{create, invalid, remove},
			setup: func(repo *mockservice.MockSyncRepository) {
				gomock.InOrder(
					repo.EXPECT().Apply(gomock.Any(), userID, create).
						Return(domain.MutationResult{MutationID: create.ID, Status: domain.MutationApplied, Seq: 1}, nil),
					repo.EXPECT().Reject(gomock.Any(), userID, gomock.Any(), domain.RejectValidation, gomock.Any()).
						DoAndReturn(func(_ context.Context, _ domain.UserID, m domain.Mutation, reason domain.RejectReason, msg string) (domain.MutationResult, error) {
							assert.Equal(t, invalid.ID, m.ID)
							return domain.MutationResult{MutationID: m.ID, Status: domain.MutationRejected, Seq: 2, Reason: reason, Message: msg}, nil
						}),
					repo.EXPECT().Apply(gomock.Any(), userID, remove).
						Return(domain.MutationResult{MutationID: remove.ID, Status: domain.MutationRejected, Seq: 3, Reason: domain.RejectNotFound}, nil),
				)
			},
			want: []domain.MutationResult{
				{MutationID: create.ID, Status: domain.MutationApplied, Seq: 1},
				{
					MutationID: invalid.ID, Status: domain.MutationRejected, Seq: 2, Reason: domain.RejectValidation,
					Message: "name validation error: string is empty or longer than 100 characters",
				},
				{MutationID: remove.ID, Status: domain.MutationRejected, Seq: 3, Reason: domain.RejectNotFound},
			},
			wantErr: require.NoError,
		},
		{
			name:      "empty batch",
			mutations: nil,
			setup:     func(*mockservice.MockSyncRepository) {},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				_, ok := errors.AsType[domain.ValidationError](err)
				require.True(t, ok, "expected validation error, got %v", err)
			},
		},
		{
			name:      "batch above limit",
			mutations: make([]domain.Mutation, service.MaxPushMutations+1),
			setup:     func(*mockservice.MockSyncRepository) {},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				_, ok := errors.AsType[domain.ValidationError](err)
				require.True(t, ok, "expected validation error, got %v", err)
			},
		},
		{
			name:      "mutation without id rejects the whole batch",
			mutations: []domain.Mutation{create, {Type: domain.MutationDeleteProduct, ProductID: remove.ProductID}},
			setup:     func(*mockservice.MockSyncRepository) {},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.EqualError(t, err, "mutations[1].mutationId validation error: required")
			},
		},
		{
			name:      "repository failure stops the batch",
			mutations: []domain.Mutation{create, remove},
			setup: func(repo *mockservice.MockSyncRepository) {
				repo.EXPECT().Apply(gomock.Any(), userID, create).Return(domain.MutationResult{}, repoErr)
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, repoErr)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mockservice.NewMockSyncRepository(ctrl)
			tt.setup(repo)

			got, err := service.NewSyncService(repo).Push(context.Background(), userID, tt.mutations)

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSyncService_Pull(t *testing.T) {
	t.Parallel()

	userID := domain.UserID(uuid.New())
	applied := func(seqs ...int64) []domain.AppliedMutation {
		out := make([]domain.AppliedMutation, 0, len(seqs))
		for _, s := range seqs {
			out = append(out, domain.AppliedMutation{Seq: s, Type: domain.MutationDeleteProduct})
		}
		return out
	}

	tests := []struct {
		name    string
		since   int64
		limit   *int
		setup   func(repo *mockservice.MockSyncRepository)
		want    dto.PullPage
		wantErr bool
	}{
		{
			name:  "default limit and last page",
			since: 5,
			setup: func(repo *mockservice.MockSyncRepository) {
				repo.EXPECT().ListApplied(gomock.Any(), userID, int64(5), int32(service.DefaultPullLimit+1)).Return(applied(6, 9), nil)
			},
			want: dto.PullPage{Mutations: applied(6, 9), Cursor: 9},
		},
		{
			name:  "has more pages",
			since: 0,
			limit: new(2),
			setup: func(repo *mockservice.MockSyncRepository) {
				repo.EXPECT().ListApplied(gomock.Any(), userID, int64(0), int32(3)).Return(applied(1, 2, 3), nil)
			},
			want: dto.PullPage{Mutations: applied(1, 2), Cursor: 2, HasMore: true},
		},
		{
			name:  "nothing new keeps cursor",
			since: 42,
			setup: func(repo *mockservice.MockSyncRepository) {
				repo.EXPECT().ListApplied(gomock.Any(), userID, int64(42), gomock.Any()).Return(nil, nil)
			},
			want: dto.PullPage{Cursor: 42},
		},
		{
			name:    "negative cursor",
			since:   -1,
			setup:   func(*mockservice.MockSyncRepository) {},
			wantErr: true,
		},
		{
			name:    "limit above maximum",
			limit:   new(service.MaxPullLimit + 1),
			setup:   func(*mockservice.MockSyncRepository) {},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repo := mockservice.NewMockSyncRepository(ctrl)
			tt.setup(repo)

			got, err := service.NewSyncService(repo).Pull(context.Background(), userID, tt.since, tt.limit)

			if tt.wantErr {
				_, ok := errors.AsType[domain.ValidationError](err)
				require.True(t, ok, "expected validation error, got %v", err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
