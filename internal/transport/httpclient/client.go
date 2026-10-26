package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func GetUserInfo[T any](
	ctx context.Context,
	client *http.Client,
	url, token string,
) (*T, error) {
	// Create request
	req, err := createRequest(ctx, url, token)
	if err != nil {
		return nil, fmt.Errorf("get user info: %w", err)
	}

	// Exec and parsing
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status error: %d %s", resp.StatusCode, resp.Status)
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("json decode failed: %w", err)
	}

	return &result, nil
}

func createRequest(ctx context.Context, url, token string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "OAuth "+token)

	return req, nil
}
