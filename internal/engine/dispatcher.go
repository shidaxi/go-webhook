package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/shidaxi/go-webhook/internal/config"
)

// Dispatch sends an HTTP request with the given body to the target URL.
// It retries on 5xx and network errors with exponential backoff.
func Dispatch(ctx context.Context, targetURL, method string, body map[string]any, headers map[string]string, timeout time.Duration, maxRetries int) config.DispatchResult {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return config.DispatchResult{
			TargetURL: targetURL,
			Error:     fmt.Errorf("failed to marshal body: %w", err),
		}
	}

	client := &http.Client{Timeout: timeout}

	var lastResult config.DispatchResult
	for attempt := range maxRetries {
		lastResult = doRequest(ctx, client, targetURL, method, jsonBody, headers)

		if lastResult.Success {
			return lastResult
		}

		// Don't retry on 4xx (client errors) — only retry on 5xx or network errors
		if lastResult.StatusCode >= 400 && lastResult.StatusCode < 500 {
			return lastResult
		}

		// Don't retry if context is canceled
		if ctx.Err() != nil {
			return lastResult
		}

		// Exponential backoff before next retry (skip sleep on last attempt)
		if attempt < maxRetries-1 {
			backoff := time.Duration(1<<uint(attempt)) * 100 * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				lastResult.Error = ctx.Err()
				return lastResult
			}
		}
	}

	return lastResult
}

func doRequest(ctx context.Context, client *http.Client, targetURL, method string, jsonBody []byte, headers map[string]string) config.DispatchResult {
	req, err := http.NewRequestWithContext(ctx, method, targetURL, bytes.NewReader(jsonBody))
	if err != nil {
		return config.DispatchResult{
			TargetURL: targetURL,
			Error:     fmt.Errorf("failed to create request: %w", err),
		}
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return config.DispatchResult{
			TargetURL: targetURL,
			Error:     fmt.Errorf("request failed: %w", err),
		}
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	return config.DispatchResult{
		TargetURL:  targetURL,
		StatusCode: resp.StatusCode,
		Success:    resp.StatusCode >= 200 && resp.StatusCode < 300,
	}
}
