package cookie

import (
	"net/http"
	"smart-fridge-backend/internal/dto"
)

const (
	path = "/api/v1/auth"
	name = "refresh_token"
)

type Factory struct {
	secure bool
}

func NewFactory(secure bool) *Factory {
	return &Factory{secure: secure}
}

// buildRefreshTokenCookie returns the base refresh-token cookie with all shared
// attributes set. Callers only vary Value and MaxAge.
func (f *Factory) buildRefreshTokenCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   f.secure,
		SameSite: http.SameSiteStrictMode,
	}
}

// CreateRefreshTokenCookie builds a serialized Set-Cookie value containing the
// given refresh token. The cookie is scoped to path, HttpOnly, Secure, and
// SameSite=Strict, with Max-Age derived from ttl in whole seconds.
func (f *Factory) CreateRefreshTokenCookie(refresh dto.Token) *string {
	ck := f.buildRefreshTokenCookie(refresh.Body, int(refresh.TTL.Seconds()))
	return new(ck.String())
}

// CreateEmptyRefreshTokenCookie builds a serialized Set-Cookie value that clears
// the refresh-token cookie. Same attributes as CreateRefreshTokenCookie, but
// with an empty value and Max-Age=0.
func (f *Factory) CreateEmptyRefreshTokenCookie() *string {
	ck := f.buildRefreshTokenCookie("", 0)
	return new(ck.String())
}
