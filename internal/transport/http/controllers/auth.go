package controllers

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"net/url"
	"smart-fridge-backend/internal/domain"
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

	frontendURL string
}

func NewAuthController(
	cookies *cookie.Factory,
	auth *service.AuthService,
	session *service.SessionService,
	frontendURL string,
) *AuthController {
	return &AuthController{
		cookies:     cookies,
		auth:        auth,
		session:     session,
		frontendURL: frontendURL,
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
				c.cookies.CreateOAuthCookie(oauthStateCookie, output.State, output.TTL).String(),
				c.cookies.CreateOAuthCookie(oauthVerifierCookie, output.Verifier, output.TTL).String(),
			},
		},
	}, nil
}

func (c *AuthController) OauthCallback(ctx context.Context, request api.OauthCallbackRequestObject) (api.OauthCallbackResponseObject, error) {
	p := request.Params

	if subtle.ConstantTimeCompare([]byte(p.State), []byte(p.OauthState)) != 1 {
		return nil, echo.ErrBadRequest
	}

	if p.Error != nil && *p.Error != "" {
		if *p.Error == oauthErrAccessDenied {
			return c.oauthRedirect(oauthErrAccessDenied, nil), nil
		}
		return nil, domain.ErrProviderUnavailable
	}

	if p.Code == nil || *p.Code == "" {
		return nil, echo.ErrBadRequest
	}

	refresh, err := c.auth.OauthCallback(ctx, dto.OAuthCallbackInput{
		ProviderName: string(request.Provider),
		Code:         *p.Code,
		Verifier:     p.OauthVerifier,
	})
	switch {
	case errors.Is(err, domain.ErrAlreadyExists):
		return c.oauthRedirect(oauthErrEmailTaken, nil), nil
	case errors.Is(err, domain.ErrEmailRequired):
		return c.oauthRedirect(oauthErrEmailRequired, nil), nil
	case err != nil:
		return nil, err
	}

	return c.oauthRedirect("", c.cookies.CreateRefreshTokenCookie(refresh)), nil
}

const (
	oauthStateCookie    = "oauth_state"
	oauthVerifierCookie = "oauth_verifier"

	oauthErrAccessDenied  = "access_denied"
	oauthErrEmailRequired = "email_required"
	oauthErrEmailTaken    = "email_taken"
)

func (c *AuthController) oauthRedirect(errCode string, refresh *http.Cookie) api.OauthCallback302Response {
	location := c.frontendURL
	if errCode != "" {
		if u, err := url.Parse(c.frontendURL); err == nil {
			q := u.Query()
			q.Set("error", errCode)
			u.RawQuery = q.Encode()
			location = u.String()
		}
	}

	cookies := []string{
		c.cookies.ClearOAuthCookie(oauthStateCookie).String(),
		c.cookies.ClearOAuthCookie(oauthVerifierCookie).String(),
	}
	if refresh != nil {
		cookies = append(cookies, refresh.String())
	}

	return api.OauthCallback302Response{
		Headers: api.OauthCallback302ResponseHeaders{
			Location:  new(location + "/fridge"),
			SetCookie: cookies,
		},
	}
}
