package controllers

import (
	"context"
	"errors"
	"fmt"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/cookie"
	"smart-fridge-backend/internal/usecase/service"
	"strings"

	"github.com/labstack/echo/v5"
)

type AuthController struct {
	cookies *cookie.Factory

	auth    *service.AuthService
	session *service.SessionService

	frontendURL config.FrontendURL
}

func NewAuthController(
	cookies *cookie.Factory,
	auth *service.AuthService,
	session *service.SessionService,
	frontendURL config.FrontendURL,
) *AuthController {
	return &AuthController{
		cookies:     cookies,
		auth:        auth,
		session:     session,
		frontendURL: frontendURL,
	}
}

// SignUp processes a user registration request via email and password.
func (c *AuthController) SignUp(ctx context.Context, req api.SignUpRequestObject) (api.SignUpResponseObject, error) {
	resp, err := c.authenticate(ctx, req.Body, c.auth.SignUp)
	if err != nil {
		return nil, err
	}

	return api.SignUp201JSONResponse{AuthSuccessJSONResponse: resp}, nil
}

// SignIn handles the user login request using an email and password.
func (c *AuthController) SignIn(ctx context.Context, req api.SignInRequestObject) (api.SignInResponseObject, error) {
	resp, err := c.authenticate(ctx, req.Body, c.auth.SignIn)
	if err != nil {
		return nil, err
	}

	return api.SignIn200JSONResponse{AuthSuccessJSONResponse: resp}, nil
}

// Refresh handles the request to update the session and its associated tokens.
func (c *AuthController) Refresh(ctx context.Context, request api.RefreshRequestObject) (api.RefreshResponseObject, error) {
	oldRefresh := request.Params.RefreshToken

	pair, err := c.session.Rotate(ctx, oldRefresh)
	if err != nil {
		return nil, err
	}

	return api.Refresh200JSONResponse{
		AuthSuccessJSONResponse: c.newAuthSuccessResponse(pair),
	}, nil
}

// Logout handles the request to log out of the system.
func (c *AuthController) Logout(ctx context.Context, request api.LogoutRequestObject) (api.LogoutResponseObject, error) {
	refresh := request.Params.RefreshToken

	if err := c.session.Revoke(ctx, refresh); err != nil {
		return nil, err
	}

	return api.Logout204Response{
		Headers: api.Logout204ResponseHeaders{
			SetCookie: []string{
				c.cookies.ClearCookie(cookie.RefreshTokenCookieName).String(),
			},
		},
	}, nil
}

// OauthSignIn handles an authorization request via a third-party provider and redirects to the corresponding authorization site.
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
				c.cookies.CreateOAuthStateCookie(output.State, output.TTL).String(),
				c.cookies.CreateOAuthVerifierCookie(output.Verifier, output.TTL).String(),
			},
		},
	}, nil
}

// OauthCallback handles the request received from a third-party provider and then redirects the user to the appropriate page.
func (c *AuthController) OauthCallback(ctx context.Context, request api.OauthCallbackRequestObject) (api.OauthCallbackResponseObject, error) {
	p := request.Params

	if p.State != p.OauthState {
		return nil, echo.ErrBadRequest
	}

	if p.Error != nil {
		return c.redirectWithErrorOnCallback(c.frontendPath("/login"), *p.Error), nil
	}

	if p.Code == nil || *p.Code == "" {
		return nil, echo.ErrBadRequest
	}

	refresh, err := c.auth.OauthCallback(ctx, dto.OAuthCallbackInput{
		ProviderName: string(request.Provider),
		Code:         *p.Code,
		Verifier:     p.OauthVerifier,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrAlreadyExists):
			return c.redirectWithErrorOnCallback(c.frontendPath("/login"), "already_exists"), nil
		case errors.Is(err, domain.ErrProviderUnavailable):
			return c.redirectWithErrorOnCallback(c.frontendPath("/login"), "provider_unavailable"), nil
		}

		return nil, err
	}

	return api.OauthCallback302Response{
		Headers: api.OauthCallback302ResponseHeaders{
			Location: new(c.frontendPath("/fridge")),
			SetCookie: []string{
				c.cookies.ClearCookie(cookie.OAuthStateCookieName).String(),
				c.cookies.ClearCookie(cookie.OAuthVerifierCookieName).String(),
				c.cookies.CreateRefreshTokenCookie(refresh.Body, refresh.TTL).String(),
			},
		},
	}, nil
}

// authenticate is a helper method for user authentication.
func (c *AuthController) authenticate(
	ctx context.Context,
	body *api.Credentials,
	call func(context.Context, dto.CredentialsInput) (dto.TokenPair, error),
) (api.AuthSuccessJSONResponse, error) {
	pair, err := call(ctx, dto.CredentialsInput{
		Email:    string(body.Email),
		Password: body.Password,
	})
	if err != nil {
		return api.AuthSuccessJSONResponse{}, err
	}

	return c.newAuthSuccessResponse(pair), nil
}

// newAuthSuccessResponse is a helper method for constructing a response upon successful authentication.
func (c *AuthController) newAuthSuccessResponse(pair dto.TokenPair) api.AuthSuccessJSONResponse {
	return api.AuthSuccessJSONResponse{
		Body: api.TokenResponse{
			AccessToken: pair.Access.Body,
			ExpiresIn:   int(pair.Access.TTL.Seconds()),
			TokenType:   "Bearer",
		},
		Headers: api.AuthSuccessResponseHeaders{
			SetCookie: []string{
				c.cookies.CreateRefreshTokenCookie(pair.Refresh.Body, pair.Refresh.TTL).String(),
			},
		},
	}
}

// redirectWithErrorOnCallback is a helper method for generating a redirect response that includes errors.
func (c *AuthController) redirectWithErrorOnCallback(location, errStr string) api.OauthCallback302Response {
	return api.OauthCallback302Response{
		Headers: api.OauthCallback302ResponseHeaders{
			Location: new(fmt.Sprintf("%s?error=%s", location, errStr)),
			SetCookie: []string{
				c.cookies.ClearCookie(cookie.OAuthStateCookieName).String(),
				c.cookies.ClearCookie(cookie.OAuthVerifierCookieName).String(),
			},
		},
	}
}

func (c *AuthController) frontendPath(path string) string {
	return strings.TrimSuffix(string(c.frontendURL), "/") + path
}
