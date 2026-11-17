package dto

import "github.com/google/uuid"

type CategorySaveInput struct {
	CategoryID  uuid.UUID
	UserID      uuid.UUID
	BaseVersion int32
	Name        string
}

type CategoryDeleteInput struct {
	CategoryID  uuid.UUID
	UserID      uuid.UUID
	BaseVersion int32
}
