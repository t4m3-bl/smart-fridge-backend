package domain

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type Identity struct {
	IdentityID     IdentityID
	UserID         UserID
	Provider       Provider
	ProviderUserID *string
	PasswordHash   *string
}

func NewExternalIdentity(userId UserID, provider Provider, providerUserId string) (*Identity, error) {
	id := uuid.Must(uuid.NewV7())

	if provider == ProviderNative {
		return nil, NewValidationErr("provider", "user with native provider must have password")
	}

	if providerUserId == "" {
		return nil, NewValidationErr("providerUserId", "user with external identity must have uid")
	}

	return &Identity{
		IdentityID:     IdentityID(id),
		UserID:         userId,
		Provider:       provider,
		ProviderUserID: &providerUserId,
	}, nil
}

func NewNativeIdentity(userId UserID, passwordHash string) (*Identity, error) {
	id := uuid.Must(uuid.NewV7())

	if passwordHash == "" {
		return nil, NewValidationErr("passwordHash", "user with native identity must have password")
	}

	return &Identity{
		IdentityID:   IdentityID(id),
		UserID:       userId,
		Provider:     ProviderNative,
		PasswordHash: &passwordHash,
	}, nil
}

type IdentityID uuid.UUID

type Provider int

const (
	ProviderUnknown Provider = iota
	ProviderNative
	ProviderYandex
)

func ToProvider(name string) (Provider, error) {
	switch name {
	case "native":
		return ProviderNative, nil
	case "yandex":
		return ProviderYandex, nil
	default:
		return ProviderUnknown, NewValidationErr("provider", "unknown provider")
	}
}

func (p Provider) String() string {
	switch p {
	case ProviderNative:
		return "native"
	case ProviderYandex:
		return "yandex"
	default:
		return "unknown"
	}
}

func (p *Provider) UnmarshalText(text []byte) error {
	name := strings.ToLower(strings.TrimSpace(string(text)))
	provider, err := ToProvider(name)
	if err != nil {
		return fmt.Errorf("invalid identity provider: %s", name)
	}

	*p = provider
	return nil
}
