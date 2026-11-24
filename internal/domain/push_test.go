package domain_test

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"smart-fridge-backend/internal/domain"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPushSubscriptionValidation(t *testing.T) {
	t.Parallel()
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	public := base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())
	auth := base64.RawURLEncoding.EncodeToString(make([]byte, 16))
	require.NoError(t, domain.ValidatePushSubscription("https://fcm.googleapis.com/push/abc", public, auth))
	for _, endpoint := range []string{"http://example.com/push", "https://user:pass@example.com/push", "https://example.com:8080/push", "https://example.com/push#fragment", "not-url"} {
		require.Error(t, domain.ValidatePushSubscription(endpoint, public, auth))
	}
	require.Error(t, domain.ValidatePushSubscription("https://example.com", "invalid", auth))
	require.Error(t, domain.ValidatePushSubscription("https://example.com", public, "invalid"))
}

func TestIndicatorThresholds(t *testing.T) {
	t.Parallel()
	// Colors and ordering do not assign fixed meanings; all configured bounds are valid.
	valid := []domain.Indicator{{Color: "#123456", MaxDays: new(int32(2))}, {Color: "#654321", MaxDays: new(int32(1))}, {Color: "#ABCDEF"}, {Color: "#112233", MaxDays: new(int32(5))}}
	changes := domain.SettingsChanges{Indicators: &valid}
	require.NoError(t, changes.Validate())
	duplicate := append([]domain.Indicator(nil), valid...)
	duplicate[1].MaxDays = new(int32(2))
	changes.Indicators = &duplicate
	require.Error(t, changes.Validate())
	unlimited := append([]domain.Indicator(nil), valid...)
	unlimited[0].MaxDays = nil
	changes.Indicators = &unlimited
	require.Error(t, changes.Validate())
	negative := append([]domain.Indicator(nil), valid...)
	negative[0].MaxDays = new(int32(-1))
	changes.Indicators = &negative
	require.Error(t, changes.Validate())
}
