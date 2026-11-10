package http

import (
	"context"
	"fmt"
	"net/http"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/transport/http/middleware"
	"strings"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/google/uuid"
	oapiV5Middleware "github.com/oapi-codegen/echo-v5-middleware"
)

type AccessTokenVerifier interface {
	Verify(token string) (domain.UserID, error)
}

type Authenticator struct {
	verifier AccessTokenVerifier
}

func NewAuthenticator(verifier AccessTokenVerifier) (*Authenticator, error) {
	return &Authenticator{
		verifier: verifier,
	}, nil
}

func (a *Authenticator) Authenticate(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
	eCtx := oapiV5Middleware.GetEchoContext(ctx)
	token, err := extractBearerToken(input.RequestValidationInput.Request)
	if err != nil {
		return fmt.Errorf("authenticator: %w", err)
	}

	userID, err := a.verifier.Verify(token)
	if err != nil {
		return fmt.Errorf("authenticator: %w", err)
	}

	middleware.SetUserID(eCtx, uuid.UUID(userID))
	return nil
}

func extractBearerToken(req *http.Request) (string, error) {
	values := req.Header.Values("Authorization")
	if len(values) != 1 {
		return "", domain.NewValidationErr("authorization", "header is empty")
	}

	parts := strings.Fields(values[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", domain.NewValidationErr("authorization", "header does not contains `Bearer` key-word or values")
	}

	return parts[1], nil
}
