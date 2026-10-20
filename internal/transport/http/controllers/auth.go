package controllers

import (
	"context"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/cookie"
	"smart-fridge-backend/internal/usecase/service"

	"github.com/labstack/echo/v5"
)

type AuthController struct {
	cookies *cookie.Factory

	auth    *service.AuthService
	session *service.SessionService
}

func NewAuthController(
	cookies *cookie.Factory,
	auth *service.AuthService,
	session *service.SessionService,
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
				SetCookie: []string{
					c.cookies.CreateRefreshTokenCookie(pair.Refresh).String(),
				},
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
				SetCookie: []string{
					c.cookies.CreateRefreshTokenCookie(pair.Refresh).String(),
				},
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
				SetCookie: []string{
					c.cookies.CreateRefreshTokenCookie(pair.Refresh).String(),
				},
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
			SetCookie: []string{
				c.cookies.ClearCookie("refresh_token").String(),
			},
		},
	}, nil
}

func (c *AuthController) OauthSignIn(_ context.Context, request api.OauthSignInRequestObject) (api.OauthSignInResponseObject, error) {
	providerName := string(request.Provider)

	output, err := c.auth.OauthSignIn(providerName)
	if err != nil {
		return nil, err
	}

	return api.OauthSignIn302Response{
		Headers: api.OauthSignIn302ResponseHeaders{
			Location: &output.URL,
			SetCookie: []string{
				c.cookies.CreateSecuredCookie("oauth_state", output.State, output.TTL).String(),
				c.cookies.CreateSecuredCookie("oauth_verifier", output.Verifier, output.TTL).String(),
			},
		},
	}, nil
}

func (c *AuthController) OauthCallback(ctx context.Context, request api.OauthCallbackRequestObject) (api.OauthCallbackResponseObject, error) {
	provider := string(request.Provider)
	p := request.Params

	if p.State != p.OauthState {
		return nil, echo.ErrBadRequest
	}

	if p.Error != nil && *p.Error != "" {
		switch *p.Error {
		case "access_denied":
			return nil, echo.ErrForbidden
		case "server_error":
			return nil, echo.ErrBadGateway
		}
	}

	refresh, err := c.auth.OauthCallback(ctx, dto.OAuthCallbackInput{
		ProviderName: provider,
		Code:         *p.Code,
		Verifier:     p.OauthVerifier,
	})
	if err != nil {
		return nil, err
	}

	return api.OauthCallback302Response{
		Headers: api.OauthCallback302ResponseHeaders{
			Location: new("http://localhost:3000/"),
			SetCookie: []string{
				c.cookies.CreateRefreshTokenCookie(refresh).String(),
			},
		},
	}, nil
}
