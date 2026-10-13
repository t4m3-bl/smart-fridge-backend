package domain

import (
	"fmt"
	"net/mail"
	"strings"
)

// User represents an application user in the domain.
//
// Each field is a value object with its own validation rules. Use NewUser to
// create a User with a generated ID and validated email and password hash.
type User struct {
	ID           ID
	Email        Email
	PasswordHash string
}

func NewUser(rawEmail, pwHash string) (*User, error) {
	email, err := NewEmail(rawEmail)
	if err != nil {
		return nil, err
	}

	if pwHash == "" {
		return nil, NewValidationErr("password_hash", "empty")
	}

	id, err := NewID()
	if err != nil {
		return nil, err
	}

	return &User{
		ID:           id,
		Email:        email,
		PasswordHash: pwHash,
	}, nil
}

// Email is a validated, normalized email address.
//
// Email values are trimmed and lowercased. The zero value is invalid; create valid
// values with NewEmail.
type Email string

const MaxEmailLen = 254

func NewEmail(raw string) (Email, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" || len(raw) > MaxEmailLen {
		return "", NewValidationErr("email", fmt.Sprintf("string is empty or longer than %v characters", MaxEmailLen))
	}

	addr, err := mail.ParseAddress(raw)
	if err != nil || addr.Address != raw || !strings.Contains(addr.Address, ".") {
		return "", NewValidationErr("email", "string is not an email address")
	}

	return Email(raw), nil
}
