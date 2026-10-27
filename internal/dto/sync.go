package dto

import "smart-fridge-backend/internal/domain"

type PullPage struct {
	Mutations []domain.AppliedMutation
	Cursor    int64
	HasMore   bool
}
