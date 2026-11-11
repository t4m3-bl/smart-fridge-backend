//go:build integration

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"smart-fridge-backend/internal/config"
	"smart-fridge-backend/internal/domain"
	db "smart-fridge-backend/internal/repository/postgres"
	"smart-fridge-backend/internal/storage"
	"smart-fridge-backend/internal/transport/http/api"
	"smart-fridge-backend/internal/transport/http/controllers"
	"smart-fridge-backend/internal/usecase/service"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type imageTestVerifier struct{}

func (imageTestVerifier) Verify(token string) (domain.UserID, error) {
	id, err := uuid.Parse(token)
	return domain.UserID(id), err
}

func TestProductImageHTTP(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	container, err := postgres.Run(ctx, "postgres:18-alpine", postgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		_ = container.Terminate(cleanupCtx)
	})
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, db.Migrate(ctx, pool))
	owner, category, product := uuid.New(), uuid.New(), uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO users(id,email) VALUES($1,'photos@example.com')`, owner)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO categories(user_id,id,name) VALUES($1,$2,'Dairy')`, owner, category)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO products(user_id,id,category_id,name,expires_at) VALUES($1,$2,$3,'Milk','2027-01-01')`, owner, product, category)
	require.NoError(t, err)

	var mu sync.Mutex
	objects := make(map[string][]byte)
	var bumpOnUpload atomic.Bool
	fakeS3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPut:
			data, err := io.ReadAll(r.Body)
			if err != nil || r.Header.Get("Content-Type") != "image/png" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if bumpOnUpload.Swap(false) {
				if _, err = pool.Exec(r.Context(), `UPDATE products SET version=version+1 WHERE user_id=$1 AND id=$2`, owner, product); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
			objects[r.URL.Path] = data
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			delete(objects, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			data, ok := objects[r.URL.Path]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = w.Write(data)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(fakeS3.Close)
	cfg := &config.Config{Storage: config.Storage{
		Endpoint: fakeS3.URL, Region: "ru-central1", Bucket: "test-photos", AccessKeyID: "test-access", SecretAccessKey: "test-secret",
		MaxImageSize: 1024, DownloadTTL: 15 * time.Minute,
	}}
	store, err := storage.New(ctx, cfg.Storage)
	require.NoError(t, err)
	products := service.NewProductService(db.NewProductRepository(pool), store, cfg.Storage.DownloadTTL)
	auth, err := NewAuthenticator(imageTestVerifier{})
	require.NoError(t, err)
	srv, err := NewServer(cfg, auth, NewHandler(nil, nil, controllers.NewProductController(products, cfg), nil))
	require.NoError(t, err)
	var img bytes.Buffer
	require.NoError(t, png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	productPath := "/api/v1/products/" + product.String()
	imagePath := productPath + "/image"

	request := func(method, path, version, contentType string, body io.Reader, user uuid.UUID) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequestWithContext(ctx, method, path, body)
		req.Header.Set("Authorization", "Bearer "+user.String())
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if version != "" {
			req.Header.Set("If-Match", version)
		}
		response := httptest.NewRecorder()
		srv.echo.ServeHTTP(response, req)
		return response
	}
	upload := func(version string, data []byte, user uuid.UUID) *httptest.ResponseRecorder {
		t.Helper()
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		part, err := form.CreateFormFile("image", "photo.png")
		require.NoError(t, err)
		_, err = part.Write(data)
		require.NoError(t, err)
		require.NoError(t, form.Close())
		return request(http.MethodPost, imagePath, version, form.FormDataContentType(), &body, user)
	}
	assertStatus := func(want int, response *httptest.ResponseRecorder) {
		t.Helper()
		require.Equal(t, want, response.Code, response.Body.String())
	}
	storedCount := func() int { mu.Lock(); defer mu.Unlock(); return len(objects) }

	assertStatus(http.StatusPreconditionRequired, upload("", img.Bytes(), owner))
	assertStatus(http.StatusBadRequest, upload(`"1"`, []byte("text file"), owner))
	assertStatus(http.StatusRequestEntityTooLarge, upload(`"1"`, make([]byte, 1025), owner))
	assertStatus(http.StatusRequestEntityTooLarge, upload("", make([]byte, 70*1024), owner))
	assertStatus(http.StatusPreconditionFailed, upload(`"1"`, img.Bytes(), uuid.New()))
	require.Zero(t, storedCount())
	result := upload(`"1"`, img.Bytes(), owner)
	assertStatus(http.StatusOK, result)
	require.Equal(t, `"2"`, result.Header().Get("ETag"))
	require.Equal(t, 1, storedCount())
	result = request(http.MethodGet, "/api/v1/products", "", "", nil, owner)
	assertStatus(http.StatusOK, result)
	var list api.ProductList
	require.NoError(t, json.Unmarshal(result.Body.Bytes(), &list))
	require.Len(t, list.Items, 1)
	require.NotNil(t, list.Items[0].ImageLink)
	link, err := url.Parse(*list.Items[0].ImageLink)
	require.NoError(t, err)
	require.NotEmpty(t, link.Query().Get("X-Amz-Signature"))

	bumpOnUpload.Store(true)
	assertStatus(http.StatusPreconditionFailed, upload(`"2"`, img.Bytes(), owner))
	require.Equal(t, 1, storedCount())
	mu.Lock()
	original := append([]byte(nil), objects[link.Path]...)
	mu.Unlock()
	require.Equal(t, img.Bytes(), original)

	replacement := upload(`"3"`, img.Bytes(), owner)
	assertStatus(http.StatusOK, replacement)
	require.Equal(t, `"4"`, replacement.Header().Get("ETag"))
	require.Equal(t, 1, storedCount())
	productBody, err := json.Marshal(map[string]string{"id": product.String(), "name": "Fresh milk", "categoryId": category.String(), "expiresAt": "2027-01-01"})
	require.NoError(t, err)
	result = request(http.MethodPut, productPath, `"4"`, "application/json", bytes.NewReader(productBody), owner)
	assertStatus(http.StatusOK, result)
	require.Equal(t, `"5"`, result.Header().Get("ETag"))
	var updated api.Product
	require.NoError(t, json.Unmarshal(result.Body.Bytes(), &updated))
	require.NotNil(t, updated.ImageLink)
	require.Contains(t, *updated.ImageLink, "X-Amz-Signature=")

	assertStatus(http.StatusPreconditionFailed, request(http.MethodDelete, imagePath, `"4"`, "", nil, owner))
	require.Equal(t, 1, storedCount())
	result = request(http.MethodDelete, imagePath, `"5"`, "", nil, owner)
	assertStatus(http.StatusOK, result)
	require.Equal(t, `"6"`, result.Header().Get("ETag"))
	require.Zero(t, storedCount())
	assertStatus(http.StatusOK, upload(`"6"`, img.Bytes(), owner))
	assertStatus(http.StatusNoContent, request(http.MethodDelete, productPath, `"7"`, "", nil, owner))
	require.Zero(t, storedCount())
}
