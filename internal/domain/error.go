package domain

import (
	"errors"
	"fmt"
)

var (
	ErrImageStorageUnavailable = errors.New("image_storage_unavailable")
	ErrImageStorageFailed      = errors.New("image_storage_failed")
	ErrVersionConflict         = errors.New("version_conflict")
	ErrNotFound                = errors.New("not_found")
	ErrAlreadyExists           = errors.New("already_exists")
	ErrInvalidCredentials      = errors.New("invalid_credentials")
	ErrInvalidSession          = errors.New("invalid_session")
	ErrUnprocessableEntity     = errors.New("unprocessable_entity")
	ErrUnknownProvider         = errors.New("unknown_provider")
	ErrProviderUnavailable     = errors.New("provider_unavailable")
	ErrPreconditionRequired    = errors.New("precondition_required")
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
