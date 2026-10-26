package domain

import (
	"fmt"
	"net/mail"
	"strings"

	"github.com/google/uuid"
)

type User struct {
	ID         UserID
	Email      Email
	Identities []Identity
}

func NewUser(rawEmail string) (*User, error) {
	id := uuid.Must(uuid.NewV7())

	email, err := NewEmail(rawEmail)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:         UserID(id),
		Email:      email,
		Identities: make([]Identity, 0),
	}, nil
}

func (u *User) HasIdentity(p Provider) bool {
	for _, identity := range u.Identities {
		if p == identity.Provider {
			return true
		}
	}

	return false
}

func (u *User) AddNativeIdentity(passwordHash string) error {
	if u.HasIdentity(ProviderNative) {
		return ErrAlreadyExists
	}

	identity, err := NewNativeIdentity(u.ID, passwordHash)
	if err != nil {
		return err
	}

	u.Identities = append(u.Identities, *identity)
	return nil
}

func (u *User) AddExternalIdentity(p Provider, providerUserID string) error {
	if u.HasIdentity(p) {
		return ErrAlreadyExists
	}

	identity, err := NewExternalIdentity(u.ID, p, providerUserID)
	if err != nil {
		return err
	}

	u.Identities = append(u.Identities, *identity)
	return nil
}

func (u *User) GetPasswordHash() (string, error) {
	for _, identity := range u.Identities {
		if identity.Provider == ProviderNative {
			return *identity.PasswordHash, nil
		}
	}

	return "", ErrUnprocessableEntity
}

type UserID uuid.UUID

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
