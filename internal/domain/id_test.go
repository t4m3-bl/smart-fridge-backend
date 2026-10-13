package domain_test

import (
	"smart-fridge-backend/internal/domain"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewID(t *testing.T) {
	is := assert.New(t)

	id, err := domain.NewID()

	is.NoError(err)
	is.NotEmpty(id)
}
