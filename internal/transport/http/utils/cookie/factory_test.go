package cookie_test

import (
	"net/http"
	"smart-fridge-backend/internal/transport/http/utils/cookie"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBrowserCookies(t *testing.T) {
	t.Parallel()
	factory := cookie.NewFactory(true)
	for _, c := range []*http.Cookie{factory.CreateRefreshTokenCookie("refresh", time.Hour), factory.CreatePushBindingCookie("binding", time.Hour), factory.ClearCookie(cookie.PushBindingCookieName)} {
		require.Equal(t, "/api/v1/auth/", c.Path)
		require.True(t, c.Secure)
		require.True(t, c.HttpOnly)
	}
	require.Equal(t, "/api/v1/auth/refresh", factory.ClearLegacyRefreshCookie().Path)
}
