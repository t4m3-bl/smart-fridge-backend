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

func (c *Client) Send()
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

}
