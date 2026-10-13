package domain_test

import (
	"smart-fridge-backend/internal/domain"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEmail(t *testing.T) {
	type testCase struct {
		name      string
		raw       string
		want      domain.Email
		errString string
	}

	tests := []testCase{
		{
			name:      "valid email address",
			raw:       "example@test.com",
			want:      "example@test.com",
			errString: "",
		},
		{
			name:      "empty string",
			raw:       "",
			errString: "email validation error: string is empty or longer than 254 characters",
		},
		{
			name:      "string exceeds maximum length",
			raw:       strings.Repeat("a", 255),
			errString: "email validation error: string is empty or longer than 254 characters",
		},
		{
			name:      "missing dot in domain",
			raw:       "example@test",
			errString: "email validation error: string is not an email address",
		},
		{
			name:      "missing at symbol",
			raw:       "example.com",
			errString: "email validation error: string is not an email address",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := assert.New(t)

			email, err := domain.NewEmail(tc.raw)
			if tc.errString != "" && is.Error(err) {
				is.Equal(tc.errString, err.Error())
			} else {
				is.NoError(err)
			}

			is.Equal(tc.want, email)
		})
	}
}

func TestNewUser(t *testing.T) {
	type args struct {
		email        string
		passwordHash string
	}

	type testCase struct {
		name      string
		args      args
		want      domain.User
		errString string
	}

	tests := []testCase{
		{
			name: "successful user creation",
			args: args{
				email:        "example@test.com",
				passwordHash: "good_secret",
			},
			want: domain.User{
				Email:        "example@test.com",
				PasswordHash: "good_secret",
			},
			errString: "",
		},
		{
			name: "failed - validation fails on any invalid email",
			args: args{
				email:        "invalid-email-format",
				passwordHash: "good_secret",
			},
			errString: "email validation error: string is not an email address",
		},
		{
			name: "failed - password hash is empty",
			args: args{
				email:        "alex@fridge.com",
				passwordHash: "",
			},
			errString: "password_hash validation error: empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := assert.New(t)

			user, err := domain.NewUser(tc.args.email, tc.args.passwordHash)
			if tc.errString != "" && is.Error(err) {
				is.Equal(tc.errString, err.Error())
				is.Nil(user)
			} else {
				is.NoError(err)
				is.NotEmpty(user.ID)
				is.Equal(tc.want.Email, user.Email)
				is.Equal(tc.want.PasswordHash, user.PasswordHash)
			}
		})
	}
}
