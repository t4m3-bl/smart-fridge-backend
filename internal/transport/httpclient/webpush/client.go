package webpush

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/domain"
	"smart-fridge-backend/internal/usecase/service"
	"strings"
	"time"

	"github.com/SherClockHolmes/webpush-go"
)

func (c *Client) Send(ctx context.Context, subscription domain.PushSubscriptionInput, payload []byte) (service.PushSendResult, error) {
	sub := &webpush.Subscription{
		Endpoint: subscription.Endpoint,
		Keys: webpush.Keys{
			P256dh: subscription.P256DH,
			Auth:   subscription.Auth,
		},
	}

	response, err := webpush.SendNotificationWithContext(ctx, payload, sub, &webpush.Options{
		Subscriber:      c.cfg.Subject,
		VAPIDPublicKey:  c.cfg.PublicKey,
		VAPIDPrivateKey: c.cfg.PrivateKey,
		HTTPClient:      c.httpClient,
	})
	if err != nil {
		return service.PushSendResult{}, fmt.Errorf(
			"web push request failed: %w",
			err,
		)
	}

	if response == nil {
		return service.PushSendResult{}, fmt.Errorf(
			"web push returned empty HTTP response",
		)
	}

	defer func() {
		_ = response.Body.Close()
	}()

	// Drain a small response body to allow connection reuse.
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))

	return service.PushSendResult{
		StatusCode: response.StatusCode,
		RetryAfter: response.Header.Get("Retry-After"),
	}, nil
}

func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(
			_ *http.Request,
			_ []*http.Request,
		) error {
			// Do not follow redirects from untrusted push endpoints.
			return http.ErrUseLastResponse
		},
	}
}
