package webpush

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/domain"
	"time"

	push "github.com/SherClockHolmes/webpush-go"
)

type Client struct {
	cfg        config.Push
	httpClient *http.Client
}

func New(cfg config.Push) *Client {
	transport := &http.Transport{DialContext: publicDial, TLSHandshakeTimeout: cfg.RequestTimeout}
	return &Client{cfg: cfg, httpClient: &http.Client{
		Timeout: cfg.RequestTimeout, Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

// Resolve and dial the verified IP directly to prevent DNS rebinding and private-network requests.
func publicDial(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, errors.New("push endpoint DNS lookup failed")
	}
	if len(addresses) == 0 {
		return nil, errors.New("push endpoint has no addresses")
	}
	for _, ip := range addresses {
		if !publicAddress(ip) {
			return nil, errors.New("push endpoint resolved to a non-public address")
		}
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return dialer.DialContext(ctx, network, net.JoinHostPort(addresses[0].String(), port))
}

func publicAddress(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, blocked := range []string{"100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "2001:db8::/32", "64:ff9b::/96", "2002::/16"} {
		if netip.MustParsePrefix(blocked).Contains(ip) {
			return false
		}
	}
	return true
}

func (c *Client) Send(ctx context.Context, message domain.PushMessage) (domain.PushResult, error) {
	if err := domain.ValidatePushSubscription(message.Subscription.Endpoint, message.Subscription.P256DH, message.Subscription.Auth); err != nil {
		return domain.PushResult{}, err
	}
	sub := &push.Subscription{Endpoint: message.Subscription.Endpoint, Keys: push.Keys{P256dh: message.Subscription.P256DH, Auth: message.Subscription.Auth}}
	response, err := push.SendNotificationWithContext(ctx, message.Payload, sub, &push.Options{
		Subscriber: c.cfg.Subject, VAPIDPublicKey: c.cfg.PublicKey, VAPIDPrivateKey: c.cfg.PrivateKey,
		HTTPClient: c.httpClient, TTL: max(1, int(message.TTL.Seconds())), Topic: "fridge-daily", Urgency: push.UrgencyNormal,
	})
	if err != nil {
		return domain.PushResult{}, errors.New("web push request failed")
	}
	defer func() { _ = response.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	result := domain.PushResult{Status: response.StatusCode}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return result, fmt.Errorf("push provider returned HTTP %d", response.StatusCode)
	}
	return result, nil
}
