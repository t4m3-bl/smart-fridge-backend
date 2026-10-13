package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrAlreadyExists      = errors.New("already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidSession     = errors.New("invalid or expired session")
)

type ValidationErr struct {
	field  string
	reason string
}

func NewValidationErr(field, reason string) ValidationErr {
	return ValidationErr{
		field:  field,
		reason: reason,
	}
}

func (e ValidationErr) Error() string {
	return fmt.Sprintf("%s validation error: %s", e.field, e.reason)
}
