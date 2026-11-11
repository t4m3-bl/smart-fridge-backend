package domain

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const MaxCategoryNameLen = 100

type CategoryID uuid.UUID

type Category struct {
	ID      CategoryID
	UserID  UserID
	Name    string
	Version int32
}

func NewCategory(id CategoryID, userID UserID, version int32, name string) (*Category, error) {
	if uuid.UUID(id) == uuid.Nil {
		return nil, NewValidationErr("id", "category id must not be nil")
	}
	if uuid.UUID(userID) == uuid.Nil {
		return nil, NewValidationErr("userId", "user id must not be nil")
	}
	if version < 0 {
		return nil, NewValidationErr("version", "version must not be negative")
	}

	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n < 1 || n > MaxCategoryNameLen {
		return nil, NewValidationErr("name", fmt.Sprintf("must be 1..%d characters", MaxCategoryNameLen))
	}
	if strings.ContainsRune(name, 0) {
		return nil, NewValidationErr("name", "contains invalid characters")
	}

	return &Category{ID: id, UserID: userID, Name: name, Version: version}, nil
}

func (c *Category) UpdateVersion() { c.Version++ }
