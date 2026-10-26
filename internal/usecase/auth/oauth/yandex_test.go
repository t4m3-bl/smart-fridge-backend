package oauth_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"smart-fridge-backend/internal/usecase/auth/oauth"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/yandex"

	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/dto"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestYandexProvider_AuthCodeURL(t *testing.T) {
	t.Parallel()

	const verifier = "test-verifier-0123456789-abcdefghijklmnopqrstuvwxyz"

	tests := []struct {
		name       string
		cfg        config.OAuthProvider
		state      string
		wantParams map[string]string
		wantAbsent []string
	}{
		{
			name: "with scopes and redirect url",
			cfg: config.OAuthProvider{
				ClientID:    "client-id",
				RedirectURL: "https://app.example/callback",
				Scopes:      []string{"login:email", "login:info"},
			},
			state: "state-123",
			wantParams: map[string]string{
				"client_id":             "client-id",
				"response_type":         "code",
				"redirect_uri":          "https://app.example/callback",
				"scope":                 "login:email login:info",
				"state":                 "state-123",
				"code_challenge":        oauth2.S256ChallengeFromVerifier(verifier),
				"code_challenge_method": "S256",
			},
		},
		{
			name: "without scopes",
			cfg: config.OAuthProvider{
				ClientID:    "client-id",
				RedirectURL: "https://app.example/callback",
			},
			state: "another-state",
			wantParams: map[string]string{
				"client_id":             "client-id",
				"response_type":         "code",
				"redirect_uri":          "https://app.example/callback",
				"state":                 "another-state",
				"code_challenge":        oauth2.S256ChallengeFromVerifier(verifier),
				"code_challenge_method": "S256",
			},
			wantAbsent: []string{"scope"},
		},
		{
			name: "without redirect url",
			cfg: config.OAuthProvider{
				ClientID: "client-id",
				Scopes:   []string{"login:info"},
			},
			state: "s",
			wantParams: map[string]string{
				"client_id":             "client-id",
				"response_type":         "code",
				"scope":                 "login:info",
				"state":                 "s",
				"code_challenge":        oauth2.S256ChallengeFromVerifier(verifier),
				"code_challenge_method": "S256",
			},
			wantAbsent: []string{"redirect_uri"},
		},
	}

	wantBase, err := url.Parse(yandex.Endpoint.AuthURL)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := oauth.NewYandexProvider(tt.cfg, &http.Client{})

			got, err := url.Parse(p.AuthCodeURL(tt.state, verifier))
			require.NoError(t, err)

			assert.Equal(t, wantBase.Scheme, got.Scheme)
			assert.Equal(t, wantBase.Host, got.Host)
			assert.Equal(t, wantBase.Path, got.Path)

			q := got.Query()
			for k, want := range tt.wantParams {
				assert.Equal(t, want, q.Get(k), "query param %q", k)
			}
			for _, k := range tt.wantAbsent {
				assert.False(t, q.Has(k), "query param %q must be absent", k)
			}
		})
	}
}

func TestYandexProvider_Exchange(t *testing.T) {
	t.Parallel()

	const (
		code        = "auth-code"
		verifier    = "test-verifier-0123456789-abcdefghijklmnopqrstuvwxyz"
		accessToken = "access-token"
		userInfoURL = "https://login.yandex.ru/info"
	)

	type response struct {
		status int
		body   string
	}

	validToken := response{
		status: http.StatusOK,
		body:   `{"access_token":"` + accessToken + `","token_type":"bearer","expires_in":3600}`,
	}
	validInfo := response{
		status: http.StatusOK,
		body:   `{"id":"123456","default_email":"user@yandex.ru"}`,
	}

	tests := []struct {
		name          string
		canceledCtx   bool
		token         response
		tokenErr      error
		info          response
		infoErr       error
		want          dto.OAuthIdentity
		wantErr       bool
		wantInfoCalls int32
	}{
		{
			name:          "success",
			token:         validToken,
			info:          validInfo,
			want:          dto.OAuthIdentity{ProviderUserID: "123456", Email: "user@yandex.ru"},
			wantInfoCalls: 1,
		},
		{
			name:          "success without email",
			token:         validToken,
			info:          response{status: http.StatusOK, body: `{"id":"123456"}`},
			want:          dto.OAuthIdentity{ProviderUserID: "123456"},
			wantInfoCalls: 1,
		},
		{
			name:    "token endpoint returns oauth error",
			token:   response{status: http.StatusBadRequest, body: `{"error":"invalid_grant"}`},
			wantErr: true,
		},
		{
			name:    "token response without access_token",
			token:   response{status: http.StatusOK, body: `{"token_type":"bearer"}`},
			wantErr: true,
		},
		{
			name:     "token request transport error",
			tokenErr: errors.New("network down"),
			wantErr:  true,
		},
		{
			name:        "canceled context",
			canceledCtx: true,
			token:       validToken,
			info:        validInfo,
			wantErr:     true,
		},
		{
			name:          "user info returns server error",
			token:         validToken,
			info:          response{status: http.StatusInternalServerError, body: `{"error":"boom"}`},
			wantErr:       true,
			wantInfoCalls: 1,
		},
		{
			name:          "user info returns unauthorized",
			token:         validToken,
			info:          response{status: http.StatusUnauthorized, body: `{"error":"unauthorized"}`},
			wantErr:       true,
			wantInfoCalls: 1,
		},
		{
			name:          "user info returns invalid json",
			token:         validToken,
			info:          response{status: http.StatusOK, body: `not-a-json`},
			wantErr:       true,
			wantInfoCalls: 1,
		},
		{
			name:          "user info transport error",
			token:         validToken,
			infoErr:       errors.New("network down"),
			wantErr:       true,
			wantInfoCalls: 1,
		},
		{
			name:          "user info with empty id",
			token:         validToken,
			info:          response{status: http.StatusOK, body: `{"id":"","default_email":"user@yandex.ru"}`},
			wantErr:       true,
			wantInfoCalls: 1,
		},
		{
			name:          "user info without id field",
			token:         validToken,
			info:          response{status: http.StatusOK, body: `{"default_email":"user@yandex.ru"}`},
			wantErr:       true,
			wantInfoCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var infoCalls atomic.Int32

			client := &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					if err := r.Context().Err(); err != nil {
						return nil, err
					}

					switch {
					case r.URL.String() == yandex.Endpoint.TokenURL:
						assert.NoError(t, r.ParseForm())
						assert.Equal(t, code, r.PostForm.Get("code"))
						assert.Equal(t, verifier, r.PostForm.Get("code_verifier"))
						if tt.tokenErr != nil {
							return nil, tt.tokenErr
						}
						return jsonResponse(tt.token.status, tt.token.body), nil

					case strings.HasPrefix(r.URL.String(), userInfoURL):
						infoCalls.Add(1)
						assert.Contains(t, r.Header.Get("Authorization"), accessToken)
						if tt.infoErr != nil {
							return nil, tt.infoErr
						}
						return jsonResponse(tt.info.status, tt.info.body), nil

					default:
						assert.Failf(t, "unexpected request", "url: %s", r.URL)
						return nil, errors.New("unexpected request")
					}
				}),
			}

			p := oauth.NewYandexProvider(config.OAuthProvider{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURL:  "https://app.example/callback",
			}, client)

			ctx := context.Background()
			if tt.canceledCtx {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			got, err := p.Exchange(ctx, code, verifier)

			if tt.wantErr {
				require.ErrorIs(t, err, domain.ErrProviderUnavailable)
				assert.Zero(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			assert.Equal(t, tt.wantInfoCalls, infoCalls.Load())
		})
	}
}
