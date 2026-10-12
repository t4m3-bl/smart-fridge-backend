package stub

import (
	"context"
	"errors"
	"smart-fridge-backend/internal/domain"
	"time"
)

var ErrNotImplemented = errors.New("repository is not implemented yet")

type UserRepository struct{}

func NewUserRepository() *UserRepository { return &UserRepository{} }

func (UserRepository) Create(context.Context, *domain.User) error {
	return ErrNotImplemented
}

func (UserRepository) FindByEmail(context.Context, string) (*domain.User, error) {
	return nil, ErrNotImplemented
}

func (UserRepository) FindByID(context.Context, domain.ID) (*domain.User, error) {
	return nil, ErrNotImplemented
}

type SessionRepository struct{}

func NewSessionRepository() *SessionRepository { return &SessionRepository{} }

func (SessionRepository) Create(context.Context, *domain.Session) error {
	return ErrNotImplemented
}

func (SessionRepository) Rotate(context.Context, string, string, time.Duration) (*domain.Session, error) {
	return nil, ErrNotImplemented
}

func (SessionRepository) Revoke(context.Context, string) error {
	return ErrNotImplemented
}
