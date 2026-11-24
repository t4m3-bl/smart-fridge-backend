package config_test

import (
	"smart-fridge-backend/internal/config"
	"strings"
	"testing"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/stretchr/testify/require"
)

func TestPushConfig(t *testing.T) {
	t.Parallel()
	require.NoError(t, (config.Push{}).Validate())
	private, public, err := webpush.GenerateVAPIDKeys()
	require.NoError(t, err)
	cfg := config.Push{Enabled: true, Subject: "mailto:admin@example.com", PrivateKey: private, PublicKey: public, RequestTimeout: time.Second}
	require.NoError(t, cfg.Validate())
	cfg.PublicKey = "broken"
	require.Error(t, cfg.Validate())
	cfg.PublicKey = public
	cfg.Subject = "admin@example.com"
	require.Error(t, cfg.Validate())
}

func TestPushEnvironment(t *testing.T) {
	t.Chdir(t.TempDir())
	private, public, err := webpush.GenerateVAPIDKeys()
	require.NoError(t, err)
	t.Setenv("SMART_FRIDGE_AUTH_JWT_SECRET", strings.Repeat("t", 32))
	t.Setenv("SMART_FRIDGE_DATABASE_URL", "postgres://test:test@localhost/test")
	t.Setenv("SMART_FRIDGE_PUSH_ENABLED", "true")
	t.Setenv("SMART_FRIDGE_PUSH_SUBJECT", "mailto:admin@example.com")
	t.Setenv("SMART_FRIDGE_PUSH_PRIVATE_KEY", private)
	t.Setenv("SMART_FRIDGE_PUSH_PUBLIC_KEY", public)
	t.Setenv("SMART_FRIDGE_PUSH_POLL_INTERVAL", "7s")
	cfg, err := config.Load()
	require.NoError(t, err)
	require.True(t, cfg.Push.Enabled)
	require.Equal(t, 7*time.Second, cfg.Push.PollInterval)
	require.Equal(t, public, cfg.Push.PublicKey)
}

func TestWorkerConfigWithoutAPICredentials(t *testing.T) {
	t.Chdir(t.TempDir())
	private, public, err := webpush.GenerateVAPIDKeys()
	require.NoError(t, err)
	t.Setenv("SMART_FRIDGE_DATABASE_URL", "postgres://test:test@localhost/test")
	t.Setenv("SMART_FRIDGE_AUTH_JWT_SECRET", "")
	t.Setenv("SMART_FRIDGE_PUSH_ENABLED", "true")
	t.Setenv("SMART_FRIDGE_PUSH_SUBJECT", "mailto:admin@example.com")
	t.Setenv("SMART_FRIDGE_PUSH_PUBLIC_KEY", public)
	t.Setenv("SMART_FRIDGE_PUSH_PRIVATE_KEY", private)
	cfg, err := config.LoadWorker()
	require.NoError(t, err)
	require.Equal(t, public, cfg.Push.PublicKey)
	t.Setenv("SMART_FRIDGE_PUSH_PRIVATE_KEY", "")
	_, err = config.LoadWorker()
	require.Error(t, err)
	t.Setenv("SMART_FRIDGE_AUTH_JWT_SECRET", strings.Repeat("t", 32))
	_, err = config.Load()
	require.NoError(t, err) // API needs only the public key.
}
