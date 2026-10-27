package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type MutationID uuid.UUID

func (id MutationID) IsZero() bool {
	return uuid.UUID(id) == uuid.Nil
}

type MutationType string

const (
	MutationCreateProduct MutationType = "createProduct"
	MutationUpdateProduct MutationType = "updateProduct"
	MutationDeleteProduct MutationType = "deleteProduct"
)

type Mutation struct {
	ID        MutationID
	Type      MutationType
	ProductID ProductID
	Data      *ProductData
}

func (m *Mutation) Validate() error {
	if uuid.UUID(m.ProductID) == uuid.Nil {
		return NewValidationErr("productId", "required")
	}

	switch m.Type {
	case MutationCreateProduct, MutationUpdateProduct:
		if m.Data == nil {
			return NewValidationErr("data", "required")
		}
		return m.Data.Normalize()
	case MutationDeleteProduct:
		return nil
	default:
		return NewValidationErr("type", "unknown mutation type")
	}
}

type MutationStatus string

const (
	MutationApplied  MutationStatus = "applied"
	MutationRejected MutationStatus = "rejected"
)

type RejectReason string

const (
	RejectValidation    RejectReason = "validation_error"
	RejectNotFound      RejectReason = "not_found"
	RejectAlreadyExists RejectReason = "already_exists"
)

type MutationResult struct {
	MutationID MutationID
	Status     MutationStatus
	Seq        int64
	Reason     RejectReason
	Message    string
}

func RejectReasonOf(err error) (RejectReason, bool) {
	if _, ok := errors.AsType[ValidationError](err); ok {
		return RejectValidation, true
	}

	switch {
	case errors.Is(err, ErrNotFound):
		return RejectNotFound, true
	case errors.Is(err, ErrAlreadyExists):
		return RejectAlreadyExists, true
	default:
		return "", false
	}
}

type AppliedMutation struct {
	Seq       int64
	ID        MutationID
	Type      MutationType
	ProductID ProductID
	Data      *ProductData
	CreatedAt time.Time
}
