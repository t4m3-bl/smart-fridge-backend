package cookie

import (
	"net/http"
	"time"
)

const (
	RefreshTokenCookieName  = "refresh_token"
	OAuthStateCookieName    = "oauth_state"
	OAuthVerifierCookieName = "oauth_verifier"
)

type Factory struct {
	secure bool
}

func NewFactory(secure bool) *Factory {
	return &Factory{secure: secure}
}

func (f *Factory) CreateRefreshTokenCookie(value string, ttl time.Duration) *http.Cookie {
	// #nosec G124
	return &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    value,
		Path:     "/api/v1/auth/refresh",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   f.secure,
		SameSite: http.SameSiteLaxMode,
	}
}

func (f *Factory) CreateOAuthStateCookie(value string, ttl time.Duration) *http.Cookie {
	// #nosec G124
	return &http.Cookie{
		Name:     OAuthStateCookieName,
		Value:    value,
		Path:     "/api/v1/auth/",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   f.secure,
		SameSite: http.SameSiteLaxMode,
	}
}

func (f *Factory) CreateOAuthVerifierCookie(value string, ttl time.Duration) *http.Cookie {
	// #nosec G124
	return &http.Cookie{
		Name:     OAuthVerifierCookieName,
		Value:    value,
		Path:     "/api/v1/auth/",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   f.secure,
		SameSite: http.SameSiteLaxMode,
	}
}

func (f *Factory) ClearCookie(name string) *http.Cookie {
	// #nosec G124
	return &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/api/v1/auth/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   f.secure,
		SameSite: http.SameSiteLaxMode,
	}
}
