package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/shidaxi/go-webhook/internal/config"
	"github.com/shidaxi/go-webhook/internal/metrics"
)

// Dispatch sends an HTTP request with the given body to the target URL.
// It retries on 5xx and network errors with exponential backoff.
func Dispatch(ctx context.Context, targetURL, method string, body map[string]any, headers map[string]string, timeout time.Duration, maxRetries int) (result config.DispatchResult) {
	start := time.Now()
	defer func() {
		if result.RuleName != "" {
			status := "error"
			if result.StatusCode > 0 {
				status = strconv.Itoa(result.StatusCode)
			}
			metrics.DispatchTotal.WithLabelValues(result.RuleName, targetURL, status).Inc()
			metrics.DispatchDuration.WithLabelValues(result.RuleName).Observe(time.Since(start).Seconds())
		}
	}()

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return config.DispatchResult{
			TargetURL: targetURL,
			Error:     fmt.Errorf("failed to marshal body: %w", err),
		}
	}

	client := &http.Client{Timeout: timeout}

	for attempt := range maxRetries {
		result = doRequest(ctx, client, targetURL, method, jsonBody, headers)

		if result.Success {
			return result
		}

		// Don't retry on 4xx (client errors) — only retry on 5xx or network errors
		if result.StatusCode >= 400 && result.StatusCode < 500 {
			return result
		}

		// Don't retry if context is canceled
		if ctx.Err() != nil {
			return result
		}

		// Exponential backoff before next retry (skip sleep on last attempt)
		if attempt < maxRetries-1 {
			backoff := time.Duration(1<<uint(attempt)) * 100 * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				result.Error = ctx.Err()
				return result
			}
		}
	}

	return result
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
