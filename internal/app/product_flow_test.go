//go:build integration

package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"smart-fridge-backend/internal/app"
	"smart-fridge-backend/internal/config"
	db "smart-fridge-backend/internal/repository/postgres"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestApplicationProductFlow(t *testing.T) {
	ctx := t.Context()
	pg, err := postgres.Run(ctx, "postgres:18-alpine", postgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = pg.Terminate(cleanup)
	})
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, db.Migrate(ctx, pool))

	var mu sync.Mutex
	objects := map[string][]byte{}
	storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPut:
			data, e := io.ReadAll(r.Body)
			if e != nil {
				w.WriteHeader(400)
				return
			}
			objects[r.URL.Path] = data
		case http.MethodHead:
			if _, ok := objects[r.URL.Path]; !ok {
				w.WriteHeader(404)
			}
		case http.MethodGet:
			data, ok := objects[r.URL.Path]
			if !ok {
				w.WriteHeader(404)
				return
			}
			_, _ = w.Write(data)
		case http.MethodDelete:
			delete(objects, r.URL.Path)
			w.WriteHeader(204)
		default:
			w.WriteHeader(405)
		}
	}))
	t.Cleanup(storage.Close)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())
	for key, value := range map[string]string{
		"SMART_FRIDGE_HTTP_ADDR":                 addr,
		"SMART_FRIDGE_HTTP_SWAGGER_ENABLED":      "true",
		"SMART_FRIDGE_DATABASE_URL":              dsn,
		"SMART_FRIDGE_AUTH_JWT_SECRET":           "test-jwt-secret-at-least-thirty-two-bytes",
		"SMART_FRIDGE_STORAGE_ENDPOINT":          storage.URL,
		"SMART_FRIDGE_STORAGE_REGION":            "ru-central1",
		"SMART_FRIDGE_STORAGE_BUCKET":            "test-photos",
		"SMART_FRIDGE_STORAGE_ACCESS_KEY_ID":     "test-access",
		"SMART_FRIDGE_STORAGE_SECRET_ACCESS_KEY": "test-secret",
	} {
		t.Setenv(key, value)
	}
	cfg, err := config.Load()
	require.NoError(t, err)
	application := app.New(cfg)
	require.NoError(t, application.Err())
	require.NoError(t, application.Start(ctx))
	t.Cleanup(func() {
		stop, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		require.NoError(t, application.Stop(stop))
	})
	client := &http.Client{Timeout: 10 * time.Second}
	base := "http://" + addr + "/api/v1"
	token := ""
	request := func(method, path string, body any) map[string]any {
		t.Helper()
		var data io.Reader
		if body != nil {
			raw, e := json.Marshal(body)
			require.NoError(t, e)
			data = bytes.NewReader(raw)
		}
		req, e := http.NewRequestWithContext(ctx, method, base+path, data)
		require.NoError(t, e)
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, e := client.Do(req)
		require.NoError(t, e)
		defer resp.Body.Close()
		raw, e := io.ReadAll(resp.Body)
		require.NoError(t, e)
		require.Less(t, resp.StatusCode, 300, string(raw))
		result := map[string]any{}
		require.NoError(t, json.Unmarshal(raw, &result))
		return result
	}
	auth := request("POST", "/auth/sign-up", map[string]string{"email": "flow@example.com", "password": "12345678"})
	token = auth["accessToken"].(string)
	categories := request("GET", "/categories", nil)["items"].([]any)
	require.NotEmpty(t, categories)
	category := categories[0].(map[string]any)["id"]
	id := uuid.NewString()
	batch := func(kind string, version int, fields map[string]any) map[string]any {
		t.Helper()
		operation := map[string]any{"operationId": uuid.NewString(), "id": id, "type": kind}
		if version > 0 {
			operation["expectedVersion"] = version
		}
		if fields != nil {
			operation["product"] = fields
		}
		result := request("POST", "/products/batch", map[string]any{"operations": []any{operation}})
		results := result["results"].([]any)
		require.Len(t, results, 1)
		return results[0].(map[string]any)
	}
	fields := map[string]any{"name": "Milk", "categoryId": category, "expiresAt": "2027-01-01", "imageKey": nil}
	created := batch("create", 0, fields)
	require.Nil(t, created["error"])
	require.EqualValues(t, 1, created["product"].(map[string]any)["version"])
	prepared := request("POST", "/products/images/batch", map[string]any{"operations": []any{map[string]any{"operationId": uuid.NewString(), "contentType": "image/png"}}})
	target := prepared["uploads"].([]any)[0].(map[string]any)
	upload, err := http.NewRequestWithContext(ctx, "PUT", target["uploadUrl"].(string), bytes.NewReader([]byte("test image bytes")))
	require.NoError(t, err)
	for key, value := range target["requiredHeaders"].(map[string]any) {
		upload.Header.Set(key, value.(string))
	}
	response, err := client.Do(upload)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, 200, response.StatusCode)
	fields["imageKey"] = target["imageKey"]
	fields["name"] = "Fresh milk"
	updated := batch("update", 1, fields)
	require.Nil(t, updated["error"])
	saved := updated["product"].(map[string]any)
	require.EqualValues(t, 2, saved["version"])
	require.Contains(t, saved["imageUrl"], "X-Amz-Signature=")
	conflict := batch("update", 1, fields)
	require.Equal(t, "version_conflict", conflict["error"].(map[string]any)["code"])
	require.EqualValues(t, 2, conflict["product"].(map[string]any)["version"])
	products := request("GET", "/products", nil)["items"].([]any)
	require.Len(t, products, 1)
	require.NotEmpty(t, products[0].(map[string]any)["imageUrl"])
	deleted := batch("delete", 2, nil)
	require.Nil(t, deleted["error"])
	require.Empty(t, request("GET", "/products", nil)["items"])
	missing := batch("update", 2, fields)
	require.Equal(t, "not_found", missing["error"].(map[string]any)["code"])
	mu.Lock()
	remaining := len(objects)
	mu.Unlock()
	require.Zero(t, remaining)
	changedSettings := request("PATCH", "/settings", map[string]any{"notificationTime": "10:30:00", "hasNotifications": false})
	require.Equal(t, "10:30:00", changedSettings["notificationTime"])
	require.Equal(t, false, changedSettings["hasNotifications"])
	require.Equal(t, "10:30:00", request("GET", "/settings", nil)["notificationTime"])

	indicators := []map[string]any{{"color": "#FF0000", "maxDays": 1}, {"color": "#00FF00", "maxDays": nil}}
	settingsWithIndicators := request("PATCH", "/settings", map[string]any{"indicators": indicators})
	require.Len(t, settingsWithIndicators["indicators"].([]any), 2)
	fields["imageKey"] = nil
	fields["categoryId"] = 99999999
	invalidCategory := batch("create", 0, fields)
	require.Equal(t, "validation_error", invalidCategory["error"].(map[string]any)["code"])
	fields["categoryId"] = category
	badFields := map[string]any{"name": "Invalid dates", "categoryId": category, "imageKey": nil, "expiresAt": "2027-01-01", "manufacturedAt": "2027-02-01"}
	validID := uuid.NewString()
	mixed := request("POST", "/products/batch", map[string]any{"operations": []any{
		map[string]any{"operationId": uuid.NewString(), "id": uuid.NewString(), "type": "create", "product": badFields},
		map[string]any{"operationId": uuid.NewString(), "id": validID, "type": "create", "product": fields},
	}})
	results := mixed["results"].([]any)
	require.Len(t, results, 2)
	require.Equal(t, "validation_error", results[0].(map[string]any)["error"].(map[string]any)["code"])
	require.Equal(t, validID, results[1].(map[string]any)["product"].(map[string]any)["id"])
	unauthorizedRequest, err := http.NewRequestWithContext(ctx, "GET", base+"/products", nil)
	require.NoError(t, err)
	unauthorizedResponse, err := client.Do(unauthorizedRequest)
	require.NoError(t, err)
	require.NoError(t, unauthorizedResponse.Body.Close())
	require.Equal(t, http.StatusUnauthorized, unauthorizedResponse.StatusCode)
	fmt.Println("HTTP flow passed: auth, categories, direct S3 upload, product batch, conflict, imageUrl, delete, settings, Fx shutdown")
}
