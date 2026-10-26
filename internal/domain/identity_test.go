package domain_test

import (
	"smart-fridge-backend/internal/domain"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewExternalIdentity1(t *testing.T) {
	t.Parallel()

	user, _ := domain.NewUser("example@test.com")

	type args struct {
		userId         domain.UserID
		provider       domain.Provider
		providerUserId string
	}
	tests := []struct {
		name   string
		args   args
		want   domain.Identity
		errStr string
	}{
		{
			name: "successful identity creation",
			args: args{
				userId:         user.ID,
				provider:       domain.ProviderYandex,
				providerUserId: "provider-user-id",
			},
			want: domain.Identity{
				UserID:         user.ID,
				Provider:       domain.ProviderYandex,
				ProviderUserID: new("provider-user-id"),
			},
		},
		{
			name: "native provider",
			args: args{
				userId:         user.ID,
				provider:       domain.ProviderNative,
				providerUserId: "provider-user-id",
			},
			errStr: domain.NewValidationErr("provider", "user with native provider must have password").Error(),
		},
		{
			name: "empty provider user id",
			args: args{
				userId:         user.ID,
				provider:       domain.ProviderYandex,
				providerUserId: "",
			},
			errStr: domain.NewValidationErr("providerUserId", "user with external identity must have uid").Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.NewExternalIdentity(tt.args.userId, tt.args.provider, tt.args.providerUserId)

			if tt.errStr != "" {
				assert.Error(t, err)
				assert.ErrorContains(t, err, tt.errStr)
				return
			}

			assert.NoError(t, err)
			compareIdentities(t, *got, tt.want)
		})
	}
}

func TestNewNativeIdentity(t *testing.T) {
	t.Parallel()

	user, _ := domain.NewUser("example@test.com")

	type args struct {
		userId       domain.UserID
		passwordHash string
	}
	tests := []struct {
		name   string
		args   args
		want   domain.Identity
		errStr string
	}{
		{
			name: "successful identity creation",
			args: args{
				userId:       user.ID,
				passwordHash: "password-hash",
			},
			want: domain.Identity{
				UserID:       user.ID,
				Provider:     domain.ProviderNative,
				PasswordHash: new("password-hash"),
			},
		},
		{
			name: "empty password hash",
			args: args{
				userId:       user.ID,
				passwordHash: "",
			},
			errStr: domain.NewValidationErr("passwordHash", "user with native identity must have password").Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.NewNativeIdentity(tt.args.userId, tt.args.passwordHash)

			if tt.errStr != "" {
				assert.Error(t, err)
				assert.ErrorContains(t, err, tt.errStr)
				return
			}

			assert.NoError(t, err)
			compareIdentities(t, *got, tt.want)
		})
	}
}

func compareIdentities(t *testing.T, got, want domain.Identity) {
	t.Helper()

	assert.Equal(t, want.Provider, got.Provider)
	assert.Equal(t, want.UserID, got.UserID)
	assert.Equal(t, want.PasswordHash, got.PasswordHash)
	assert.Equal(t, want.ProviderUserID, got.ProviderUserID)
}

func TestToProvider(t *testing.T) {
	t.Parallel()

	type args struct {
		name string
	}
	tests := []struct {
		name   string
		args   args
		want   domain.Provider
		errStr string
	}{
		{
			name: "native provider",
			args: args{name: "native"},
			want: domain.ProviderNative,
		},
		{
			name: "yandex provider",
			args: args{name: "yandex"},
			want: domain.ProviderYandex,
		},
		{
			name:   "unknown provider",
			args:   args{name: "unknown"},
			errStr: domain.NewValidationErr("provider", "unknown provider").Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ToProvider(tt.args.name)

			if tt.errStr != "" {
				assert.Error(t, err)
				assert.ErrorContains(t, err, tt.errStr)
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProvider_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		p    domain.Provider
		want string
	}{
		{
			name: "native provider",
			p:    domain.ProviderNative,
			want: "native",
		},
		{
			name: "yandex provider",
			p:    domain.ProviderYandex,
			want: "yandex",
		},
		{
			name: "unknown provider",
			p:    domain.ProviderUnknown,
			want: "unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.p.String())
		})
	}
}

func TestProvider_UnmarshalText(t *testing.T) {
	t.Parallel()

	type args struct {
		text []byte
	}
	tests := []struct {
		name   string
		p      domain.Provider
		want   domain.Provider
		args   args
		errStr string
	}{
		{
			name: "native provider",
			args: args{text: []byte("native")},
			want: domain.ProviderNative,
		},
		{
			name: "yandex provider",
			args: args{text: []byte("yandex")},
			want: domain.ProviderYandex,
		},
		{
			name:   "unknown provider",
			args:   args{text: []byte("unknown")},
			errStr: "invalid identity provider: unknown",
		},
		{
			name: "upper case",
			args: args{text: []byte("YANDEX")},
			want: domain.ProviderYandex,
		},
		{
			name: "with spaces",
			args: args{text: []byte("  yandex ")},
			want: domain.ProviderYandex,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.p.UnmarshalText(tt.args.text)

			if tt.errStr != "" {
				assert.Error(t, err)
				assert.ErrorContains(t, err, tt.errStr)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, tt.p)
		})
	}
}
