package service_test

import (
	"context"
	"errors"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/usecase/service"
	servicemocks "smart-fridge-backend/internal/usecase/service/mocks"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAuthService_SignUp(t *testing.T) {
	hashErr := errors.New("hash failed")
	createErr := errors.New("create failed")
	sessionErr := errors.New("session failed")
	wantPair := dto.TokenPair{
		Access:  dto.Token{Body: "access", TTL: time.Hour},
		Refresh: dto.Token{Body: "refresh", TTL: 24 * time.Hour},
	}
	input := dto.CredentialsInput{Email: "example@test.com", Password: "secret"}

	tests := []struct {
		name              string
		input             dto.CredentialsInput
		setup             func(*servicemocks.MockUserRepository, *servicemocks.MockPasswordHasher, *servicemocks.MockSessionCreator)
		want              dto.TokenPair
		wantErr           error
		wantValidationErr bool
	}{
		{
			name:  "successful sign up",
			input: input,
			setup: func(repo *servicemocks.MockUserRepository, hasher *servicemocks.MockPasswordHasher, session *servicemocks.MockSessionCreator) {
				hasher.EXPECT().Hash("secret").Return("hashed", nil)
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, u *domain.User) error {
						assert.Equal(t, "example@test.com", string(u.Email))
						assert.Equal(t, "hashed", u.PasswordHash)
						return nil
					})
				session.EXPECT().Create(gomock.Any(), gomock.Any()).Return(wantPair, nil)
			},
			want: wantPair,
		},
		{
			name:  "hasher fails",
			input: input,
			setup: func(_ *servicemocks.MockUserRepository, hasher *servicemocks.MockPasswordHasher, _ *servicemocks.MockSessionCreator) {
				hasher.EXPECT().Hash("secret").Return("", hashErr)
			},
			wantErr: hashErr,
		},
		{
			name:  "invalid email fails NewUser",
			input: dto.CredentialsInput{Email: "not-an-email", Password: "secret"},
			setup: func(_ *servicemocks.MockUserRepository, hasher *servicemocks.MockPasswordHasher, _ *servicemocks.MockSessionCreator) {
				hasher.EXPECT().Hash("secret").Return("hashed", nil)
			},
			wantValidationErr: true,
		},
		{
			name:  "user repository create fails",
			input: input,
			setup: func(repo *servicemocks.MockUserRepository, hasher *servicemocks.MockPasswordHasher, _ *servicemocks.MockSessionCreator) {
				hasher.EXPECT().Hash("secret").Return("hashed", nil)
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(createErr)
			},
			wantErr: createErr,
		},
		{
			name:  "session creation fails",
			input: input,
			setup: func(repo *servicemocks.MockUserRepository, hasher *servicemocks.MockPasswordHasher, session *servicemocks.MockSessionCreator) {
				hasher.EXPECT().Hash("secret").Return("hashed", nil)
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
				session.EXPECT().Create(gomock.Any(), gomock.Any()).Return(dto.TokenPair{}, sessionErr)
			},
			wantErr: sessionErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := servicemocks.NewMockUserRepository(ctrl)
			hasher := servicemocks.NewMockPasswordHasher(ctrl)
			session := servicemocks.NewMockSessionCreator(ctrl)

			tt.setup(repo, hasher, session)

			svc := service.NewAuthService(repo, hasher, session)
			got, err := svc.SignUp(context.Background(), tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, dto.TokenPair{}, got)
				return
			}

			if tt.wantValidationErr {
				var ve domain.ValidationErr
				assert.True(t, errors.As(err, &ve))
				assert.Equal(t, dto.TokenPair{}, got)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAuthService_SignIn(t *testing.T) {
	repoErr := errors.New("db down")
	compareErr := errors.New("password mismatch")
	sessionErr := errors.New("session failed")
	wantPair := dto.TokenPair{
		Access:  dto.Token{Body: "access", TTL: time.Hour},
		Refresh: dto.Token{Body: "refresh", TTL: 24 * time.Hour},
	}
	existingUser := &domain.User{
		ID:           domain.UserID(uuid.New()),
		Email:        "example@test.com",
		PasswordHash: "stored_hash",
	}
	input := dto.CredentialsInput{Email: "example@test.com", Password: "secret"}

	tests := []struct {
		name    string
		input   dto.CredentialsInput
		setup   func(*servicemocks.MockUserRepository, *servicemocks.MockPasswordHasher, *servicemocks.MockSessionCreator)
		want    dto.TokenPair
		wantErr error
	}{
		{
			name:  "successful sign in",
			input: input,
			setup: func(repo *servicemocks.MockUserRepository, hasher *servicemocks.MockPasswordHasher, session *servicemocks.MockSessionCreator) {
				repo.EXPECT().FindByEmail(gomock.Any(), "example@test.com").Return(existingUser, nil)
				hasher.EXPECT().Compare("stored_hash", "secret").Return(nil)
				session.EXPECT().Create(gomock.Any(), existingUser).Return(wantPair, nil)
			},
			want: wantPair,
		},
		{
			name:  "invalid email maps to ErrInvalidCredentials",
			input: dto.CredentialsInput{Email: "not-an-email", Password: "secret"},
			setup: func(_ *servicemocks.MockUserRepository, _ *servicemocks.MockPasswordHasher, _ *servicemocks.MockSessionCreator) {
			},
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:  "user not found maps to ErrInvalidCredentials",
			input: input,
			setup: func(repo *servicemocks.MockUserRepository, _ *servicemocks.MockPasswordHasher, _ *servicemocks.MockSessionCreator) {
				repo.EXPECT().FindByEmail(gomock.Any(), "example@test.com").Return(nil, domain.ErrNotFound)
			},
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:  "unexpected repository error is propagated",
			input: input,
			setup: func(repo *servicemocks.MockUserRepository, _ *servicemocks.MockPasswordHasher, _ *servicemocks.MockSessionCreator) {
				repo.EXPECT().FindByEmail(gomock.Any(), "example@test.com").Return(nil, repoErr)
			},
			wantErr: repoErr,
		},
		{
			name:  "wrong password maps to ErrInvalidCredentials",
			input: dto.CredentialsInput{Email: "example@test.com", Password: "wrong"},
			setup: func(repo *servicemocks.MockUserRepository, hasher *servicemocks.MockPasswordHasher, _ *servicemocks.MockSessionCreator) {
				repo.EXPECT().FindByEmail(gomock.Any(), "example@test.com").Return(existingUser, nil)
				hasher.EXPECT().Compare("stored_hash", "wrong").Return(compareErr)
			},
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:  "session creation fails",
			input: input,
			setup: func(repo *servicemocks.MockUserRepository, hasher *servicemocks.MockPasswordHasher, session *servicemocks.MockSessionCreator) {
				repo.EXPECT().FindByEmail(gomock.Any(), "example@test.com").Return(existingUser, nil)
				hasher.EXPECT().Compare("stored_hash", "secret").Return(nil)
				session.EXPECT().Create(gomock.Any(), existingUser).Return(dto.TokenPair{}, sessionErr)
			},
			wantErr: sessionErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := servicemocks.NewMockUserRepository(ctrl)
			hasher := servicemocks.NewMockPasswordHasher(ctrl)
			session := servicemocks.NewMockSessionCreator(ctrl)

			tt.setup(repo, hasher, session)

			svc := service.NewAuthService(repo, hasher, session)
			got, err := svc.SignIn(context.Background(), tt.input)

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
