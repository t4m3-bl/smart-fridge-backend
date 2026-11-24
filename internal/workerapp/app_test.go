package workerapp

import (
	"smart-fridge-backend/internal/config"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

func TestWorkerDependencyGraph(t *testing.T) {
	t.Parallel()
	require.NoError(t, fx.ValidateApp(options(&config.WorkerConfig{})))
}
