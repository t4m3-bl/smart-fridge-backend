package domain_test

import (
	"smart-fridge-backend/internal/domain"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEmail(t *testing.T) {
	t.Parallel()

	type args struct {
		raw string
	}
	tests := []struct {
		name   string
		args   args
		want   domain.Email
		errStr string
	}{
		{
			name: "valid email address",
			args: args{raw: "example@test.com"},
			want: "example@test.com",
		},
		{
			name:   "empty string",
			args:   args{raw: ""},
			want:   "",
			errStr: "email validation error: string is empty or longer than 254 characters",
		},
		{
			name:   "string exceeds maximum length",
			args:   args{raw: strings.Repeat("a", 255)},
			want:   "",
			errStr: "email validation error: string is empty or longer than 254 characters",
		},
		{
			name:   "missing dot in domain",
			args:   args{raw: "example@test"},
			want:   "",
			errStr: "email validation error: string is not an email address",
		},
		{
			name:   "missing at symbol",
			args:   args{raw: "example.com"},
			want:   "",
			errStr: "email validation error: string is not an email address",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.NewEmail(tt.args.raw)

			if tt.errStr == "" {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
				return
			}

			assert.ErrorContains(t, err, tt.errStr)
		})
	}
}

func TestNewUser(t *testing.T) {
	t.Parallel()

	type args struct {
		rawEmail string
	}
	tests := []struct {
		name   string
		args   args
		want   domain.Email
		errStr string
	}{
		{
			name: "successful user creation",
			args: args{rawEmail: "Example@Test.com"},
			want: "example@test.com",
		},
		{
			name:   "validation fails on any invalid email",
			args:   args{rawEmail: "invalid-email-format"},
			errStr: "email validation error: string is not an email address",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.NewUser(tt.args.rawEmail)

			if tt.errStr != "" {
				assert.ErrorContains(t, err, tt.errStr)
				assert.Nil(t, got)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got.Email)
			assert.Empty(t, got.Identities)
		})
	}
}

func TestUser_HasIdentity(t *testing.T) {
	t.Parallel()

	user, _ := domain.NewUser("example@test.com")
	user.Identities = append(user.Identities, domain.Identity{
		UserID:         user.ID,
		Provider:       domain.ProviderYandex,
		ProviderUserID: new("provider-user-id"),
	})

	type args struct {
		p domain.Provider
	}
	tests := []struct {
		name string
		user *domain.User
		args args
		want bool
	}{
		{
			name: "identity found",
			user: user,
			args: args{
				p: domain.ProviderYandex,
			},
			want: true,
		},
		{
			name: "identity not found",
			user: user,
			args: args{
				p: domain.ProviderNative,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.user.HasIdentity(tt.args.p))
		})
	}
}

func TestUser_AddNativeIdentity(t *testing.T) {
	t.Parallel()

	type fields struct {
		ID         domain.UserID
		Email      domain.Email
		Identities []domain.Identity
	}
	type args struct {
		passwordHash string
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantIdentities int
		err            error
	}{
		{
			name: "successful add native identity",
			fields: fields{
				Email:      "example@test.com",
				Identities: []domain.Identity{},
			},
			args:           args{passwordHash: "password-hash"},
			wantIdentities: 1,
		},
		{
			name: "successful add native identity next to external one",
			fields: fields{
				Email: "example@test.com",
				Identities: []domain.Identity{
					{Provider: domain.ProviderYandex, ProviderUserID: new("provider-user-id")},
				},
			},
			args:           args{passwordHash: "password-hash"},
			wantIdentities: 2,
		},
		{
			name: "native identity already exists",
			fields: fields{
				Email: "example@test.com",
				Identities: []domain.Identity{
					{Provider: domain.ProviderNative, PasswordHash: new("old-hash")},
				},
			},
			args:           args{passwordHash: "new-hash"},
			err:            domain.ErrAlreadyExists,
			wantIdentities: 1,
		},
		{
			name: "empty password hash",
			fields: fields{
				Email:      "example@test.com",
				Identities: []domain.Identity{},
			},
			args:           args{passwordHash: ""},
			err:            domain.NewValidationErr("passwordHash", "user with native identity must have password"),
			wantIdentities: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u := &domain.User{
				ID:         tt.fields.ID,
				Email:      tt.fields.Email,
				Identities: tt.fields.Identities,
			}
			err := u.AddNativeIdentity(tt.args.passwordHash)

			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			}

			assert.Len(t, u.Identities, tt.wantIdentities)
			if err == nil {
				added := u.Identities[len(u.Identities)-1]
				assert.Equal(t, domain.ProviderNative, added.Provider)
				assert.Equal(t, u.ID, added.UserID)
				assert.NotNil(t, added.PasswordHash)
				assert.Equal(t, tt.args.passwordHash, *added.PasswordHash)
				assert.Nil(t, added.ProviderUserID)
			}
		})
	}
}

func TestUser_AddExternalIdentity(t *testing.T) {
	t.Parallel()

	type fields struct {
		ID         domain.UserID
		Email      domain.Email
		Identities []domain.Identity
	}
	type args struct {
		p              domain.Provider
		providerUserID string
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantIdentities int
		err            error
	}{
		{
			name: "successful add external identity",
			fields: fields{
				Email:      "example@test.com",
				Identities: []domain.Identity{},
			},
			args:           args{p: domain.ProviderYandex, providerUserID: "provider-user-id"},
			wantIdentities: 1,
		},
		{
			name: "successful add external identity next to native one",
			fields: fields{
				Email: "example@test.com",
				Identities: []domain.Identity{
					{Provider: domain.ProviderNative, PasswordHash: new("password-hash")},
				},
			},
			args:           args{p: domain.ProviderYandex, providerUserID: "provider-user-id"},
			wantIdentities: 2,
		},
		{
			name: "identity with this provider already exists",
			fields: fields{
				Email: "example@test.com",
				Identities: []domain.Identity{
					{Provider: domain.ProviderYandex, ProviderUserID: new("old-uid")},
				},
			},
			args:           args{p: domain.ProviderYandex, providerUserID: "new-uid"},
			err:            domain.ErrAlreadyExists,
			wantIdentities: 1,
		},
		{
			name: "native provider is not allowed",
			fields: fields{
				Email:      "example@test.com",
				Identities: []domain.Identity{},
			},
			args:           args{p: domain.ProviderNative, providerUserID: "provider-user-id"},
			err:            domain.NewValidationErr("provider", "user with native provider must have password"),
			wantIdentities: 0,
		},
		{
			name: "empty provider user id",
			fields: fields{
				Email:      "example@test.com",
				Identities: []domain.Identity{},
			},
			args:           args{p: domain.ProviderYandex, providerUserID: ""},
			err:            domain.NewValidationErr("providerUserId", "user with external identity must have uid"),
			wantIdentities: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u := &domain.User{
				ID:         tt.fields.ID,
				Email:      tt.fields.Email,
				Identities: tt.fields.Identities,
			}
			err := u.AddExternalIdentity(tt.args.p, tt.args.providerUserID)

			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			}

			assert.Len(t, u.Identities, tt.wantIdentities)
			if err == nil {
				added := u.Identities[len(u.Identities)-1]
				assert.Equal(t, tt.args.p, added.Provider)
				assert.Equal(t, u.ID, added.UserID)
				assert.NotNil(t, added.ProviderUserID)
				assert.Equal(t, tt.args.providerUserID, *added.ProviderUserID)
				assert.Nil(t, added.PasswordHash)
			}
		})
	}
}

func TestUser_GetPasswordHash(t *testing.T) {
	t.Parallel()

	type fields struct {
		ID         domain.UserID
		Email      domain.Email
		Identities []domain.Identity
	}
	tests := []struct {
		name   string
		fields fields
		want   string
		err    error
	}{
		{
			name: "native identity exists",
			fields: fields{
				Email: "example@test.com",
				Identities: []domain.Identity{
					{
						Provider:     domain.ProviderYandex,
						PasswordHash: new("password-hash"),
					},
					{
						Provider:     domain.ProviderNative,
						PasswordHash: new("password-hash"),
					},
				},
			},
			want: "password-hash",
		},
		{
			name: "native identity does not exist",
			fields: fields{
				Email: "example@test.com",
				Identities: []domain.Identity{
					{
						Provider:     domain.ProviderYandex,
						PasswordHash: new("password-hash"),
					},
				},
			},
			err: domain.ErrUnprocessableEntity,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u := &domain.User{
				ID:         tt.fields.ID,
				Email:      tt.fields.Email,
				Identities: tt.fields.Identities,
			}
			got, err := u.GetPasswordHash()

			if tt.err != nil {
				assert.ErrorIs(t, err, tt.err)
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
