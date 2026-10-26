package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound            = errors.New("not_found")
	ErrAlreadyExists       = errors.New("already_exists")
	ErrInvalidCredentials  = errors.New("invalid_credentials")
	ErrInvalidSession      = errors.New("invalid_session")
	ErrUnprocessableEntity = errors.New("unprocessable_entity")
	ErrUnknownProvider     = errors.New("unknown_provider")
	ErrProviderUnavailable = errors.New("provider_unavailable")
)

type ValidationError struct {
	field  string
	reason string
}

func NewValidationErr(field, reason string) ValidationError {
	return ValidationError{
		field:  field,
		reason: reason,
	}
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s validation error: %s", e.field, e.reason)
}
