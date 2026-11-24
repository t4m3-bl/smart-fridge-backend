package webpush

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"net/netip"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/domain"
	"strings"
	"testing"
	"time"

	push "github.com/SherClockHolmes/webpush-go"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSendPush(t *testing.T) {
	t.Parallel()
	secret, public, err := push.GenerateVAPIDKeys()
	require.NoError(t, err)
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	client := New(config.Push{PrivateKey: secret, PublicKey: public, Subject: "mailto:admin@example.com", RequestTimeout: time.Second})
	client.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "60", r.Header.Get("TTL"))
		require.NotEmpty(t, r.Header.Get("Authorization"))
		require.NotEmpty(t, r.Header.Get("Content-Encoding"))
		require.NotNil(t, r.Context())
		return &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{"Retry-After": []string{"120"}}, Body: io.NopCloser(strings.NewReader("limited"))}, nil
	})
	result, err := client.Send(t.Context(), domain.PushMessage{Subscription: domain.PushSubscription{Endpoint: "https://example.com/push", P256DH: base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes()), Auth: base64.RawURLEncoding.EncodeToString(make([]byte, 16))}, Payload: []byte(`{"title":"Test"}`), TTL: time.Minute})
	require.Error(t, err)
	require.Equal(t, 429, result.Status)
}

func TestPrivatePushAddressesBlocked(t *testing.T) {
	t.Parallel()
	for _, address := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "100.64.0.1", "::1", "::ffff:127.0.0.1", "2001:db8::1"} {
		require.False(t, publicAddress(netip.MustParseAddr(address)), address)
	}
	require.True(t, publicAddress(netip.MustParseAddr("8.8.8.8")))
}
