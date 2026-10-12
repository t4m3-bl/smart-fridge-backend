package domain

import "fmt"

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
