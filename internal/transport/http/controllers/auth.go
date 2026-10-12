package controllers

import (
	"context"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/cookie"
	"smart-fridge-backend/internal/usecase/service"
)

type AuthController struct {
	cookies cookie.Factory

	auth    service.AuthService
	session service.SessionService
}

func NewAuthController(
	cookies cookie.Factory,
	auth service.AuthService,
	session service.SessionService,
) *AuthController {
	return &AuthController{
		cookies: cookies,
		auth:    auth,
		session: session,
	}
}

func (c *AuthController) SignUp(ctx context.Context, request api.SignUpRequestObject) (api.SignUpResponseObject, error) {
	body := *request.Body
	input := dto.CredentialsInput{
		Email:    string(body.Email),
		Password: body.Password,
	}

	pair, err := c.auth.SignUp(ctx, input)
	if err != nil {
		return nil, err
	}

	return api.SignUp201JSONResponse{
		AuthSuccessJSONResponse: api.AuthSuccessJSONResponse{
			Body: api.TokenResponse{
				AccessToken: pair.Access.Body,
				ExpiresIn:   int(pair.Access.TTL.Seconds()),
				TokenType:   "Bearer",
			},
			Headers: api.AuthSuccessResponseHeaders{
				SetCookie: c.cookies.CreateRefreshTokenCookie(pair.Refresh),
			},
		},
	}, nil
}

func (c *AuthController) SignIn(ctx context.Context, request api.SignInRequestObject) (api.SignInResponseObject, error) {
	body := *request.Body
	input := dto.CredentialsInput{
		Email:    string(body.Email),
		Password: body.Password,
	}

	pair, err := c.auth.SignIn(ctx, input)
	if err != nil {
		return nil, err
	}

	return api.SignIn200JSONResponse{
		AuthSuccessJSONResponse: api.AuthSuccessJSONResponse{
			Body: api.TokenResponse{
				AccessToken: pair.Access.Body,
				ExpiresIn:   int(pair.Access.TTL.Seconds()),
				TokenType:   "Bearer",
			},
			Headers: api.AuthSuccessResponseHeaders{
				SetCookie: c.cookies.CreateRefreshTokenCookie(pair.Refresh),
			},
		},
	}, nil
}

func (c *AuthController) Refresh(ctx context.Context, request api.RefreshRequestObject) (api.RefreshResponseObject, error) {
	oldRefresh := request.Params.RefreshToken

	pair, err := c.session.Rotate(ctx, oldRefresh)
	if err != nil {
		return nil, err
	}

	return api.Refresh200JSONResponse{
		AuthSuccessJSONResponse: api.AuthSuccessJSONResponse{
			Body: api.TokenResponse{
				AccessToken: pair.Access.Body,
				ExpiresIn:   int(pair.Access.TTL.Seconds()),
				TokenType:   "Bearer",
			},
			Headers: api.AuthSuccessResponseHeaders{
				SetCookie: c.cookies.CreateRefreshTokenCookie(pair.Refresh),
			},
		},
	}, nil
}

func (c *AuthController) Logout(ctx context.Context, request api.LogoutRequestObject) (api.LogoutResponseObject, error) {
	refresh := request.Params.RefreshToken

	if err := c.session.Revoke(ctx, refresh); err != nil {
		return nil, err
	}

	return api.Logout204Response{
		Headers: api.Logout204ResponseHeaders{
			SetCookie: c.cookies.CreateEmptyRefreshTokenCookie(),
		},
	}, nil
}
