package domain_test

import (
	"errors"
	"fmt"
	"smart-fridge-backend/internal/domain"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMutation_Validate(t *testing.T) {
	t.Parallel()

	productID := domain.ProductID(uuid.New())

	tests := []struct {
		name     string
		mutation domain.Mutation
		errStr   string
	}{
		{
			name:     "create with valid data",
			mutation: domain.Mutation{ProductID: productID, Type: domain.MutationCreateProduct, Data: new(validProductData())},
		},
		{
			name:     "update with valid data",
			mutation: domain.Mutation{ProductID: productID, Type: domain.MutationUpdateProduct, Data: new(validProductData())},
		},
		{
			name:     "delete without data",
			mutation: domain.Mutation{ProductID: productID, Type: domain.MutationDeleteProduct},
		},
		{
			name:     "create without data",
			mutation: domain.Mutation{ProductID: productID, Type: domain.MutationCreateProduct},
			errStr:   "data validation error: required",
		},
		{
			name:     "update with invalid data",
			mutation: domain.Mutation{ProductID: productID, Type: domain.MutationUpdateProduct, Data: &domain.ProductData{}},
			errStr:   "name validation error: string is empty or longer than 100 characters",
		},
		{
			name:     "missing product id",
			mutation: domain.Mutation{Type: domain.MutationDeleteProduct},
			errStr:   "productId validation error: required",
		},
		{
			name:     "unknown type",
			mutation: domain.Mutation{ProductID: productID, Type: "renameProduct"},
			errStr:   "type validation error: unknown mutation type",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.mutation.Validate()

			if tt.errStr == "" {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, tt.errStr)
		})
	}
}

func TestRejectReasonOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		want   domain.RejectReason
		wantOK bool
	}{
		{
			name:   "validation error",
			err:    domain.NewValidationErr("name", "empty"),
			want:   domain.RejectValidation,
			wantOK: true,
		},
		{
			name:   "wrapped not found",
			err:    fmt.Errorf("product: %w", domain.ErrNotFound),
			want:   domain.RejectNotFound,
			wantOK: true,
		},
		{
			name:   "wrapped already exists",
			err:    fmt.Errorf("product: %w", domain.ErrAlreadyExists),
			want:   domain.RejectAlreadyExists,
			wantOK: true,
		},
		{
			name: "internal error",
			err:  errors.New("connection reset"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := domain.RejectReasonOf(tt.err)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}
