package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatch_Success(t *testing.T) {
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		receivedBody = body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	body := map[string]any{"msg_type": "text", "content": "hello"}
	headers := map[string]string{"X-Custom": "test-value"}

	result := Dispatch(context.Background(), server.URL, http.MethodPost, body, headers, 10*time.Second, 3)

	assert.True(t, result.Success)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Nil(t, result.Error)
	assert.Contains(t, string(receivedBody), "msg_type")
}

func TestDispatch_CustomHeaders(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	headers := map[string]string{
		"X-Webhook-Source": "go-webhook",
		"Authorization":    "Bearer token123",
	}

	result := Dispatch(context.Background(), server.URL, http.MethodPost, map[string]any{}, headers, 10*time.Second, 1)
	require.True(t, result.Success)
	assert.Equal(t, "go-webhook", receivedHeaders.Get("X-Webhook-Source"))
	assert.Equal(t, "Bearer token123", receivedHeaders.Get("Authorization"))
}

func TestDispatch_Retry5xx(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := callCount.Add(1)
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	result := Dispatch(context.Background(), server.URL, http.MethodPost, map[string]any{}, nil, 10*time.Second, 3)

	assert.True(t, result.Success)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Equal(t, int32(3), callCount.Load())
}

func TestDispatch_NoRetryOn4xx(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	result := Dispatch(context.Background(), server.URL, http.MethodPost, map[string]any{}, nil, 10*time.Second, 3)

	assert.False(t, result.Success)
	assert.Equal(t, http.StatusBadRequest, result.StatusCode)
	assert.Equal(t, int32(1), callCount.Load(), "should not retry 4xx errors")
}

func TestDispatch_MaxRetriesExhausted(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	result := Dispatch(context.Background(), server.URL, http.MethodPost, map[string]any{}, nil, 10*time.Second, 3)

	assert.False(t, result.Success)
	assert.Equal(t, http.StatusServiceUnavailable, result.StatusCode)
	assert.Equal(t, int32(3), callCount.Load())
}

func TestDispatch_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	result := Dispatch(context.Background(), server.URL, http.MethodPost, map[string]any{}, nil, 100*time.Millisecond, 1)

	assert.False(t, result.Success)
	assert.Error(t, result.Error)
}

func TestDispatch_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	result := Dispatch(ctx, server.URL, http.MethodPost, map[string]any{}, nil, 10*time.Second, 3)

	assert.False(t, result.Success)
	assert.Error(t, result.Error)
}

func TestDispatch_NetworkError(t *testing.T) {
	// Use a server that immediately closes to cause a connection refused error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close()

	result := Dispatch(context.Background(), serverURL, http.MethodPost, map[string]any{}, nil, 1*time.Second, 1)

	assert.False(t, result.Success)
	assert.Error(t, result.Error)
}
