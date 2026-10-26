package service_test

import (
	"context"
	"errors"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	oauthmocks "smart-fridge-backend/internal/usecase/auth/oauth/mocks"
	"smart-fridge-backend/internal/usecase/service"
	servicemocks "smart-fridge-backend/internal/usecase/service/mocks"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type authMocks struct {
	repo      *servicemocks.MockUserRepository
	hasher    *servicemocks.MockPasswordHasher
	session   *servicemocks.MockSessionCreator
	access    *servicemocks.MockAccessTokenGenerator
	providers *servicemocks.MockOAuthProviders
	oauth     *servicemocks.MockOauthTokensGenerator
	provider  *oauthmocks.MockProvider
}

func newAuthService(t *testing.T) (*service.AuthService, authMocks) {
	ctrl := gomock.NewController(t)
	m := authMocks{
		repo:    servicemocks.NewMockUserRepository(ctrl),
		hasher:  servicemocks.NewMockPasswordHasher(ctrl),
		session: servicemocks.NewMockSessionCreator(ctrl),
		access:  servicemocks.NewMockAccessTokenGenerator(ctrl),

		providers: servicemocks.NewMockOAuthProviders(ctrl),
		oauth:     servicemocks.NewMockOauthTokensGenerator(ctrl),
		provider:  oauthmocks.NewMockProvider(ctrl),
	}

	return service.NewAuthService(m.repo, m.hasher, m.session, m.access, m.providers, m.oauth), m
}

func TestAuthService_SignUp(t *testing.T) {
	t.Parallel()

	hashErr := errors.New("hash failed")
	createErr := errors.New("create failed")
	sessionErr := errors.New("session failed")
	accessErr := errors.New("access failed")
	refresh := dto.Token{Body: "refresh", TTL: 24 * time.Hour}
	access := dto.Token{Body: "access", TTL: time.Hour}
	input := dto.CredentialsInput{Email: "example@test.com", Password: "secret"}

	tests := []struct {
		name              string
		input             dto.CredentialsInput
		setup             func(authMocks)
		want              dto.TokenPair
		wantErr           error
		wantValidationErr bool
	}{
		{
			name:  "successful sign up",
			input: input,
			setup: func(m authMocks) {
				m.hasher.EXPECT().Hash("secret").Return("hashed", nil)
				m.repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, u *domain.User) error {
						assert.Equal(t, domain.Email("example@test.com"), u.Email)
						hash, err := u.GetPasswordHash()
						assert.NoError(t, err)
						assert.Equal(t, "hashed", hash)
						return nil
					})
				m.session.EXPECT().Create(gomock.Any(), gomock.Any()).Return(refresh, nil)
				m.access.EXPECT().Generate(gomock.Any()).Return(access, nil)
			},
			want: dto.TokenPair{Refresh: refresh, Access: access},
		},
		{
			name:  "hasher fails",
			input: input,
			setup: func(m authMocks) {
				m.hasher.EXPECT().Hash("secret").Return("", hashErr)
			},
			wantErr: hashErr,
		},
		{
			name:  "invalid email fails NewUser",
			input: dto.CredentialsInput{Email: "not-an-email", Password: "secret"},
			setup: func(m authMocks) {
				m.hasher.EXPECT().Hash("secret").Return("hashed", nil)
			},
			wantValidationErr: true,
		},
		{
			name:  "user repository create fails",
			input: input,
			setup: func(m authMocks) {
				m.hasher.EXPECT().Hash("secret").Return("hashed", nil)
				m.repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(createErr)
			},
			wantErr: createErr,
		},
		{
			name:  "session creation fails",
			input: input,
			setup: func(m authMocks) {
				m.hasher.EXPECT().Hash("secret").Return("hashed", nil)
				m.repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
				m.session.EXPECT().Create(gomock.Any(), gomock.Any()).Return(dto.Token{}, sessionErr)
			},
			wantErr: sessionErr,
		},
		{
			name:  "access token generation fails",
			input: input,
			setup: func(m authMocks) {
				m.hasher.EXPECT().Hash("secret").Return("hashed", nil)
				m.repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
				m.session.EXPECT().Create(gomock.Any(), gomock.Any()).Return(refresh, nil)
				m.access.EXPECT().Generate(gomock.Any()).Return(dto.Token{}, accessErr)
			},
			wantErr: accessErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, m := newAuthService(t)
			tt.setup(m)

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
	t.Parallel()

	repoErr := errors.New("db down")
	compareErr := errors.New("password mismatch")
	sessionErr := errors.New("session failed")
	refresh := dto.Token{Body: "refresh", TTL: 24 * time.Hour}
	access := dto.Token{Body: "access", TTL: time.Hour}

	existingUser := &domain.User{
		ID:    domain.UserID(uuid.New()),
		Email: "example@test.com",
	}
	assert.NoError(t, existingUser.AddNativeIdentity("stored_hash"))

	yandexOnlyUser := &domain.User{
		ID:    domain.UserID(uuid.New()),
		Email: "example@test.com",
	}
	assert.NoError(t, yandexOnlyUser.AddExternalIdentity(domain.ProviderYandex, "yandex-uid"))

	input := dto.CredentialsInput{Email: "example@test.com", Password: "secret"}
	email := domain.Email("example@test.com")

	tests := []struct {
		name    string
		input   dto.CredentialsInput
		setup   func(authMocks)
		want    dto.TokenPair
		wantErr error
	}{
		{
			name:  "successful sign in",
			input: input,
			setup: func(m authMocks) {
				m.repo.EXPECT().FindByEmail(gomock.Any(), email).Return(existingUser, nil)
				m.hasher.EXPECT().Compare("stored_hash", "secret").Return(nil)
				m.session.EXPECT().Create(gomock.Any(), existingUser).Return(refresh, nil)
				m.access.EXPECT().Generate(existingUser).Return(access, nil)
			},
			want: dto.TokenPair{Refresh: refresh, Access: access},
		},
		{
			name:    "invalid email fails validation",
			input:   dto.CredentialsInput{Email: "not-an-email", Password: "secret"},
			setup:   func(authMocks) {},
			wantErr: domain.ValidationErr{},
		},
		{
			name:  "user not found maps to ErrInvalidCredentials",
			input: input,
			setup: func(m authMocks) {
				m.repo.EXPECT().FindByEmail(gomock.Any(), email).Return(nil, domain.ErrNotFound)
			},
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:  "unexpected repository error is propagated",
			input: input,
			setup: func(m authMocks) {
				m.repo.EXPECT().FindByEmail(gomock.Any(), email).Return(nil, repoErr)
			},
			wantErr: repoErr,
		},
		{
			name:  "user without password maps to ErrInvalidCredentials",
			input: input,
			setup: func(m authMocks) {
				m.repo.EXPECT().FindByEmail(gomock.Any(), email).Return(yandexOnlyUser, nil)
			},
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:  "wrong password maps to ErrInvalidCredentials",
			input: dto.CredentialsInput{Email: "example@test.com", Password: "wrong"},
			setup: func(m authMocks) {
				m.repo.EXPECT().FindByEmail(gomock.Any(), email).Return(existingUser, nil)
				m.hasher.EXPECT().Compare("stored_hash", "wrong").Return(compareErr)
			},
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:  "session creation fails",
			input: input,
			setup: func(m authMocks) {
				m.repo.EXPECT().FindByEmail(gomock.Any(), email).Return(existingUser, nil)
				m.hasher.EXPECT().Compare("stored_hash", "secret").Return(nil)
				m.session.EXPECT().Create(gomock.Any(), existingUser).Return(dto.Token{}, sessionErr)
			},
			wantErr: sessionErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, m := newAuthService(t)
			tt.setup(m)

			got, err := svc.SignIn(context.Background(), tt.input)

			if tt.wantErr != nil {
				if _, ok := tt.wantErr.(domain.ValidationErr); ok {
					_, isValidation := errors.AsType[domain.ValidationErr](err)
					assert.True(t, isValidation)
				} else {
					assert.ErrorIs(t, err, tt.wantErr)
				}
				assert.Equal(t, dto.TokenPair{}, got)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAuthService_OauthCallback(t *testing.T) {
	t.Parallel()

	const yandexUID = "yandex-uid-42"
	refresh := dto.Token{Body: "refresh", TTL: 24 * time.Hour}
	input := dto.OAuthCallbackInput{ProviderName: "yandex", Code: "code", Verifier: "verifier"}
	identity := dto.OAuthIdentity{ProviderUserID: yandexUID, Email: "Alex@Yandex.ru"}
	email := domain.Email("alex@yandex.ru")
	repoErr := errors.New("db down")

	existing := &domain.User{ID: domain.UserID(uuid.New()), Email: email}

	tests := []struct {
		name              string
		input             dto.OAuthCallbackInput
		setup             func(authMocks)
		want              dto.Token
		wantErr           error
		wantValidationErr bool
	}{
		{
			name:  "returning user signs in by identity",
			input: input,
			setup: func(m authMocks) {
				m.providers.EXPECT().Use(input.ProviderName).Return(m.provider, nil)
				m.provider.EXPECT().Exchange(gomock.Any(), input.Code, input.Verifier).Return(identity, nil)
				m.repo.EXPECT().FindByIdentity(gomock.Any(), domain.ProviderYandex, yandexUID).Return(existing, nil)
				m.session.EXPECT().Create(gomock.Any(), existing).Return(refresh, nil)
			},
			want: refresh,
		},
		{
			name:  "new user is created with provider user id",
			input: input,
			setup: func(m authMocks) {
				m.providers.EXPECT().Use(input.ProviderName).Return(m.provider, nil)
				m.provider.EXPECT().Exchange(gomock.Any(), input.Code, input.Verifier).Return(identity, nil)
				m.repo.EXPECT().FindByIdentity(gomock.Any(), domain.ProviderYandex, yandexUID).Return(nil, domain.ErrNotFound)
				m.repo.EXPECT().FindByEmail(gomock.Any(), email).Return(nil, domain.ErrNotFound)
				m.repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, u *domain.User) error {
						assert.Equal(t, email, u.Email)
						assert.Len(t, u.Identities, 1)
						assert.Equal(t, domain.ProviderYandex, u.Identities[0].Provider)
						assert.Equal(t, yandexUID, *u.Identities[0].ProviderUserID)
						return nil
					})
				m.session.EXPECT().Create(gomock.Any(), gomock.Any()).Return(refresh, nil)
			},
			want: refresh,
		},
		{
			name:  "email already used by another account",
			input: input,
			setup: func(m authMocks) {
				m.providers.EXPECT().Use(input.ProviderName).Return(m.provider, nil)
				m.provider.EXPECT().Exchange(gomock.Any(), input.Code, input.Verifier).Return(identity, nil)
				m.repo.EXPECT().FindByIdentity(gomock.Any(), domain.ProviderYandex, yandexUID).Return(nil, domain.ErrNotFound)
				m.repo.EXPECT().FindByEmail(gomock.Any(), email).Return(existing, nil)
			},
			wantErr: domain.ErrAlreadyExists,
		},
		{
			name:  "provider did not return email",
			input: input,
			setup: func(m authMocks) {
				m.providers.EXPECT().Use(input.ProviderName).Return(m.provider, nil)
				m.provider.EXPECT().
					Exchange(gomock.Any(), input.Code, input.Verifier).
					Return(dto.OAuthIdentity{ProviderUserID: yandexUID}, nil)
				m.repo.EXPECT().FindByIdentity(gomock.Any(), domain.ProviderYandex, yandexUID).Return(nil, domain.ErrNotFound)
			},
			wantValidationErr: true,
		},
		{
			name:  "provider exchange fails",
			input: input,
			setup: func(m authMocks) {
				m.providers.EXPECT().Use(input.ProviderName).Return(m.provider, nil)
				m.provider.EXPECT().
					Exchange(gomock.Any(), input.Code, input.Verifier).
					Return(dto.OAuthIdentity{}, domain.ErrProviderUnavailable)
			},
			wantErr: domain.ErrProviderUnavailable,
		},
		{
			name:  "repository error on identity lookup is propagated",
			input: input,
			setup: func(m authMocks) {
				m.providers.EXPECT().Use(input.ProviderName).Return(m.provider, nil)
				m.provider.EXPECT().Exchange(gomock.Any(), input.Code, input.Verifier).Return(identity, nil)
				m.repo.EXPECT().FindByIdentity(gomock.Any(), domain.ProviderYandex, yandexUID).Return(nil, repoErr)
			},
			wantErr: repoErr,
		},
		{
			name:              "unknown provider",
			input:             dto.OAuthCallbackInput{ProviderName: "vk", Code: "code", Verifier: "verifier"},
			setup:             func(authMocks) {},
			wantValidationErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc, m := newAuthService(t)
			tt.setup(m)

			got, err := svc.OauthCallback(context.Background(), tt.input)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Equal(t, dto.Token{}, got)
				return
			}

			if tt.wantValidationErr {
				_, isValidation := errors.AsType[domain.ValidationErr](err)
				assert.True(t, isValidation, "expected ValidationErr, got: %v", err)
				assert.Equal(t, dto.Token{}, got)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAuthService_OauthSignIn(t *testing.T) {
	t.Parallel()

	svc, m := newAuthService(t)
	m.providers.EXPECT().Use("yandex").Return(m.provider, nil)
	m.provider.EXPECT().
		AuthCodeURL("state", "verifier").
		Return("https://oauth.example/authorize?state=state")
	m.oauth.EXPECT().
		Generate().
		Return(dto.OAuthTokenPair{State: "state", Verifier: "verifier", TTL: time.Minute})

	got, err := svc.OauthSignIn("yandex")

	assert.NoError(t, err)
	assert.Equal(t, dto.OAuthOutput{
		State:    "state",
		Verifier: "verifier",
		TTL:      time.Minute,
		URL:      "https://oauth.example/authorize?state=state",
	}, got)
}
