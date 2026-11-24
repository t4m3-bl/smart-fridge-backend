package domain

import (
	"crypto/ecdh"
	"encoding/base64"
	"net/url"
	"time"

	"github.com/google/uuid"
)

type PushSubscription struct {
	ID       uuid.UUID
	Endpoint string
	P256DH   string
	Auth     string
}

func ValidatePushSubscription(endpoint, publicKey, auth string) error {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || (u.Port() != "" && u.Port() != "443") || len(endpoint) > 2048 {
		return NewValidationErr("endpoint", "must be a public HTTPS push endpoint on port 443")
	}
	public, err := base64.RawURLEncoding.DecodeString(publicKey)
	if err != nil {
		return NewValidationErr("p256dh", "invalid base64url")
	}
	if _, err = ecdh.P256().NewPublicKey(public); err != nil {
		return NewValidationErr("p256dh", "invalid P-256 public key")
	}
	secret, err := base64.RawURLEncoding.DecodeString(auth)
	if err != nil || len(secret) != 16 {
		return NewValidationErr("auth", "must be a 16-byte base64url secret")
	}
	return nil
}

type NotificationSchedule struct {
	UserID           UserID
	NotificationTime time.Duration
	Timezone         string
}

type DailySummary struct {
	Payload       []byte
	Subscriptions []PushSubscription
	ExpiresAt     time.Time
}

type PushMessage struct {
	Subscription PushSubscription
	Payload      []byte
	TTL          time.Duration
}

type PushResult struct {
	Status int
}
