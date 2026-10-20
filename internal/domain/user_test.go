package domain_test

import (
	"smart-fridge-backend/internal/domain"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEmail(t *testing.T) {
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
	tests := []struct {
		name     string
		rawEmail string
		want     domain.Email
		errStr   string
	}{
		{
			name:     "successful user creation",
			rawEmail: "Example@Test.com",
			want:     "example@test.com",
		},
		{
			name:     "validation fails on any invalid email",
			rawEmail: "invalid-email-format",
			errStr:   "email validation error: string is not an email address",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewUser(tt.rawEmail)

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

func TestUser_AddNativeIdentity(t *testing.T) {
	tests := []struct {
		name    string
		hashes  []string
		wantErr error
		errStr  string
	}{
		{
			name:   "successful native identity",
			hashes: []string{"good_secret"},
		},
		{
			name:   "password hash is empty",
			hashes: []string{""},
			errStr: "passwordHash validation error",
		},
		{
			name:    "native identity already exists",
			hashes:  []string{"good_secret", "other_secret"},
			wantErr: domain.ErrAlreadyExists,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := domain.NewUser("alex@fridge.com")
			assert.NoError(t, err)

			for _, hash := range tt.hashes {
				err = user.AddNativeIdentity(hash)
			}

			switch {
			case tt.wantErr != nil:
				assert.ErrorIs(t, err, tt.wantErr)
			case tt.errStr != "":
				assert.ErrorContains(t, err, tt.errStr)
				assert.Empty(t, user.Identities)
			default:
				assert.NoError(t, err)
				hash, err := user.GetPasswordHash()
				assert.NoError(t, err)
				assert.Equal(t, tt.hashes[0], hash)
				assert.Equal(t, user.ID, user.Identities[0].UserID)
			}
		})
	}
}

func TestUser_GetPasswordHash_WithoutNativeIdentity(t *testing.T) {
	user, err := domain.NewUser("alex@fridge.com")
	assert.NoError(t, err)
	assert.NoError(t, user.AddExternalIdentity(domain.ProviderYandex, "yandex-uid"))

	_, err = user.GetPasswordHash()

	assert.ErrorIs(t, err, domain.ErrUnprocessableEntity)
}
