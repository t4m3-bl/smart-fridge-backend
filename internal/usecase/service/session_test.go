package service_test

import (
	"context"
	"errors"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/usecase/service"
	mockservice "smart-fridge-backend/internal/usecase/service/mocks"
	"smart-fridge-backend/pkg/crypto/sha256"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSessionService_Create(t *testing.T) {
	t.Parallel()

	user := &domain.User{
		ID:    domain.UserID(uuid.New()),
		Email: "example@test.com",
	}
	refreshToken := dto.Token{Body: "refresh-body", TTL: 24 * time.Hour}
	createErr := errors.New("create failed")

	tests := []struct {
		name    string
		setup   func(*mockservice.MockSessionRepository, *mockservice.MockRefreshTokenGenerator)
		want    dto.Token
		wantErr error
	}{
		{
			name: "successful session creation",
			setup: func(sessions *mockservice.MockSessionRepository, refresh *mockservice.MockRefreshTokenGenerator) {
				refresh.EXPECT().Generate().Return(refreshToken)
				sessions.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, s *domain.Session) error {
						assert.Equal(t, user.ID, s.UserID)
						assert.Equal(t, sha256.Hash(refreshToken.Body), s.RefreshTokenHash)
						assert.Equal(t, refreshToken.TTL, s.ExpiresIn)
						return nil
					})
			},
			want: refreshToken,
		},
		{
			name: "session repository create fails",
			setup: func(sessions *mockservice.MockSessionRepository, refresh *mockservice.MockRefreshTokenGenerator) {
				refresh.EXPECT().Generate().Return(refreshToken)
				sessions.EXPECT().Create(gomock.Any(), gomock.Any()).Return(createErr)
			},
			wantErr: createErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			sessions := mockservice.NewMockSessionRepository(ctrl)
			users := mockservice.NewMockUserFinder(ctrl)
			access := mockservice.NewMockAccessTokenGenerator(ctrl)
			refresh := mockservice.NewMockRefreshTokenGenerator(ctrl)

			tt.setup(sessions, refresh)

			svc := service.NewSessionService(sessions, users, access, refresh)
			got, err := svc.Create(context.Background(), user)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, dto.Token{}, got)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSessionService_Rotate(t *testing.T) {
	t.Parallel()

	const oldRefresh = "old-refresh-token"
	user := &domain.User{
		ID:    domain.UserID(uuid.New()),
		Email: "example@test.com",
	}
	newRefresh := dto.Token{Body: "new-refresh-body", TTL: 24 * time.Hour}
	newAccess := dto.Token{Body: "new-access-body", TTL: time.Hour}
	newSession := &domain.Session{
		ID:               domain.SessionID(uuid.New()),
		UserID:           user.ID,
		RefreshTokenHash: sha256.Hash(newRefresh.Body),
		ExpiresIn:        newRefresh.TTL,
	}
	rotateErr := errors.New("rotate failed")
	findErr := errors.New("find failed")
	accessErr := errors.New("access failed")

	tests := []struct {
		name    string
		setup   func(*mockservice.MockSessionRepository, *mockservice.MockUserFinder, *mockservice.MockAccessTokenGenerator, *mockservice.MockRefreshTokenGenerator)
		want    dto.TokenPair
		wantErr error
	}{
		{
			name: "successful rotation",
			setup: func(sessions *mockservice.MockSessionRepository, users *mockservice.MockUserFinder, access *mockservice.MockAccessTokenGenerator, refresh *mockservice.MockRefreshTokenGenerator) {
				refresh.EXPECT().Generate().Return(newRefresh)
				sessions.EXPECT().
					Rotate(gomock.Any(), sha256.Hash(oldRefresh), sha256.Hash(newRefresh.Body), newRefresh.TTL).
					Return(newSession, nil)
				users.EXPECT().FindByID(gomock.Any(), user.ID).Return(user, nil)
				access.EXPECT().Generate(user).Return(newAccess, nil)
			},
			want: dto.TokenPair{Access: newAccess, Refresh: newRefresh},
		},
		{
			name: "session rotate fails",
			setup: func(sessions *mockservice.MockSessionRepository, _ *mockservice.MockUserFinder, _ *mockservice.MockAccessTokenGenerator, refresh *mockservice.MockRefreshTokenGenerator) {
				refresh.EXPECT().Generate().Return(newRefresh)
				sessions.EXPECT().Rotate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, rotateErr)
			},
			wantErr: rotateErr,
		},
		{
			name: "user lookup fails",
			setup: func(sessions *mockservice.MockSessionRepository, users *mockservice.MockUserFinder, _ *mockservice.MockAccessTokenGenerator, refresh *mockservice.MockRefreshTokenGenerator) {
				refresh.EXPECT().Generate().Return(newRefresh)
				sessions.EXPECT().Rotate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(newSession, nil)
				users.EXPECT().FindByID(gomock.Any(), user.ID).Return(nil, findErr)
			},
			wantErr: findErr,
		},
		{
			name: "access generate fails",
			setup: func(sessions *mockservice.MockSessionRepository, users *mockservice.MockUserFinder, access *mockservice.MockAccessTokenGenerator, refresh *mockservice.MockRefreshTokenGenerator) {
				refresh.EXPECT().Generate().Return(newRefresh)
				sessions.EXPECT().Rotate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(newSession, nil)
				users.EXPECT().FindByID(gomock.Any(), user.ID).Return(user, nil)
				access.EXPECT().Generate(user).Return(dto.Token{}, accessErr)
			},
			wantErr: accessErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			sessions := mockservice.NewMockSessionRepository(ctrl)
			users := mockservice.NewMockUserFinder(ctrl)
			access := mockservice.NewMockAccessTokenGenerator(ctrl)
			refresh := mockservice.NewMockRefreshTokenGenerator(ctrl)

			tt.setup(sessions, users, access, refresh)

			svc := service.NewSessionService(sessions, users, access, refresh)
			got, err := svc.Rotate(context.Background(), oldRefresh)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, dto.TokenPair{}, got)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSessionService_Revoke(t *testing.T) {
	t.Parallel()

	revokeErr := errors.New("revoke failed")
	refresh := "refresh-to-revoke"

	tests := []struct {
		name    string
		refresh *string
		setup   func(*mockservice.MockSessionRepository)
		wantErr error
	}{
		{
			name:    "nil refresh is a no-op",
			refresh: nil,
			setup:   func(*mockservice.MockSessionRepository) {},
		},
		{
			name:    "successful revoke",
			refresh: &refresh,
			setup: func(sessions *mockservice.MockSessionRepository) {
				sessions.EXPECT().Revoke(gomock.Any(), sha256.Hash(refresh)).Return(nil)
			},
		},
		{
			name:    "repository revoke fails",
			refresh: &refresh,
			setup: func(sessions *mockservice.MockSessionRepository) {
				sessions.EXPECT().Revoke(gomock.Any(), sha256.Hash(refresh)).Return(revokeErr)
			},
			wantErr: revokeErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			sessions := mockservice.NewMockSessionRepository(ctrl)
			users := mockservice.NewMockUserFinder(ctrl)
			access := mockservice.NewMockAccessTokenGenerator(ctrl)
			refreshGen := mockservice.NewMockRefreshTokenGenerator(ctrl)

			tt.setup(sessions)

			svc := service.NewSessionService(sessions, users, access, refreshGen)
			err := svc.Revoke(context.Background(), tt.refresh)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			assert.NoError(t, err)
		})
	}
}
