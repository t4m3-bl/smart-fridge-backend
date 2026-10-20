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
	type args struct {
		rawEmail string
		pwHash   string
	}
	tests := []struct {
		name   string
		args   args
		want   args
		errStr string
	}{
		{
			name: "successful user creation",
			args: args{
				rawEmail: "example@test.com",
				pwHash:   "good_secret",
			},
			want: args{
				rawEmail: "example@test.com",
				pwHash:   "good_secret",
			},
		},
		{
			name: "validation fails on any invalid email",
			args: args{
				rawEmail: "invalid-email-format",
				pwHash:   "good_secret",
			},
			errStr: "email validation error: string is not an email address",
		},
		{
			name: "password hash is empty",
			args: args{
				rawEmail: "alex@fridge.com",
				pwHash:   "",
			},
			errStr: "password_hash validation error: empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewUser(tt.args.rawEmail, tt.args.pwHash)

			if tt.errStr != "" {
				assert.ErrorContains(t, err, tt.errStr)
				assert.Nil(t, got)
				return
			}

			assert.Equal(t, tt.want, args{
				rawEmail: string(got.Email),
				pwHash:   got.PasswordHash,
			})
		})
	}
}
