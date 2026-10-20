package cookie

import (
	"net/http"
	"smart-fridge-backend/internal/dto"
	"time"
)

const (
	path = "/api/v1/auth"
)

type Factory struct {
	secure bool
}

func NewFactory(secure bool) *Factory {
	return &Factory{secure: secure}
}

func (f *Factory) CreateSecuredCookie(name, value string, maxAge time.Duration) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   f.secure,
		SameSite: http.SameSiteStrictMode,
	}
}

func (f *Factory) CreateRefreshTokenCookie(refresh dto.Token) *http.Cookie {
	return f.CreateSecuredCookie(
		"refresh_token",
		refresh.Body,
		refresh.TTL,
	)
}

func (f *Factory) ClearCookie(name string) *http.Cookie {
	return f.CreateSecuredCookie(name, "", 0)
}
