package api_test

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"smart-fridge-backend/internal/transport/http/api"
	"testing"
)

func TestEmbeddedSpecVersion(t *testing.T) {
	t.Parallel()
	raw, err := api.GetSpecJSON()
	require.NoError(t, err)
	var document map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &document))
	require.JSONEq(t, `"3.1.0"`, string(document["openapi"]))
	spec, err := api.GetSpec()
	require.NoError(t, err)
	require.Equal(t, "3.1.0", spec.OpenAPI)
}
