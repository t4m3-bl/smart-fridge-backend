package app

import (
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/usecase/service"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestDependencyGraph(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	require.NoError(t, fx.ValidateApp(fx.Supply(cfg), fx.Supply(cfg.App.FrontendURL), infrastructure, usecases, transport))
}

func TestAPIDoesNotProvideWorker(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	require.Error(t, fx.ValidateApp(fx.Supply(cfg), fx.Supply(cfg.App.FrontendURL), infrastructure, usecases, transport, fx.Invoke(func(*service.NotificationWorker) {})))
}
